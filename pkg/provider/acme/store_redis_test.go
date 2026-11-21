package acme

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/safe"
)

func newStoreTestCluster(t *testing.T, server *miniredis.Miniredis, nodeID string) *cluster.Cluster {
	t.Helper()

	conf := &cluster.Configuration{Redis: &cluster.Redis{}}
	conf.SetDefaults()
	conf.Redis.SetDefaults()
	conf.NodeID = nodeID
	conf.Redis.Endpoints = []string{server.Addr()}

	member, err := cluster.New(t.Context(), conf)
	require.NoError(t, err)
	t.Cleanup(func() { _ = member.Client().Close() })

	return member
}

func newTestRedisStore(t *testing.T, server *miniredis.Miniredis, nodeID string) *RedisStore {
	t.Helper()

	store := NewRedisStore(newStoreTestCluster(t, server, nodeID))
	store.setTcertsPrefix("le", defaultTcertsPrefix)

	return store
}

func TestRedisStore_sharesAccountAndCertificates(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestRedisStore(t, server, "srv1")
	second := newTestRedisStore(t, server, "srv2")

	account, err := first.GetAccount("le")
	require.NoError(t, err)
	assert.Nil(t, account)

	certificates, err := first.GetCertificates("le")
	require.NoError(t, err)
	assert.Empty(t, certificates)
	require.NoError(t, first.Ready("le"))

	account = &Account{Email: "ops@example.com", PrivateKey: []byte("key")}
	cert := newStoreTestCertificate(t, in90Days(), "example.com", "www.example.com")
	other := newStoreTestCertificate(t, in90Days(), "example.org")
	other.Store = "other"

	require.NoError(t, first.SaveAccount("le", account))
	require.NoError(t, first.SaveCertificates("le", []*CertAndStore{cert, other}))

	got, err := second.GetAccount("le")
	require.NoError(t, err)
	assert.Equal(t, account, got)

	certificates, err = second.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{cert, other}, certificates)

	// The keys of a resolver share the braces, which keep them on one node of a Redis Cluster.
	assert.True(t, server.Exists("traefix:acme:{le}:account"))
	assert.True(t, server.Exists("traefix:acme:{le}:cert:"+certID([]string{"www.example.com", "example.com"})))

	members, err := server.SMembers("traefix:acme:{le}:certs")
	require.NoError(t, err)
	assert.Len(t, members, 2)

	// Another resolver has its own certificates.
	certificates, err = second.GetCertificates("zerossl")
	require.NoError(t, err)
	assert.Empty(t, certificates)
}

func TestRedisStore_refresh(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestRedisStore(t, server, "srv1")
	second := newTestRedisStore(t, server, "srv2")

	var changes atomic.Int32
	second.Watch("le", func() { changes.Add(1) })

	_, err := second.GetCertificates("le")
	require.NoError(t, err)

	second.refresh("le")
	assert.Equal(t, int32(0), changes.Load())

	cert := newStoreTestCertificate(t, in90Days(), "example.com")
	require.NoError(t, first.SaveCertificates("le", []*CertAndStore{cert}))

	second.refresh("le")
	assert.Equal(t, int32(1), changes.Load())

	certificates, err := second.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{cert}, certificates)

	// Nothing changed on the server: the watchers are left alone.
	second.refresh("le")
	assert.Equal(t, int32(1), changes.Load())

	renewed := newStoreTestCertificate(t, in90Days().Add(24*time.Hour), "example.com")
	require.NoError(t, first.SaveCertificates("le", []*CertAndStore{renewed}))

	second.refresh("le")
	assert.Equal(t, int32(2), changes.Load())

	certificates, err = second.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{renewed}, certificates)
}

func TestRedisStore_GetCertificates_returnsCopies(t *testing.T) {
	server := miniredis.RunT(t)

	store := newTestRedisStore(t, server, "srv1")

	cert := newStoreTestCertificate(t, in90Days(), "example.com")
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{cert}))

	// The ACME provider changes its certificates in place.
	certificates, err := store.GetCertificates("le")
	require.NoError(t, err)
	certificates[0].Key = []byte("changed")

	certificates, err = store.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, cert.Key, certificates[0].Key)
}

func TestRedisStore_Offer(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestRedisStore(t, server, "srv1")
	second := newTestRedisStore(t, server, "srv2")

	account := &Account{Email: "first@example.com", PrivateKey: []byte("key")}
	old := newStoreTestCertificate(t, in90Days(), "example.com")
	onlyFirst := newStoreTestCertificate(t, in90Days(), "first.example.com")

	// An empty server takes what the first instance has locally.
	first.Offer("le", account, []*CertAndStore{old, onlyFirst})

	certificates, err := first.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{old, onlyFirst}, certificates)

	renewed := newStoreTestCertificate(t, in90Days().Add(24*time.Hour), "example.com")
	onlySecond := newStoreTestCertificate(t, in90Days(), "second.example.com")

	// The second instance has another account, a more recent certificate, and one the server does not have.
	second.Offer("le", &Account{Email: "second@example.com"}, []*CertAndStore{renewed, onlySecond})

	got, err := second.GetAccount("le")
	require.NoError(t, err)
	assert.Equal(t, account, got)

	certificates, err = second.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{renewed, onlyFirst, onlySecond}, certificates)

	// An older certificate never replaces the one of the server.
	third := newTestRedisStore(t, server, "srv3")
	third.Offer("le", nil, []*CertAndStore{old})

	certificates, err = third.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{renewed, onlyFirst, onlySecond}, certificates)

	first.refresh("le")

	certificates, err = first.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{renewed, onlyFirst, onlySecond}, certificates)
}

func TestRedisStore_serverDown(t *testing.T) {
	server := miniredis.RunT(t)

	store := newTestRedisStore(t, server, "srv1")
	server.Close()

	account := &Account{Email: "ops@example.com"}
	cert := newStoreTestCertificate(t, in90Days(), "example.com")

	// The instance starts with what it has locally, and obtains nothing.
	store.Offer("le", account, []*CertAndStore{cert})

	got, err := store.GetAccount("le")
	require.NoError(t, err)
	assert.Equal(t, account, got)

	certificates, err := store.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{cert}, certificates)

	require.Error(t, store.Ready("le"))

	// A certificate that cannot be sent is kept, and sent when the server is back.
	issued := newStoreTestCertificate(t, in90Days(), "example.org")
	require.Error(t, store.SaveCertificates("le", []*CertAndStore{cert, issued}))

	require.NoError(t, server.Restart())

	var changes atomic.Int32
	store.Watch("le", func() { changes.Add(1) })

	store.refresh("le")
	require.NoError(t, store.Ready("le"))
	assert.Equal(t, int32(0), changes.Load())

	other := newTestRedisStore(t, server, "srv2")

	certificates, err = other.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{cert, issued}, certificates)
}

func TestRedisStore_serverLosesItsData(t *testing.T) {
	server := miniredis.RunT(t)

	store := newTestRedisStore(t, server, "srv1")

	cert := newStoreTestCertificate(t, in90Days(), "example.com")
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{cert}))

	server.FlushAll()

	store.refresh("le")

	other := newTestRedisStore(t, server, "srv2")

	certificates, err := other.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{cert}, certificates)
}

func readTcertsKey(t *testing.T, server *miniredis.Miniredis, key string) *certJSON {
	t.Helper()

	data, err := server.Get(key)
	require.NoError(t, err)

	doc := &certJSON{}
	require.NoError(t, json.Unmarshal([]byte(data), doc))

	return doc
}

func TestRedisStore_tcerts(t *testing.T) {
	server := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	events := client.Subscribe(t.Context(), "tcerts:events")
	t.Cleanup(func() { _ = events.Close() })
	_, err := events.Receive(t.Context())
	require.NoError(t, err)

	store := newTestRedisStore(t, server, "srv1")

	cert := newStoreTestCertificate(t, in90Days(), "example.com", "*.example.com", "www.example.org")
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{cert}))

	for _, name := range []string{"example.com", "www.example.org"} {
		doc := readTcertsKey(t, server, "tcerts:cert:"+name)
		assert.Equal(t, name, doc.FQDN)
		assert.Equal(t, []string{"example.com", "*.example.com", "www.example.org"}, doc.SANs)
		assert.Equal(t, "acme", doc.Source)
		assert.Equal(t, "le", doc.AcmeResolver)
		assert.Equal(t, string(cert.Certificate.Certificate), doc.PEM.Fullchain)
		assert.Equal(t, string(cert.Key), doc.PEM.Privkey)
	}

	var announced []string
	for range 2 {
		message, err := events.ReceiveMessage(t.Context())
		require.NoError(t, err)
		announced = append(announced, message.Payload)
	}
	assert.ElementsMatch(t, []string{"example.com", "www.example.org"}, announced)

	// A renewal replaces the copy, and an older certificate does not.
	renewed := newStoreTestCertificate(t, in90Days().Add(24*time.Hour), "example.com", "*.example.com", "www.example.org")
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{renewed}))
	assert.Equal(t, string(renewed.Key), readTcertsKey(t, server, "tcerts:cert:example.com").PEM.Privkey)

	stale := newTestRedisStore(t, server, "srv2")
	stale.Offer("le", nil, []*CertAndStore{cert})
	assert.Equal(t, string(renewed.Key), readTcertsKey(t, server, "tcerts:cert:example.com").PEM.Privkey)
}

func TestRedisStore_tcerts_wildcardCollision(t *testing.T) {
	server := miniredis.RunT(t)

	store := newTestRedisStore(t, server, "srv1")

	wildcard := newStoreTestCertificate(t, in90Days().Add(24*time.Hour), "*.example.com")
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{wildcard}))
	assert.Equal(t, []string{"*.example.com"}, readTcertsKey(t, server, "tcerts:cert:example.com").SANs)

	apex := newStoreTestCertificate(t, in90Days(), "example.com")
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{wildcard, apex}))

	assert.Equal(t, []string{"example.com"}, readTcertsKey(t, server, "tcerts:cert:example.com").SANs)
	assert.Equal(t, []string{"*.example.com"}, readTcertsKey(t, server, "tcerts:cert:_wildcard.example.com").SANs)
}

func TestRedisStore_tcerts_disabledOrRenamed(t *testing.T) {
	server := miniredis.RunT(t)

	store := newTestRedisStore(t, server, "srv1")
	store.setTcertsPrefix("le", "")
	store.setTcertsPrefix("zerossl", "certs")

	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{newStoreTestCertificate(t, in90Days(), "example.com")}))
	require.NoError(t, store.SaveCertificates("zerossl", []*CertAndStore{newStoreTestCertificate(t, in90Days(), "example.org")}))

	assert.False(t, server.Exists("tcerts:cert:example.com"))
	assert.False(t, server.Exists("certs:cert:example.com"))
	assert.True(t, server.Exists("certs:cert:example.org"))
}

// With the instances running, a certificate one of them stores reaches the others through the cluster events.
func TestRedisStore_Run(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestRedisStore(t, server, "srv1")
	second := newTestRedisStore(t, server, "srv2")

	changed := make(chan struct{}, 4)
	second.Watch("le", func() { changed <- struct{}{} })

	_, err := second.GetCertificates("le")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	connected := make(chan struct{}, 1)
	second.cluster.OnConnect(func() {
		select {
		case connected <- struct{}{}:
		default:
		}
	})

	go func() {
		defer close(done)
		go second.Run(ctx)
		second.cluster.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	select {
	case <-connected:
	case <-time.After(5 * time.Second):
		require.Fail(t, "the instance did not connect")
	}

	cert := newStoreTestCertificate(t, in90Days(), "example.com")
	require.NoError(t, first.SaveCertificates("le", []*CertAndStore{cert}))

	select {
	case <-changed:
	case <-time.After(5 * time.Second):
		require.Fail(t, "the certificate did not reach the other instance")
	}

	certificates, err := second.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{cert}, certificates)
}

func TestStoreBuilder_Build_redisWithDirExport(t *testing.T) {
	server := miniredis.RunT(t)
	tmp := t.TempDir()

	conf := func(dir string) *Configuration {
		return &Configuration{
			Storage:   filepath.Join(tmp, "unused.json"),
			CertStore: &CertStore{Primary: StoreRedis, Dir: filepath.Join(tmp, dir), Export: []string{StoreDir}},
		}
	}

	first, err := NewStoreBuilder(safe.NewPool(t.Context()), newStoreTestCluster(t, server, "srv1")).Build("le", conf("first"))
	require.NoError(t, err)

	second, err := NewStoreBuilder(safe.NewPool(t.Context()), newStoreTestCluster(t, server, "srv2")).Build("le", conf("second"))
	require.NoError(t, err)

	changed := make(chan struct{}, 4)
	second.(storeWatcher).Watch("le", func() { changed <- struct{}{} })

	certificates, err := second.GetCertificates("le")
	require.NoError(t, err)
	assert.Empty(t, certificates)

	account := &Account{Email: "ops@example.com", PrivateKey: []byte("key")}
	cert := newStoreTestCertificate(t, in90Days(), "example.com")

	require.NoError(t, first.SaveAccount("le", account))
	require.NoError(t, first.SaveCertificates("le", []*CertAndStore{cert}))

	// The instance that obtained the certificate has it in its files.
	assert.FileExists(t, filepath.Join(tmp, "first", "example.com", "privkey.pem"))
	assert.FileExists(t, filepath.Join(tmp, "first", "_traefix", "le", "account.json"))

	// The other instance writes its files when it learns about the certificate.
	second.(*MultiStore).primary.(*RedisStore).refresh("le")

	select {
	case <-changed:
	default:
		require.Fail(t, "the certificate did not reach the other instance")
	}

	assert.FileExists(t, filepath.Join(tmp, "second", "example.com", "privkey.pem"))
	assert.FileExists(t, filepath.Join(tmp, "second", "_traefix", "le", "account.json"))

	require.NoError(t, second.(storeGate).Ready("le"))

	// An instance that starts without the server serves what is in its files, and obtains nothing.
	thirdMember := newStoreTestCluster(t, server, "srv3")
	server.Close()

	third, err := NewStoreBuilder(safe.NewPool(t.Context()), thirdMember).Build("le", conf("second"))
	require.NoError(t, err)

	certificates, err = third.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{cert}, certificates)

	got, err := third.GetAccount("le")
	require.NoError(t, err)
	assert.Equal(t, account, got)

	require.Error(t, third.(storeGate).Ready("le"))
}

func TestStoreBuilder_Build_redisNeedsTheCluster(t *testing.T) {
	_, err := NewStoreBuilder(safe.NewPool(t.Context()), nil).Build("le", &Configuration{
		Storage:   filepath.Join(t.TempDir(), "acme.json"),
		CertStore: &CertStore{Primary: StoreRedis},
	})
	require.Error(t, err)
}

// An instance that joins the cluster brings the content of its storage file.
func TestStoreBuilder_Build_redisTakesTheStorageFile(t *testing.T) {
	server := miniredis.RunT(t)
	storage := filepath.Join(t.TempDir(), "acme.json")

	account := &Account{Email: "ops@example.com", PrivateKey: []byte("key")}
	cert := newStoreTestCertificate(t, in90Days(), "example.com")
	writeStoreTestFile(t, storage, map[string]*StoredData{
		"le": {Account: account, Certificates: []*CertAndStore{cert}},
	})

	store, err := NewStoreBuilder(safe.NewPool(t.Context()), newStoreTestCluster(t, server, "srv1")).Build("le", &Configuration{
		Storage:   storage,
		CertStore: &CertStore{Primary: StoreRedis},
	})
	require.NoError(t, err)

	certificates, err := store.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{cert}, certificates)

	other := newTestRedisStore(t, server, "srv2")

	got, err := other.GetAccount("le")
	require.NoError(t, err)
	assert.Equal(t, account, got)

	certificates, err = other.GetCertificates("le")
	require.NoError(t, err)
	assert.Equal(t, []*CertAndStore{cert}, certificates)
}

func TestProvider_checkStore(t *testing.T) {
	server := miniredis.RunT(t)

	store := newTestRedisStore(t, server, "srv1")
	other := newTestRedisStore(t, server, "srv2")

	provider := &Provider{
		Configuration: &Configuration{CAServer: "https://acme.example.com/directory"},
		ResolverName:  "le",
		Store:         store,
	}

	checkStore := func(provider *Provider) error {
		release, err := provider.checkStore(t.Context())
		release()

		return err
	}

	// The certificates were not read from the server yet.
	require.Error(t, checkStore(provider))

	_, err := store.GetCertificates("le")
	require.NoError(t, err)

	require.NoError(t, checkStore(provider))
	assert.Nil(t, provider.account)

	// Another instance registered the account in the meantime: the resolver uses it instead of registering one.
	account := &Account{Email: "ops@example.com", Registration: &Resource{URI: "https://acme.example.com/acct/1"}}
	require.NoError(t, other.SaveAccount("le", account))

	require.NoError(t, checkStore(provider))
	assert.Equal(t, account, provider.account)

	// A store that is not shared never holds the resolver back.
	provider = &Provider{Configuration: &Configuration{}, ResolverName: "le", Store: NewDirStore(t.TempDir())}
	require.NoError(t, checkStore(provider))
}
