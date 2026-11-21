package acme

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	"github.com/traefik/traefik/v3/pkg/types"
)

const (
	// redisStoreEvent is the cluster event an instance sends after it stored a certificate.
	redisStoreEvent = "cert"

	// redisStoreCheckInterval is how often an instance compares itself with the server,
	// to catch up on the events it missed.
	redisStoreCheckInterval = 15 * time.Second

	redisStoreTimeout = 5 * time.Second
	redisStoreBatch   = 100
)

var errRedisStoreDown = errors.New("the cluster Redis server is unreachable")

var (
	_ Store        = (*RedisStore)(nil)
	_ storeWatcher = (*RedisStore)(nil)
	_ storeGate    = (*RedisStore)(nil)
	_ storeOfferer = (*RedisStore)(nil)

	_ storeCoordinated = (*RedisStore)(nil)
)

// storeWatcher is a store that other instances write to.
type storeWatcher interface {
	// Watch registers a function called after the certificates of a resolver changed in the store.
	Watch(resolverName string, onChange func())
}

// storeGate is a store that is not always able to take what a resolver would obtain.
type storeGate interface {
	// Ready returns an error when the resolver must not ask the ACME server for anything.
	Ready(resolverName string) error
}

// storeOfferer is a shared store that takes what an instance has locally, when it does not have it or has an older one.
type storeOfferer interface {
	Offer(resolverName string, account *Account, certificates []*CertAndStore)
}

// RedisStore keeps the ACME account and certificates in the Valkey or Redis server of the cluster,
// where every instance reads them. Each instance keeps a copy in memory, which it serves when the server is unreachable.
type RedisStore struct {
	cluster *cluster.Cluster
	now     func() time.Time

	mu        sync.Mutex
	resolvers map[string]*redisResolver
	down      bool

	requests chan string
}

type redisResolver struct {
	// tcertsPrefix is empty when the certificates are not copied for the tools of traefik-certs.
	tcertsPrefix string
	coordinator  *coordinator

	// loaded is false until the certificates were read from the server once.
	loaded  bool
	version string
	account *Account

	certs map[string]*redisCert
	// dirty holds the certificates that the server does not have yet.
	dirty    map[string]struct{}
	watchers []func()
}

type redisCert struct {
	cert     *CertAndStore
	notAfter int64
	// sum identifies the content, to tell a certificate that changed from one that did not.
	sum string
}

// redisCertRecord is the value of a certificate key.
type redisCertRecord struct {
	Domain      types.Domain `json:"domain"`
	Store       string       `json:"store"`
	Certificate string       `json:"certificate"`
	Key         string       `json:"key"`
	NotAfter    int64        `json:"notAfter"`
	Node        string       `json:"node"`
	UpdatedAt   string       `json:"updatedAt"`
}

// NewRedisStore creates a RedisStore on the server of a cluster.
func NewRedisStore(member *cluster.Cluster) *RedisStore {
	s := &RedisStore{
		cluster:   member,
		now:       time.Now,
		resolvers: map[string]*redisResolver{},
		requests:  make(chan string, 64),
	}

	member.Subscribe(redisStoreEvent, func(event cluster.Event) {
		if event.Sender != member.NodeID() && len(event.Args) > 0 {
			s.request(event.Args[0])
		}
	})

	member.OnConnect(func() {
		s.mu.Lock()
		names := slices.Collect(maps.Keys(s.resolvers))
		for _, resolver := range s.resolvers {
			// The events sent while the instance was away are lost: nothing tells that the copy is current.
			resolver.version = ""
		}
		s.mu.Unlock()

		for _, name := range names {
			s.request(name)
		}
	})

	return s
}

// Run keeps the copy of each resolver in line with the server until the context is done.
func (s *RedisStore) Run(ctx context.Context) {
	ticker := time.NewTicker(redisStoreCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case name := <-s.requests:
			s.refresh(name)
		case <-ticker.C:
			s.mu.Lock()
			names := slices.Sorted(maps.Keys(s.resolvers))
			s.mu.Unlock()

			for _, name := range names {
				s.refresh(name)
			}
		case <-ctx.Done():
			return
		}
	}
}

// GetAccount returns the ACME account of a resolver.
func (s *RedisStore) GetAccount(resolverName string) (*Account, error) {
	ctx, cancel := context.WithTimeout(context.Background(), redisStoreTimeout)
	defer cancel()

	// A server known to be away is not asked again here: each attempt waits for the retries of the client.
	var data []byte
	err := errRedisStoreDown
	if !s.isDown() {
		data, err = s.cluster.Client().Get(ctx, s.key(resolverName, "account")).Bytes()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	resolver := s.resolver(resolverName)

	switch {
	case errors.Is(err, redis.Nil):
		return nil, nil
	case err != nil:
		// The account this instance had locally lets it keep its certificates up while the server is away.
		return resolver.account, nil
	}

	account := &Account{}
	//nolint:musttag // Account keeps the untagged encoding it has in the storage file.
	if err := json.Unmarshal(data, account); err != nil {
		return nil, fmt.Errorf("parsing the ACME account of the cluster Redis server: %w", err)
	}

	resolver.account = account

	return account, nil
}

// SaveAccount stores the ACME account of a resolver.
func (s *RedisStore) SaveAccount(resolverName string, account *Account) error {
	//nolint:musttag // Account keeps the untagged encoding it has in the storage file.
	data, err := json.Marshal(account)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.resolver(resolverName).account = account
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), redisStoreTimeout)
	defer cancel()

	return s.cluster.Client().Set(ctx, s.key(resolverName, "account"), data, 0).Err()
}

// GetCertificates returns the certificates of a resolver, from the copy of this instance.
func (s *RedisStore) GetCertificates(resolverName string) ([]*CertAndStore, error) {
	s.mu.Lock()
	loaded := s.resolver(resolverName).loaded
	s.mu.Unlock()

	if !loaded && !s.isDown() {
		// Without the server, the instance starts with what it was offered from its local stores.
		if _, err := s.sync(resolverName); err != nil {
			log.Warn().Err(err).Str(logs.ProviderName, resolverName+resolverSuffix).
				Msg("Unable to read the certificates from the cluster Redis server")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	resolver := s.resolver(resolverName)

	certificates := make([]*CertAndStore, 0, len(resolver.certs))
	for _, id := range slices.Sorted(maps.Keys(resolver.certs)) {
		// The ACME provider changes its certificates in place: it gets its own.
		certificate := *resolver.certs[id].cert
		certificates = append(certificates, &certificate)
	}

	slices.SortStableFunc(certificates, func(a, b *CertAndStore) int {
		return strings.Compare(a.Domain.Main, b.Domain.Main)
	})

	return certificates, nil
}

// SaveCertificates stores the certificates of a resolver that the server does not have, or has an older one of.
// A certificate that cannot be sent stays in the copy of this instance, and is sent when the server is back.
func (s *RedisStore) SaveCertificates(resolverName string, certificates []*CertAndStore) error {
	s.merge(resolverName, certificates, true)

	s.mu.Lock()
	loaded := s.resolver(resolverName).loaded
	s.mu.Unlock()

	if !loaded {
		_, err := s.sync(resolverName)
		return err
	}

	conflict, err := s.flush(resolverName)
	if conflict {
		// The server has a more recent certificate: the instance takes it from there.
		s.request(resolverName)
	}

	return s.result(err)
}

// Offer adds what the instance has locally: the server keeps the most recent of each certificate.
func (s *RedisStore) Offer(resolverName string, account *Account, certificates []*CertAndStore) {
	s.merge(resolverName, certificates, false)

	if _, err := s.sync(resolverName); err != nil {
		log.Warn().Err(err).Str(logs.ProviderName, resolverName+resolverSuffix).
			Msg("Unable to reach the cluster Redis server, the instance starts with its local certificates")
	}

	if account == nil {
		return
	}

	s.mu.Lock()
	resolver := s.resolver(resolverName)
	if resolver.account == nil {
		resolver.account = account
	}
	down := s.down
	s.mu.Unlock()

	if down {
		return
	}

	//nolint:musttag // Account keeps the untagged encoding it has in the storage file.
	data, err := json.Marshal(account)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), redisStoreTimeout)
	defer cancel()

	// The account of the server wins: the instances must share one.
	_ = s.cluster.Client().SetNX(ctx, s.key(resolverName, "account"), data, 0).Err()
}

// Watch registers a function called after the certificates of a resolver changed in the store.
func (s *RedisStore) Watch(resolverName string, onChange func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resolver := s.resolver(resolverName)
	resolver.watchers = append(resolver.watchers, onChange)
}

// Ready returns an error while the certificates were not read from the server, or when the server is unreachable.
// An instance on its own must not obtain certificates: the other instances would obtain them too.
func (s *RedisStore) Ready(resolverName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.down || !s.resolver(resolverName).loaded {
		return errRedisStoreDown
	}

	return nil
}

func (s *RedisStore) isDown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.down
}

// setTcertsPrefix sets the prefix of the copy for the tools of traefik-certs, empty for no copy.
func (s *RedisStore) setTcertsPrefix(resolverName, prefix string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.resolver(resolverName).tcertsPrefix = prefix
}

// setElection makes the instances agree on the one that obtains each certificate of a resolver.
func (s *RedisStore) setElection(resolverName string, step, lockTTL time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.resolver(resolverName).coordinator = newCoordinator(s.cluster, s, resolverName, step, lockTTL)
}

func (s *RedisStore) coordinatorFor(resolverName string) *coordinator {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.resolver(resolverName).coordinator
}

// resolver returns the state of a resolver. The mutex must be held.
func (s *RedisStore) resolver(resolverName string) *redisResolver {
	resolver, ok := s.resolvers[resolverName]
	if !ok {
		resolver = &redisResolver{certs: map[string]*redisCert{}, dirty: map[string]struct{}{}}
		s.resolvers[resolverName] = resolver
	}

	return resolver
}

// key builds a key of a resolver. The braces make a Redis Cluster keep the keys of a resolver on one node,
// which lets one script write a certificate, its index and its version together.
func (s *RedisStore) key(resolverName string, parts ...string) string {
	return s.cluster.Key(append([]string{"acme", "{" + resolverName + "}"}, parts...)...)
}

// request asks Run to compare a resolver with the server. A request that does not fit is caught by the next check.
func (s *RedisStore) request(resolverName string) {
	select {
	case s.requests <- resolverName:
	default:
	}
}

// refresh compares a resolver with the server and tells its watchers when its certificates changed.
// It is never called with a lock of the ACME provider held: the watchers take them.
func (s *RedisStore) refresh(resolverName string) {
	changed, err := s.sync(resolverName)
	if err != nil {
		log.Debug().Err(err).Str(logs.ProviderName, resolverName+resolverSuffix).
			Msg("Unable to compare the certificates with the cluster Redis server")
	}

	if !changed {
		return
	}

	s.mu.Lock()
	watchers := slices.Clone(s.resolver(resolverName).watchers)
	s.mu.Unlock()

	for _, watcher := range watchers {
		watcher()
	}
}

// merge adds certificates to the copy of the instance, and marks them to be sent to the server.
// With replace, a certificate takes the place of the one with the same domains, as after a renewal.
// Without it, only a more recent certificate does.
func (s *RedisStore) merge(resolverName string, certificates []*CertAndStore, replace bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resolver := s.resolver(resolverName)

	for _, certificate := range certificates {
		id, entry, err := newRedisCert(certificate)
		if err != nil {
			log.Error().Err(err).Str(logs.ProviderName, resolverName+resolverSuffix).Strs("domains", certificate.Domain.ToStrArray()).
				Msg("Unable to read a certificate, it is not sent to the cluster Redis server")
			continue
		}

		current, ok := resolver.certs[id]
		switch {
		case ok && current.sum == entry.sum:
			continue
		case ok && !replace && current.notAfter >= entry.notAfter:
			continue
		}

		resolver.certs[id] = entry
		resolver.dirty[id] = struct{}{}
	}
}

// sync reads the certificates of a resolver from the server when they changed there,
// sends the ones the server does not have, and reports whether the copy of the instance changed.
func (s *RedisStore) sync(resolverName string) (bool, error) {
	changed, err := s.load(resolverName)
	if err != nil {
		return changed, s.result(err)
	}

	conflict, err := s.flush(resolverName)
	if err != nil {
		return changed, s.result(err)
	}

	if conflict {
		// The server refused an older certificate: the one it has was written after the read.
		again, err := s.load(resolverName)
		return changed || again, s.result(err)
	}

	return changed, s.result(nil)
}

// result records whether the server answered, which Ready reports.
func (s *RedisStore) result(err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wasDown := s.down
	s.down = err != nil

	switch {
	case err != nil && !wasDown:
		log.Warn().Err(err).Str(logs.ProviderName, "acme").
			Msg("The cluster Redis server is unreachable: the certificates in memory are served, and none is obtained or renewed")
	case err == nil && wasDown:
		log.Info().Str(logs.ProviderName, "acme").Msg("The cluster Redis server is back")
	}

	return err
}

// load reads the certificates of a resolver when their version changed on the server.
func (s *RedisStore) load(resolverName string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), redisStoreTimeout)
	defer cancel()

	client := s.cluster.Client()

	version, err := client.Get(ctx, s.key(resolverName, "version")).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return false, err
	}

	s.mu.Lock()
	resolver := s.resolver(resolverName)
	current := resolver.loaded && resolver.version == version
	s.mu.Unlock()

	if current {
		return false, nil
	}

	ids, err := client.SMembers(ctx, s.key(resolverName, "certs")).Result()
	if err != nil {
		return false, err
	}

	remote := map[string]*redisCert{}
	for batch := range slices.Chunk(ids, redisStoreBatch) {
		keys := make([]string, 0, len(batch))
		for _, id := range batch {
			keys = append(keys, s.key(resolverName, "cert", id))
		}

		values, err := client.MGet(ctx, keys...).Result()
		if err != nil {
			return false, err
		}

		for i, value := range values {
			data, ok := value.(string)
			if !ok {
				continue
			}

			entry, err := parseRedisCert(data)
			if err != nil {
				log.Warn().Err(err).Str(logs.ProviderName, resolverName+resolverSuffix).Str("key", keys[i]).
					Msg("Ignored a certificate of the cluster Redis server that cannot be read")
				continue
			}

			remote[batch[i]] = entry
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	changed := false

	for id, entry := range remote {
		local, ok := resolver.certs[id]
		if ok && local.sum == entry.sum {
			delete(resolver.dirty, id)
			continue
		}

		// A certificate this instance obtained while the server was away is more recent than the one of the server.
		if _, dirty := resolver.dirty[id]; ok && dirty && local.notAfter > entry.notAfter {
			continue
		}

		resolver.certs[id] = entry
		delete(resolver.dirty, id)
		changed = true
	}

	// A server that lost its data gets it back from the instances.
	for id := range resolver.certs {
		if _, ok := remote[id]; !ok {
			resolver.dirty[id] = struct{}{}
		}
	}

	resolver.loaded = true
	resolver.version = version

	return changed, nil
}

// flush sends the certificates the server does not have, and reports whether the server refused one as older than its own.
func (s *RedisStore) flush(resolverName string) (bool, error) {
	s.mu.Lock()
	resolver := s.resolver(resolverName)
	pending := make(map[string]*redisCert, len(resolver.dirty))
	for id := range resolver.dirty {
		pending[id] = resolver.certs[id]
	}
	s.mu.Unlock()

	if len(pending) == 0 {
		return false, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), redisStoreTimeout)
	defer cancel()

	conflict := false
	var written []*redisCert

	for _, id := range slices.Sorted(maps.Keys(pending)) {
		entry := pending[id]

		data, err := json.Marshal(redisCertRecord{
			Domain:      entry.cert.Domain,
			Store:       entry.cert.Store,
			Certificate: string(entry.cert.Certificate.Certificate),
			Key:         string(entry.cert.Key),
			NotAfter:    entry.notAfter,
			Node:        s.cluster.NodeID(),
			UpdatedAt:   s.now().UTC().Format(time.RFC3339),
		})
		if err != nil {
			return conflict, err
		}

		keys := []string{s.key(resolverName, "cert", id), s.key(resolverName, "certs"), s.key(resolverName, "version")}

		version, err := storeCertIfNotOlder.Run(ctx, s.cluster.Client(), keys, data, entry.notAfter, id).Int64()
		if err != nil {
			return conflict, err
		}

		if version == 0 {
			conflict = true
		} else {
			written = append(written, entry)
		}

		s.mu.Lock()
		// The certificate may have been replaced while it was sent: then it stays to be sent.
		if resolver.certs[id] == entry {
			delete(resolver.dirty, id)
		}
		// The copy is only known to be current when this write is the single one since the last read.
		if known, _ := strconv.ParseInt(resolver.version, 10, 64); version != 0 && version == known+1 {
			resolver.version = strconv.FormatInt(version, 10)
		} else {
			resolver.version = ""
		}
		s.mu.Unlock()
	}

	if len(written) == 0 {
		return conflict, nil
	}

	s.exportTcerts(ctx, written)

	if err := s.cluster.Publish(ctx, redisStoreEvent, resolverName); err != nil {
		// The other instances find the certificates at their next check.
		log.Debug().Err(err).Str(logs.ProviderName, resolverName+resolverSuffix).Msg("Unable to announce the certificates to the cluster")
	}

	return conflict, nil
}

// exportTcerts writes the certificates the way the traefik-certs project stores them: one key per domain,
// with the cert.json document, and the domain announced on a channel.
func (s *RedisStore) exportTcerts(ctx context.Context, written []*redisCert) {
	s.mu.Lock()
	certs := map[string][]*CertAndStore{}
	prefixes := map[string]string{}
	for name, resolver := range s.resolvers {
		prefixes[name] = resolver.tcertsPrefix
		for _, entry := range resolver.certs {
			certs[name] = append(certs[name], entry.cert)
		}
	}
	s.mu.Unlock()

	// The names are given among the certificates of every resolver: a domain and its wildcard may want the same one.
	layout := certLayout(certs)

	names := map[string]struct{}{}
	for _, entry := range written {
		for _, domain := range entry.cert.Domain.ToStrArray() {
			name := strings.TrimPrefix(domain, "*.")
			names[name] = struct{}{}
			names[dirStoreWildcardPrefix+name] = struct{}{}
		}
	}

	for _, name := range slices.Sorted(maps.Keys(names)) {
		claim, ok := layout[name]
		if !ok || prefixes[claim.resolverName] == "" {
			continue
		}

		prefix := prefixes[claim.resolverName]

		doc, err := newCertJSON(name, claim.resolverName, claim.cert, s.now())
		if err != nil {
			continue
		}

		data, err := json.Marshal(doc)
		if err != nil {
			continue
		}

		sans := slices.Clone(doc.SANs)
		slices.Sort(sans)

		stored, err := storeTcertsIfNotOlder.Run(ctx, s.cluster.Client(), []string{prefix + ":cert:" + name},
			data, doc.NotAfterUnix, doc.CertMD5, strings.Join(sans, ",")).Int()

		countStoreWrite(claim.resolverName, "tcerts", err)

		if err != nil {
			log.Warn().Err(err).Str(logs.ProviderName, claim.resolverName+resolverSuffix).Str("domain", name).
				Msg("Unable to write the traefik-certs copy of a certificate")
			continue
		}

		if stored == 1 {
			_ = s.cluster.Client().Publish(ctx, prefix+":events", name).Err()
		}
	}
}

// newRedisCert returns the identifier and the copy of a certificate.
func newRedisCert(certificate *CertAndStore) (string, *redisCert, error) {
	_, _, leaf, err := decomposePEM(string(certificate.Certificate.Certificate))
	if err != nil {
		return "", nil, err
	}

	domains := certificate.Domain.ToStrArray()
	if len(domains) == 0 {
		return "", nil, errors.New("certificate without domain")
	}

	sum := sha256.New()
	for _, part := range [][]byte{
		certificate.Certificate.Certificate, certificate.Key, []byte(certificate.Store), []byte(strings.Join(domains, ",")),
	} {
		sum.Write(part)
		sum.Write([]byte{0})
	}

	copied := *certificate

	return certID(domains), &redisCert{cert: &copied, notAfter: leaf.NotAfter.Unix(), sum: hex.EncodeToString(sum.Sum(nil))}, nil
}

func parseRedisCert(data string) (*redisCert, error) {
	var record redisCertRecord
	if err := json.Unmarshal([]byte(data), &record); err != nil {
		return nil, err
	}

	_, entry, err := newRedisCert(&CertAndStore{
		Certificate: Certificate{
			Domain:      record.Domain,
			Certificate: []byte(record.Certificate),
			Key:         []byte(record.Key),
		},
		Store: record.Store,
	})

	return entry, err
}

// certID identifies a certificate by the set of its domains: a renewal keeps it.
func certID(domains []string) string {
	names := make([]string, 0, len(domains))
	for _, domain := range domains {
		names = append(names, strings.ToLower(domain))
	}

	slices.Sort(names)
	names = slices.Compact(names)

	sum := sha256.Sum256([]byte(strings.Join(names, ",")))

	return hex.EncodeToString(sum[:])
}

var (
	// storeCertIfNotOlder writes a certificate and returns the new version of the resolver,
	// or 0 when the server has a more recent certificate for the same domains:
	// two instances that obtained it at the same time end with the same one.
	// KEYS: certificate, index of the resolver, version of the resolver. ARGV: record, notAfter, identifier.
	//nolint:dupword // Lua closes each block with the same word.
	storeCertIfNotOlder = redis.NewScript(`
local current = redis.call("get", KEYS[1])
if current then
	local ok, record = pcall(cjson.decode, current)
	if ok and type(record) == "table" and tonumber(record.notAfter) and tonumber(record.notAfter) > tonumber(ARGV[2]) then
		return 0
	end
end
redis.call("set", KEYS[1], ARGV[1])
redis.call("sadd", KEYS[2], ARGV[3])
return redis.call("incr", KEYS[3])
`)

	// storeTcertsIfNotOlder follows the rule of traefik-certs: a certificate never replaces a more recent one for the same names.
	// KEYS: certificate. ARGV: cert.json, not_after_unix, cert_md5, sorted names joined by commas.
	//nolint:dupword // Lua closes each block with the same word.
	storeTcertsIfNotOlder = redis.NewScript(`
local current = redis.call("get", KEYS[1])
if current then
	local ok, doc = pcall(cjson.decode, current)
	if ok and type(doc) == "table" then
		if doc.cert_md5 == ARGV[3] then
			return 0
		end
		local sans = {}
		if type(doc.sans) == "table" then
			for i, san in ipairs(doc.sans) do
				sans[i] = san
			end
		end
		table.sort(sans)
		if table.concat(sans, ",") == ARGV[4] and tonumber(doc.not_after_unix) and tonumber(doc.not_after_unix) > tonumber(ARGV[2]) then
			return 0
		end
	end
end
redis.call("set", KEYS[1], ARGV[1])
return 1
`)
)
