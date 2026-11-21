package acme

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	"github.com/traefik/traefik/v3/pkg/safe"
)

var (
	_ Store        = (*MultiStore)(nil)
	_ storeWatcher = (*MultiStore)(nil)
	_ storeGate    = (*MultiStore)(nil)

	_ storeCoordinated = (*MultiStore)(nil)
)

// MultiStore reads from a primary store, the source of truth, and copies what it saves to the exports.
type MultiStore struct {
	primary Store
	exports []storeExport
	// legacy is the storage file read once to fill an empty primary store, when moving away from it.
	legacy Store
	// offers are the local stores whose content is offered to a primary store shared by several instances,
	// which keeps the most recent of each certificate.
	offers []Store

	mu       sync.Mutex
	prepared map[string]struct{}
}

type storeExport struct {
	kind  string
	store Store
	// account is false for the stores that only receive certificates.
	account bool
}

// GetAccount returns the ACME account of a resolver.
func (s *MultiStore) GetAccount(resolverName string) (*Account, error) {
	if err := s.prepare(resolverName); err != nil {
		return nil, err
	}

	return s.primary.GetAccount(resolverName)
}

// SaveAccount stores the ACME account of a resolver.
func (s *MultiStore) SaveAccount(resolverName string, account *Account) error {
	if err := s.prepare(resolverName); err != nil {
		return err
	}

	// The exports are written even when the primary store fails:
	// with a shared primary store that is unreachable, they are what the instance starts from.
	err := s.primary.SaveAccount(resolverName, account)

	s.exportAccount(resolverName, account)

	return err
}

// GetCertificates returns the certificates of a resolver.
func (s *MultiStore) GetCertificates(resolverName string) ([]*CertAndStore, error) {
	if err := s.prepare(resolverName); err != nil {
		return nil, err
	}

	return s.primary.GetCertificates(resolverName)
}

// SaveCertificates stores the certificates of a resolver.
func (s *MultiStore) SaveCertificates(resolverName string, certificates []*CertAndStore) error {
	if err := s.prepare(resolverName); err != nil {
		return err
	}

	err := s.primary.SaveCertificates(resolverName, certificates)

	countStoreWrite(resolverName, storeTarget(s.primary), err)

	s.exportCertificates(resolverName, certificates)

	return err
}

// Watch registers a function called after the certificates of a resolver changed in the primary store.
// The exports are brought in line first.
func (s *MultiStore) Watch(resolverName string, onChange func()) {
	watcher, ok := s.primary.(storeWatcher)
	if !ok {
		return
	}

	watcher.Watch(resolverName, func() {
		if account, err := s.primary.GetAccount(resolverName); err == nil && account != nil {
			s.exportAccount(resolverName, account)
		}

		if certificates, err := s.primary.GetCertificates(resolverName); err == nil {
			s.exportCertificates(resolverName, certificates)
		}

		onChange()
	})
}

// Ready returns an error when the primary store cannot take what the resolver would obtain.
func (s *MultiStore) Ready(resolverName string) error {
	if gate, ok := s.primary.(storeGate); ok {
		return gate.Ready(resolverName)
	}

	return nil
}

func (s *MultiStore) coordinatorFor(resolverName string) *coordinator {
	if coordinated, ok := s.primary.(storeCoordinated); ok {
		return coordinated.coordinatorFor(resolverName)
	}

	return nil
}

// prepare runs once for each resolver, before its first read or write:
// it fills an empty primary store from the legacy storage file, then brings the exports in line with the primary.
func (s *MultiStore) prepare(resolverName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.prepared[resolverName]; ok {
		return nil
	}

	if offerer, ok := s.primary.(storeOfferer); ok {
		offerer.Offer(s.readOffers(resolverName))
	}

	account, err := s.primary.GetAccount(resolverName)
	if err != nil {
		return err
	}

	certificates, err := s.primary.GetCertificates(resolverName)
	if err != nil {
		return err
	}

	if s.legacy != nil && account == nil && len(certificates) == 0 {
		account, certificates, err = s.importLegacy(resolverName)
		if err != nil {
			return err
		}
	}

	if account != nil {
		s.exportAccount(resolverName, account)
	}
	s.exportCertificates(resolverName, certificates)

	if s.prepared == nil {
		s.prepared = map[string]struct{}{}
	}
	s.prepared[resolverName] = struct{}{}

	return nil
}

func (s *MultiStore) importLegacy(resolverName string) (*Account, []*CertAndStore, error) {
	// Going on with an empty store would ask the ACME server for every certificate again.
	account, err := s.legacy.GetAccount(resolverName)
	if err != nil {
		return nil, nil, fmt.Errorf("reading the storage file to fill the empty primary certificate store: %w", err)
	}

	certificates, err := s.legacy.GetCertificates(resolverName)
	if err != nil {
		return nil, nil, fmt.Errorf("reading the storage file to fill the empty primary certificate store: %w", err)
	}

	if account == nil && len(certificates) == 0 {
		return nil, nil, nil
	}

	if account != nil {
		if err := s.primary.SaveAccount(resolverName, account); err != nil {
			return nil, nil, err
		}
	}

	if err := s.primary.SaveCertificates(resolverName, certificates); err != nil {
		return nil, nil, err
	}

	log.Info().Str(logs.ProviderName, resolverName+resolverSuffix).
		Int("certificates", len(certificates)).Bool("account", account != nil).
		Msg("Imported the ACME data of the storage file into the primary certificate store")

	return account, certificates, nil
}

// readOffers returns what the local stores hold for a resolver.
// A local store that cannot be read does not stop the resolver: the shared store is the source of truth.
func (s *MultiStore) readOffers(resolverName string) (string, *Account, []*CertAndStore) {
	var account *Account
	var certificates []*CertAndStore

	for _, offer := range s.offers {
		offered, err := offer.GetAccount(resolverName)
		if err != nil {
			log.Warn().Err(err).Str(logs.ProviderName, resolverName+resolverSuffix).Msg("Unable to read the ACME account of a local store")
		}
		if account == nil {
			account = offered
		}

		certs, err := offer.GetCertificates(resolverName)
		if err != nil {
			log.Warn().Err(err).Str(logs.ProviderName, resolverName+resolverSuffix).Msg("Unable to read the certificates of a local store")
		}
		certificates = append(certificates, certs...)
	}

	return resolverName, account, certificates
}

// exportAccount and exportCertificates never fail the caller: the primary store already holds the data.
func (s *MultiStore) exportAccount(resolverName string, account *Account) {
	for _, export := range s.exports {
		if !export.account {
			continue
		}

		if err := export.store.SaveAccount(resolverName, account); err != nil {
			log.Error().Err(err).Str(logs.ProviderName, resolverName+resolverSuffix).Str("export", export.kind).
				Msg("Unable to export the ACME account")
		}
	}
}

func (s *MultiStore) exportCertificates(resolverName string, certificates []*CertAndStore) {
	for _, export := range s.exports {
		err := export.store.SaveCertificates(resolverName, certificates)

		countStoreWrite(resolverName, export.kind, err)

		if err != nil {
			log.Error().Err(err).Str(logs.ProviderName, resolverName+resolverSuffix).Str("export", export.kind).
				Msg("Unable to export the ACME certificates")
		}
	}
}

// StoreBuilder builds the store of each certificate resolver from its configuration.
// The resolvers that name the same storage file or the same directory share one store.
type StoreBuilder struct {
	routinesPool *safe.Pool
	// member is nil when the instance is not in a cluster.
	member *cluster.Cluster

	files map[string]*LocalStore
	dirs  map[string]*DirStore
	redis *RedisStore
}

// NewStoreBuilder creates a StoreBuilder. The cluster is nil when the instance is not in one.
func NewStoreBuilder(routinesPool *safe.Pool, member *cluster.Cluster) *StoreBuilder {
	return &StoreBuilder{
		routinesPool: routinesPool,
		member:       member,
		files:        map[string]*LocalStore{},
		dirs:         map[string]*DirStore{},
	}
}

// Build returns the store of a certificate resolver.
// Without the certStore option it is the storage file alone, as in Traefik.
func (b *StoreBuilder) Build(resolverName string, conf *Configuration) (Store, error) {
	if conf.CertStore == nil {
		return b.file(conf.Storage), nil
	}

	store := &MultiStore{}

	switch conf.CertStore.Primary {
	case StoreRedis:
		if b.member == nil {
			return nil, errors.New("the redis certStore needs the cluster option")
		}

		if b.redis == nil {
			b.redis = NewRedisStore(b.member)
			b.routinesPool.GoCtx(b.redis.Run)
		}

		b.redis.setTcertsPrefix(resolverName, conf.CertStore.tcertsPrefix())
		step, lockTTL := conf.CertStore.election()
		b.redis.setElection(resolverName, step, lockTTL)
		store.primary = b.redis

		// The storage file of an instance that joins the cluster is offered once, as its exports are at each start.
		if info, err := os.Stat(conf.Storage); err == nil && info.Size() > 0 && !slices.Contains(conf.CertStore.Export, StoreFile) {
			store.offers = append(store.offers, b.file(conf.Storage))
		}
	case StoreDir:
		store.primary = b.dir(conf.CertStore.Dir)

		// The storage file is only opened when it is there: opening it creates it.
		if info, err := os.Stat(conf.Storage); err == nil && info.Size() > 0 {
			store.legacy = b.file(conf.Storage)
		}
	default:
		store.primary = b.file(conf.Storage)
	}

	shared := conf.CertStore.Primary == StoreRedis

	for _, kind := range conf.CertStore.Export {
		var export storeExport

		switch kind {
		case StoreDir:
			// With a shared primary store, the directory also keeps the account: an instance starts from its exports alone.
			export = storeExport{kind: kind, store: b.dir(conf.CertStore.Dir), account: shared}
		case StoreFile:
			export = storeExport{kind: kind, store: b.file(conf.Storage), account: true}
		default:
			continue
		}

		store.exports = append(store.exports, export)
		if shared {
			store.offers = append(store.offers, export.store)
		}
	}

	return store, nil
}

// ReportUnmanaged logs what the certificate directories hold that no resolver manages.
func (b *StoreBuilder) ReportUnmanaged(resolverNames []string) {
	for _, path := range slices.Sorted(maps.Keys(b.dirs)) {
		b.dirs[path].ReportUnmanaged(resolverNames)
	}
}

func (b *StoreBuilder) file(filename string) *LocalStore {
	if b.files[filename] == nil {
		b.files[filename] = NewLocalStore(filename, b.routinesPool)
	}

	return b.files[filename]
}

func (b *StoreBuilder) dir(path string) *DirStore {
	if b.dirs[path] == nil {
		b.dirs[path] = NewDirStore(path)
	}

	return b.dirs[path]
}
