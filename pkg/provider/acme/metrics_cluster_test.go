package acme

import (
	"errors"
	"net/http"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	gokitmetrics "github.com/go-kit/kit/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/observability/metrics"
	"github.com/traefik/traefik/v3/pkg/safe"
)

// recordingCounter keeps the label values of each increment.
type recordingCounter struct {
	mu    *sync.Mutex
	adds  *[][]string
	label []string
}

func newRecordingCounter() *recordingCounter {
	return &recordingCounter{mu: &sync.Mutex{}, adds: &[][]string{}}
}

func (c *recordingCounter) With(labelValues ...string) gokitmetrics.Counter {
	return &recordingCounter{mu: c.mu, adds: c.adds, label: append(slices.Clone(c.label), labelValues...)}
}

func (c *recordingCounter) Add(float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	*c.adds = append(*c.adds, c.label)
}

func (c *recordingCounter) count(labelValues ...string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for _, labels := range *c.adds {
		if slices.Equal(labels, labelValues) {
			count++
		}
	}

	return count
}

type recordingRegistry struct {
	metrics.Registry

	elections *recordingCounter
	issuances *recordingCounter
	lookups   *recordingCounter
	writes    *recordingCounter
}

func (r *recordingRegistry) ACMEClusterElectionsCounter() gokitmetrics.Counter { return r.elections }

func (r *recordingRegistry) ACMEClusterIssuancesCounter() gokitmetrics.Counter { return r.issuances }

func (r *recordingRegistry) ACMEClusterChallengeLookupsCounter() gokitmetrics.Counter {
	return r.lookups
}

func (r *recordingRegistry) ACMEStoreWritesCounter() gokitmetrics.Counter { return r.writes }

func setRecordingRegistry(t *testing.T) *recordingRegistry {
	t.Helper()

	registry := &recordingRegistry{
		Registry:  metrics.NewVoidRegistry(),
		elections: newRecordingCounter(),
		issuances: newRecordingCounter(),
		lookups:   newRecordingCounter(),
		writes:    newRecordingCounter(),
	}

	SetMetricsRegistry(registry)
	t.Cleanup(func() { metricsRegistry.Store(nil) })

	return registry
}

func TestMetrics_electionsAndIssuances(t *testing.T) {
	registry := setRecordingRegistry(t)
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 100, true)
	second := newTestInstance(t, server, "srv2", 50, true)

	resolveAll("example.com", first, second)
	require.True(t, second.covers("example.com"))

	assert.Equal(t, 1, registry.elections.count("resolver", "le", "result", "won"))
	assert.Equal(t, 1, registry.issuances.count("resolver", "le", "kind", "obtain", "result", "success"))

	first.err = errors.New("boom")
	first.resolve("example.org")

	assert.Equal(t, 1, registry.issuances.count("resolver", "le", "kind", "obtain", "result", "failure"))

	// The failure makes every instance wait: the request is skipped.
	second.resolve("example.org")

	assert.Positive(t, registry.elections.count("resolver", "le", "result", "skipped"))

	// A renewal that does not replace the certificate is a failure.
	server.FastForward(testElectionLockTTL + time.Second)

	elected, done := first.provider.electRenewals(t.Context(), first.certificates())
	require.Len(t, elected, 1)
	done()

	assert.Equal(t, 1, registry.issuances.count("resolver", "le", "kind", "renew", "result", "failure"))
}

func TestMetrics_challengeLookups(t *testing.T) {
	registry := setRecordingRegistry(t)
	server := miniredis.RunT(t)

	first := newTestClusterChallenge(t, server, "srv1")
	second := newTestClusterChallenge(t, server, "srv2")

	require.NoError(t, first.Present(t.Context(), "app.example.com", testChallengeToken, testChallengeKeyAuth))

	assert.Equal(t, http.StatusOK, askChallenge(t, first, "app.example.com", testChallengeToken).Code)
	assert.Equal(t, http.StatusOK, askChallenge(t, second, "app.example.com", testChallengeToken).Code)
	assert.Equal(t, http.StatusNotFound, askChallenge(t, second, "other.example.com", testChallengeToken).Code)
	assert.Equal(t, http.StatusNotFound, askChallenge(t, second, "other.example.com", testChallengeToken).Code)
	assert.Equal(t, http.StatusNotFound, askChallenge(t, second, "app.example.com", "xmrlpc.php").Code)

	server.Close()

	assert.Equal(t, http.StatusNotFound, askChallenge(t, second, "down.example.com", testChallengeToken).Code)

	assert.Equal(t, 1, registry.lookups.count("result", "local"))
	assert.Equal(t, 1, registry.lookups.count("result", "redis"))
	assert.Equal(t, 2, registry.lookups.count("result", "miss"))
	assert.Equal(t, 1, registry.lookups.count("result", "invalid"))
	assert.Equal(t, 1, registry.lookups.count("result", "error"))
}

func TestMetrics_storeWrites(t *testing.T) {
	registry := setRecordingRegistry(t)
	server := miniredis.RunT(t)
	tmp := t.TempDir()

	store, err := NewStoreBuilder(safe.NewPool(t.Context()), newStoreTestCluster(t, server, "srv1")).Build("le", &Configuration{
		Storage:   filepath.Join(tmp, "unused.json"),
		CertStore: &CertStore{Primary: StoreRedis, Dir: filepath.Join(tmp, "certs"), Export: []string{StoreDir}},
	})
	require.NoError(t, err)

	// The first use of the store brings the directory in line with the server.
	_, err = store.GetCertificates("le")
	require.NoError(t, err)

	seeded := registry.writes.count("resolver", "le", "target", "dir", "result", "success")

	cert := newStoreTestCertificate(t, in90Days(), "example.com")
	require.NoError(t, store.SaveCertificates("le", []*CertAndStore{cert}))

	assert.Equal(t, 1, registry.writes.count("resolver", "le", "target", "redis", "result", "success"))
	assert.Equal(t, seeded+1, registry.writes.count("resolver", "le", "target", "dir", "result", "success"))
	assert.Equal(t, 1, registry.writes.count("resolver", "le", "target", "tcerts", "result", "success"))

	// Without the server, the files are still written.
	server.Close()

	other := newStoreTestCertificate(t, in90Days(), "example.org")
	require.Error(t, store.SaveCertificates("le", []*CertAndStore{cert, other}))

	assert.Equal(t, 1, registry.writes.count("resolver", "le", "target", "redis", "result", "failure"))
	assert.Equal(t, seeded+2, registry.writes.count("resolver", "le", "target", "dir", "result", "success"))
	assert.FileExists(t, filepath.Join(tmp, "certs", "example.org", "privkey.pem"))
}
