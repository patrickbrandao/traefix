package acme

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-acme/lego/v5/certificate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/safe"
	"github.com/traefik/traefik/v3/pkg/types"
)

const (
	testElectionStep    = 60 * time.Millisecond
	testElectionLockTTL = 10 * time.Second
)

// testInstance is a Traefix instance of a cluster, with one resolver whose requests to the ACME server are recorded.
type testInstance struct {
	t        *testing.T
	nodeID   string
	member   *cluster.Cluster
	store    *RedisStore
	provider *Provider
	coord    *coordinator
	now      time.Time

	mu       sync.Mutex
	obtained []string
	err      error
}

// newTestInstance creates an instance. With resolver set to false, the instance is in the cluster without the resolver.
func newTestInstance(t *testing.T, server *miniredis.Miniredis, nodeID string, priority int, resolver bool) *testInstance {
	t.Helper()

	conf := &cluster.Configuration{Redis: &cluster.Redis{}}
	conf.SetDefaults()
	conf.Redis.SetDefaults()
	conf.NodeID = nodeID
	conf.Priority = priority
	conf.Redis.Endpoints = []string{server.Addr()}

	member, err := cluster.New(t.Context(), conf)
	require.NoError(t, err)

	instance := &testInstance{t: t, nodeID: nodeID, member: member, now: time.Now()}

	if resolver {
		instance.store = NewRedisStore(member)
		instance.store.setTcertsPrefix("le", "")
		instance.store.setElection("le", testElectionStep, testElectionLockTTL)

		instance.provider = &Provider{
			Configuration:     &Configuration{CAServer: "https://acme.example.com/directory", Email: "ops@example.com"},
			ResolverName:      "le",
			Store:             instance.store,
			configurationChan: make(chan dynamic.Message, 1000),
			resolvingDomains:  map[string]struct{}{},
			pool:              safe.NewPool(t.Context()),
		}

		instance.coord = instance.store.coordinatorFor("le")
		instance.coord.jitter = func() time.Duration { return 0 }
		instance.coord.settle = 0
		instance.coord.now = func() time.Time {
			instance.mu.Lock()
			defer instance.mu.Unlock()

			return instance.now
		}
		instance.coord.obtain = instance.obtain
		instance.coord.attach(instance.provider)

		instance.store.Watch("le", instance.provider.reloadCertificates)

		_, err = instance.store.GetCertificates("le")
		require.NoError(t, err)
	}

	// The instance announces itself, with its resolver, before the test goes on.
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		member.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	require.Eventually(t, func() bool {
		nodes, err := member.Nodes(t.Context())
		if err != nil {
			return false
		}

		for _, node := range nodes {
			if node.ID == nodeID {
				return true
			}
		}

		return false
	}, 5*time.Second, 5*time.Millisecond)

	return instance
}

// obtain stands for the ACME server: it returns a new certificate for the domains, or the error set on the instance.
func (i *testInstance) obtain(_ context.Context, domain types.Domain, _ string) (types.Domain, *certificate.Resource, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.obtained = append(i.obtained, domain.Main)

	if i.err != nil {
		return types.Domain{}, nil, i.err
	}

	cert := newStoreTestCertificate(i.t, in90Days(), domain.Main, domain.SANs...)

	return domain, &certificate.Resource{Certificate: cert.Certificate.Certificate, PrivateKey: cert.Key}, nil
}

func (i *testInstance) requests() int {
	i.mu.Lock()
	defer i.mu.Unlock()

	return len(i.obtained)
}

func (i *testInstance) advance(d time.Duration) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.now = i.now.Add(d)
}

// resolve asks for the certificate of a domain as the configuration of the instance would, and stores it when obtained.
func (i *testInstance) resolve(domain string) {
	i.t.Helper()

	dom, cert, err := i.provider.resolveCertificate(i.t.Context(), types.Domain{Main: domain}, "default")
	if err != nil {
		return
	}

	require.NoError(i.t, i.provider.addCertificateForDomain(dom, cert, "default"))
}

func (i *testInstance) covers(domain string) bool {
	i.store.refresh("le")

	return len(i.provider.uncoveredDomains([]string{domain}, "default")) == 0
}

func resolveAll(domain string, instances ...*testInstance) {
	var wg sync.WaitGroup
	for _, instance := range instances {
		wg.Go(func() { instance.resolve(domain) })
	}
	wg.Wait()
}

func TestCoordinator_highestPriorityObtains(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 100, true)
	second := newTestInstance(t, server, "srv2", 50, true)
	third := newTestInstance(t, server, "srv3", 50, true)

	resolveAll("example.com", third, second, first)

	assert.Equal(t, 1, first.requests())
	assert.Equal(t, 0, second.requests())
	assert.Equal(t, 0, third.requests())

	// Every instance ends with the certificate, and none asks again.
	for _, instance := range []*testInstance{first, second, third} {
		assert.True(t, instance.covers("example.com"), instance.nodeID)
	}

	resolveAll("example.com", third, second, first)
	assert.Equal(t, 1, first.requests()+second.requests()+third.requests())

	// The request is closed for the cluster.
	assert.False(t, server.Exists("traefix:acme:{le}:want:"+certID([]string{"example.com"})))
}

// The instances with the same priority race for the request, and one of them gets it.
func TestCoordinator_samePriority(t *testing.T) {
	server := miniredis.RunT(t)

	// An instance with a higher priority that does not have the resolver does not count.
	newTestInstance(t, server, "srv1", 100, false)
	second := newTestInstance(t, server, "srv2", 50, true)
	third := newTestInstance(t, server, "srv3", 50, true)

	resolveAll("example.com", second, third)

	assert.Equal(t, 1, second.requests()+third.requests())
	assert.True(t, second.covers("example.com"))
	assert.True(t, third.covers("example.com"))
}

func TestCoordinator_lowerPriorityWaitsForItsTurn(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 100, true)
	second := newTestInstance(t, server, "srv2", 50, true)

	// The instance with the highest priority does not ask: the other one does, after its step.
	start := time.Now()
	second.resolve("example.com")

	assert.Equal(t, 1, second.requests())
	assert.GreaterOrEqual(t, time.Since(start), testElectionStep)
	assert.True(t, first.covers("example.com"))
}

func TestCoordinator_priorityZeroNeverObtains(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 0, true)
	second := newTestInstance(t, server, "srv2", 50, true)

	// Only the first instance has the router.
	first.resolve("example.com")

	assert.Equal(t, 0, first.requests())
	assert.True(t, server.Exists("traefix:acme:{le}:want:"+certID([]string{"example.com"})))

	// The second instance learns about the request, obtains the certificate and stores it.
	second.coord.poll(t.Context())

	require.Eventually(t, func() bool { return first.covers("example.com") }, 5*time.Second, 10*time.Millisecond)

	assert.Equal(t, 0, first.requests())
	assert.Equal(t, 1, second.requests())

	// The instance that asked for it forgets the request.
	first.coord.poll(t.Context())
	assert.False(t, server.Exists("traefix:acme:{le}:want:"+certID([]string{"example.com"})))
}

func TestCoordinator_failure(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 50, true)
	second := newTestInstance(t, server, "srv2", 50, true)

	first.mu.Lock()
	first.err = errors.New("the validation failed")
	first.mu.Unlock()

	first.resolve("example.com")
	require.Equal(t, 1, first.requests())

	failure, err := second.coord.readFail(t.Context(), certID([]string{"example.com"}))
	require.NoError(t, err)
	require.NotNil(t, failure)
	assert.Equal(t, 1, failure.Attempts)
	assert.Equal(t, "srv1", failure.Node)
	assert.Equal(t, "the validation failed", failure.Error)
	assert.Equal(t, first.now.Add(2*time.Minute).Unix(), failure.NextRetryAt)

	// No instance asks before the time of the next attempt.
	resolveAll("example.com", first, second)
	assert.Equal(t, 1, first.requests())
	assert.Equal(t, 0, second.requests())

	// At the next attempt, the instance that failed lets the other one go first.
	first.advance(3 * time.Minute)
	second.advance(3 * time.Minute)

	resolveAll("example.com", first, second)
	assert.Equal(t, 1, first.requests())
	assert.Equal(t, 1, second.requests())

	assert.True(t, first.covers("example.com"))
	assert.False(t, server.Exists("traefix:acme:{le}:fail:"+certID([]string{"example.com"})))
}

func TestCoordinator_failed_schedule(t *testing.T) {
	server := miniredis.RunT(t)

	instance := newTestInstance(t, server, "srv1", 50, true)

	id := certID([]string{"example.com"})
	instance.coord.request(t.Context(), id, []string{"example.com"}, "default")

	expected := []time.Duration{
		2 * time.Minute, 15 * time.Minute, 30 * time.Minute, time.Hour, 2 * time.Hour, 6 * time.Hour, 6 * time.Hour,
	}

	for attempt, wait := range expected {
		instance.coord.failed(t.Context(), id, errors.New("failed"), true)

		failure, err := instance.coord.readFail(t.Context(), id)
		require.NoError(t, err)
		require.NotNil(t, failure)

		assert.Equal(t, attempt+1, failure.Attempts)
		assert.Equal(t, instance.now.Add(wait).Unix(), failure.NextRetryAt)

		instance.advance(wait)
	}

	// A renewal is never abandoned.
	instance.advance(15 * 24 * time.Hour)
	instance.coord.failed(t.Context(), id, errors.New("failed"), false)
	assert.True(t, server.Exists("traefix:acme:{le}:fail:"+id))

	// A certificate that keeps failing is abandoned, until the configuration asks for it again.
	instance.coord.failed(t.Context(), id, errors.New("failed"), true)
	assert.False(t, server.Exists("traefix:acme:{le}:fail:"+id))
	assert.False(t, server.Exists("traefix:acme:{le}:want:"+id))
	assert.Empty(t, instance.coord.pending)
}

// An instance that stops in the middle of a request does not hold the certificate back for long.
func TestCoordinator_holderStops(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 50, true)
	second := newTestInstance(t, server, "srv2", 50, true)

	_, ok, err := first.member.TryLock(t.Context(), first.coord.lockName(certID([]string{"example.com"})), testElectionLockTTL)
	require.NoError(t, err)
	require.True(t, ok)

	second.resolve("example.com")
	assert.Equal(t, 0, second.requests())

	server.FastForward(testElectionLockTTL + time.Second)

	second.resolve("example.com")
	assert.Equal(t, 1, second.requests())
}

// The instance checks the store again once it has the lock: another instance may have just stored the certificate.
func TestCoordinator_storedInTheMeantime(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 50, true)
	second := newTestInstance(t, server, "srv2", 50, true)

	cert := newStoreTestCertificate(t, in90Days(), "example.com")
	require.NoError(t, first.store.SaveCertificates("le", []*CertAndStore{cert}))

	// The second instance did not hear about the certificate yet.
	require.NotEmpty(t, second.provider.uncoveredDomains([]string{"example.com"}, "default"))

	second.resolve("example.com")
	assert.Equal(t, 0, second.requests())

	_, held, err := second.member.LockHolder(t.Context(), second.coord.lockName(certID([]string{"example.com"})))
	require.NoError(t, err)
	assert.False(t, held)
}

func TestProvider_electRenewals(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 50, true)
	second := newTestInstance(t, server, "srv2", 50, true)
	observer := newTestInstance(t, server, "srv3", 0, true)

	first.resolve("example.com")
	require.True(t, second.covers("example.com"))
	require.True(t, observer.covers("example.com"))

	// The lock of the request that obtained the certificate is left to expire.
	server.FastForward(testElectionLockTTL + time.Second)

	current := func(instance *testInstance) []*CertAndStore {
		instance.provider.certificatesMu.RLock()
		defer instance.provider.certificatesMu.RUnlock()

		return append([]*CertAndStore(nil), instance.provider.certificates...)
	}

	// An instance with the priority 0 renews nothing.
	elected, done := observer.provider.electRenewals(t.Context(), current(observer))
	done()
	assert.Empty(t, elected)

	// One instance renews the certificate: the other one has nothing to do.
	elected, done = first.provider.electRenewals(t.Context(), current(first))
	require.Len(t, elected, 1)

	others, otherDone := second.provider.electRenewals(t.Context(), current(second))
	otherDone()
	assert.Empty(t, others)

	renewed := newStoreTestCertificate(t, in90Days().Add(24*time.Hour), "example.com")
	require.NoError(t, first.provider.addCertificateForDomain(elected[0].Domain,
		&certificate.Resource{Certificate: renewed.Certificate.Certificate, PrivateKey: renewed.Key}, "default"))
	done()

	id := certID([]string{"example.com"})

	_, held, err := first.member.LockHolder(t.Context(), first.coord.lockName(id))
	require.NoError(t, err)
	assert.False(t, held)
	assert.False(t, server.Exists("traefix:acme:{le}:fail:"+id))

	// The other instance gets the renewed certificate, and does not renew it again.
	second.store.refresh("le")
	require.True(t, second.provider.hasCertificate(types.Domain{Main: "example.com"}, renewed.Certificate.Certificate))

	// A renewal that fails is recorded, and no instance tries again before the next attempt.
	elected, done = second.provider.electRenewals(t.Context(), current(second))
	require.Len(t, elected, 1)
	done()

	failure, err := first.coord.readFail(t.Context(), id)
	require.NoError(t, err)
	require.NotNil(t, failure)
	assert.Equal(t, "srv2", failure.Node)

	elected, done = first.provider.electRenewals(t.Context(), current(first))
	done()
	assert.Empty(t, elected)
}

// The instances share one ACME account: the first one registers it, and the others wait for it.
func TestProvider_checkStore_registersTheAccountOnce(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 50, true)
	second := newTestInstance(t, server, "srv2", 50, true)

	release, err := first.provider.checkStore(t.Context())
	require.NoError(t, err)
	assert.Nil(t, first.provider.account)

	waited := make(chan error, 1)
	go func() {
		releaseSecond, err := second.provider.checkStore(t.Context())
		releaseSecond()
		waited <- err
	}()

	select {
	case err := <-waited:
		require.Failf(t, "the second instance did not wait for the account", "error: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	account := &Account{Email: "ops@example.com", Registration: &Resource{URI: "https://acme.example.com/acct/1"}}
	require.NoError(t, first.store.SaveAccount("le", account))
	release()

	select {
	case err := <-waited:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		require.Fail(t, "the second instance did not get the account")
	}

	assert.Equal(t, account, second.provider.account)
}

func Test_resolverDigest(t *testing.T) {
	conf := &Configuration{CAServer: "https://acme.example.com/directory", Email: "ops@example.com", KeyType: "EC256", HTTPChallenge: &HTTPChallenge{EntryPoint: "web"}}
	same := &Configuration{CAServer: "https://acme.example.com/directory", Email: "ops@example.com", KeyType: "EC256", HTTPChallenge: &HTTPChallenge{EntryPoint: "http"}, Storage: "other.json"}
	other := &Configuration{CAServer: "https://acme.example.com/directory", Email: "ops@example.com", KeyType: "RSA4096", HTTPChallenge: &HTTPChallenge{EntryPoint: "web"}}

	// The entry point and the storage are local to an instance.
	assert.Equal(t, resolverDigest(conf), resolverDigest(same))
	assert.NotEqual(t, resolverDigest(conf), resolverDigest(other))
}
