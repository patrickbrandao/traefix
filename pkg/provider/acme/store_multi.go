package acme

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	"github.com/traefik/traefik/v3/pkg/safe"
)

var _ Store = (*MultiStore)(nil)

// MultiStore reads from a primary store, the source of truth, and copies what it saves to the exports.
type MultiStore struct {
	primary Store
	exports []storeExport
	// legacy is the storage file read once to fill an empty primary store, when moving away from it.
	legacy Store

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

	if err := s.primary.SaveAccount(resolverName, account); err != nil {
		return err
	}

	s.exportAccount(resolverName, account)

	return nil
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

	if err := s.primary.SaveCertificates(resolverName, certificates); err != nil {
		return err
	}

	s.exportCertificates(resolverName, certificates)

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
		if err := export.store.SaveCertificates(resolverName, certificates); err != nil {
			log.Error().Err(err).Str(logs.ProviderName, resolverName+resolverSuffix).Str("export", export.kind).
				Msg("Unable to export the ACME certificates")
		}
	}
}

// StoreBuilder builds the store of each certificate resolver from its configuration.
// The resolvers that name the same storage file or the same directory share one store.
type StoreBuilder struct {
	routinesPool *safe.Pool

	files map[string]*LocalStore
	dirs  map[string]*DirStore
}

// NewStoreBuilder creates a StoreBuilder.
func NewStoreBuilder(routinesPool *safe.Pool) *StoreBuilder {
	return &StoreBuilder{
		routinesPool: routinesPool,
		files:        map[string]*LocalStore{},
		dirs:         map[string]*DirStore{},
	}
}

// Build returns the store of a certificate resolver.
// Without the certStore option it is the storage file alone, as in Traefik.
func (b *StoreBuilder) Build(conf *Configuration) Store {
	if conf.CertStore == nil {
		return b.file(conf.Storage)
	}

	store := &MultiStore{}

	switch conf.CertStore.Primary {
	case StoreDir:
		store.primary = b.dir(conf.CertStore.Dir)

		// The storage file is only opened when it is there: opening it creates it.
		if info, err := os.Stat(conf.Storage); err == nil && info.Size() > 0 {
			store.legacy = b.file(conf.Storage)
		}
	default:
		store.primary = b.file(conf.Storage)
	}

	for _, kind := range conf.CertStore.Export {
		switch kind {
		case StoreDir:
			store.exports = append(store.exports, storeExport{kind: kind, store: b.dir(conf.CertStore.Dir)})
		case StoreFile:
			store.exports = append(store.exports, storeExport{kind: kind, store: b.file(conf.Storage), account: true})
		}
	}

	return store
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
