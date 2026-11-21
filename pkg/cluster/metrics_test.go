package cluster

import (
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	gokitmetrics "github.com/go-kit/kit/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/observability/metrics"
)

type recordingGauge struct {
	mu    sync.Mutex
	value float64
	set   bool
}

func (g *recordingGauge) With(...string) gokitmetrics.Gauge { return g }

func (g *recordingGauge) Add(float64) {}

func (g *recordingGauge) Set(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.value = value
	g.set = true
}

func (g *recordingGauge) get() (float64, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.value, g.set
}

type recordingRegistry struct {
	metrics.Registry

	redisUp *recordingGauge
	nodes   *recordingGauge
}

func (r *recordingRegistry) ClusterRedisUpGauge() gokitmetrics.Gauge { return r.redisUp }

func (r *recordingRegistry) ClusterNodesGauge() gokitmetrics.Gauge { return r.nodes }

func TestCluster_metrics(t *testing.T) {
	registry := &recordingRegistry{Registry: metrics.NewVoidRegistry(), redisUp: &recordingGauge{}, nodes: &recordingGauge{}}

	SetMetricsRegistry(registry)
	t.Cleanup(func() { metricsRegistry.Store(nil) })

	server := miniredis.RunT(t)

	first := newTestCluster(t, server, "srv1", 100)
	second := newTestCluster(t, server, "srv2", 100)

	first.heartbeat(t.Context())
	second.heartbeat(t.Context())

	up, set := registry.redisUp.get()
	require.True(t, set)
	assert.InDelta(t, 1, up, 0)

	nodes, _ := registry.nodes.get()
	assert.InDelta(t, 2, nodes, 0)

	server.Close()

	first.heartbeat(t.Context())

	up, _ = registry.redisUp.get()
	assert.InDelta(t, 0, up, 0)

	// The count of instances is the last one seen.
	nodes, _ = registry.nodes.get()
	assert.InDelta(t, 2, nodes, 0)
}
