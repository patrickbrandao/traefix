package mesh

import (
	"context"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	gokitmetrics "github.com/go-kit/kit/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/observability/metrics"
)

// testNode is an instance of a mesh that the test starts and stops.
type testNode struct {
	t       *testing.T
	conf    *cluster.Configuration
	address string
	mesh    *Mesh
	stop    func()
}

// newTestNode prepares an instance that listens on a free loopback address, and links to the given addresses.
func newTestNode(t *testing.T, nodeID, token string, change func(conf *cluster.Configuration), peers ...string) *testNode {
	t.Helper()

	conf := newTestConfiguration()
	conf.NodeID = nodeID
	conf.Token = token
	conf.Mesh.Address = freeAddress(t)
	conf.Mesh.Advertise = []string{conf.Mesh.Address}
	conf.Mesh.Peers = peers
	conf.Mesh.LinkTimeout = ptypes.Duration(1500 * time.Millisecond)

	if change != nil {
		change(conf)
	}

	node := &testNode{t: t, conf: conf, address: conf.Mesh.Address}
	node.start()

	return node
}

func (n *testNode) start() {
	n.t.Helper()

	member, err := cluster.New(n.t.Context(), n.conf)
	require.NoError(n.t, err)

	n.mesh, err = New(n.conf, member)
	require.NoError(n.t, err)

	// The waits of the real thing would make the tests last minutes.
	n.mesh.redialFirst = 50 * time.Millisecond
	n.mesh.redialLongest = 200 * time.Millisecond
	n.mesh.quicDialTimeout = 500 * time.Millisecond

	ctx, cancel := context.WithCancel(n.t.Context())
	done := make(chan struct{})

	go func() {
		defer close(done)
		n.mesh.Run(ctx)
	}()

	stopped := false
	n.stop = func() {
		if stopped {
			return
		}
		stopped = true

		cancel()
		<-done
	}

	n.t.Cleanup(n.stop)
}

// link returns the state of the only link the instance opens.
func (n *testNode) link() LinkStatus {
	n.t.Helper()

	status := n.mesh.Status()
	require.Len(n.t, status.Links, 1)

	return status.Links[0]
}

func (n *testNode) waitLink(up bool) LinkStatus {
	n.t.Helper()

	require.Eventually(n.t, func() bool { return n.link().Up == up }, 10*time.Second, 10*time.Millisecond)

	return n.link()
}

func TestMesh_link(t *testing.T) {
	testCases := []struct {
		desc      string
		transport string
		expected  string
	}{
		{desc: "auto prefers QUIC", transport: cluster.MeshTransportAuto, expected: TransportQUIC},
		{desc: "quic", transport: cluster.MeshTransportQUIC, expected: TransportQUIC},
		{desc: "tcp", transport: cluster.MeshTransportTCP, expected: TransportTCP},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			withTransport := func(conf *cluster.Configuration) { conf.Mesh.Transport = test.transport }

			first := newTestNode(t, "srv1", testToken, withTransport)
			second := newTestNode(t, "srv2", testToken, withTransport, first.address)

			status := second.waitLink(true)

			assert.Equal(t, "srv1", status.NodeID)
			assert.Equal(t, test.expected, status.Transport)
			assert.Empty(t, status.Error)

			// The instance that was linked to knows who did, and how to reach it back.
			require.Eventually(t, func() bool { return len(first.mesh.Status().Inbound) == 1 }, 5*time.Second, 10*time.Millisecond)

			inbound := first.mesh.Status().Inbound[0]
			assert.Equal(t, "srv2", inbound.NodeID)
			assert.Equal(t, []string{second.address}, inbound.Advertise)

			// The pings measure the link.
			require.Eventually(t, func() bool { return second.link().RTT > 0 }, 5*time.Second, 10*time.Millisecond)
		})
	}
}

func TestMesh_link_fallsBackToTCP(t *testing.T) {
	// The first instance does not answer QUIC, like one behind a network that drops UDP.
	first := newTestNode(t, "srv1", testToken, func(conf *cluster.Configuration) { conf.Mesh.Transport = cluster.MeshTransportTCP })
	second := newTestNode(t, "srv2", testToken, nil, first.address)

	status := second.waitLink(true)

	assert.Equal(t, "srv1", status.NodeID)
	assert.Equal(t, TransportTCP, status.Transport)
}

func TestMesh_link_upgradesToQUIC(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, func(conf *cluster.Configuration) { conf.Mesh.Transport = cluster.MeshTransportTCP })

	second := newTestNode(t, "srv2", testToken, nil, first.address)
	require.Equal(t, TransportTCP, second.waitLink(true).Transport)

	// QUIC gets through again: the instance comes back answering it, on the same address.
	first.stop()
	second.waitLink(false)

	second.mesh.quicRetryInterval = 100 * time.Millisecond

	first.conf.Mesh.Transport = cluster.MeshTransportAuto
	first.start()

	require.Eventually(t, func() bool {
		status := second.link()
		return status.Up && status.Transport == TransportQUIC
	}, 10*time.Second, 10*time.Millisecond)
}

func TestMesh_link_lostAndBack(t *testing.T) {
	for _, transport := range transports() {
		t.Run(transport, func(t *testing.T) {
			withTransport := func(conf *cluster.Configuration) { conf.Mesh.Transport = transport }

			first := newTestNode(t, "srv1", testToken, withTransport)
			second := newTestNode(t, "srv2", testToken, withTransport, first.address)

			up := second.waitLink(true)

			first.stop()

			down := second.waitLink(false)
			assert.NotEmpty(t, down.Error)
			assert.Equal(t, "srv1", down.NodeID)
			assert.False(t, down.Since.Before(up.Since))

			// The instance comes back on the same address: the link does too, without anyone asking.
			first.start()

			assert.Equal(t, "srv1", second.waitLink(true).NodeID)
		})
	}
}

func TestMesh_link_anotherToken(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv2", testOtherToken, nil, first.address)

	require.Eventually(t, func() bool { return second.link().Error != "" }, 10*time.Second, 10*time.Millisecond)

	status := second.link()
	assert.False(t, status.Up)
	assert.Empty(t, status.NodeID)
	assert.Contains(t, status.Error, "token")
	assert.Empty(t, first.mesh.Status().Inbound)
}

func TestMesh_link_toItself(t *testing.T) {
	// Every instance can be given the same list of peers, which then names the instance itself.
	address := freeAddress(t)

	node := newTestNode(t, "srv1", testToken, func(conf *cluster.Configuration) {
		conf.Mesh.Address = address
		conf.Mesh.Advertise = []string{address}
		conf.Mesh.Peers = []string{address, address}
	})

	require.Eventually(t, func() bool { return node.link().Self }, 10*time.Second, 10*time.Millisecond)

	status := node.link()
	assert.False(t, status.Up)
	assert.Empty(t, status.Error)
	assert.Empty(t, node.mesh.Status().Inbound)
}

func TestMesh_link_sameNodeID(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv1", testToken, nil, first.address)

	require.Eventually(t, func() bool { return second.link().Error != "" }, 10*time.Second, 10*time.Millisecond)

	assert.Contains(t, second.link().Error, "nodeID")
	assert.False(t, second.link().Up)
}

// recordingGauge keeps the last value of each set of label values.
type recordingGauge struct {
	mu     *sync.Mutex
	values map[string]float64
	labels string
}

func newRecordingGauge() *recordingGauge {
	return &recordingGauge{mu: &sync.Mutex{}, values: map[string]float64{}}
}

func (g *recordingGauge) With(labelValues ...string) gokitmetrics.Gauge {
	return &recordingGauge{mu: g.mu, values: g.values, labels: g.labels + strings.Join(labelValues, "=") + ";"}
}

func (g *recordingGauge) Add(float64) {}

func (g *recordingGauge) Set(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.values[g.labels] = value
}

func (g *recordingGauge) get(labels string) (float64, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	value, ok := g.values[labels]

	return value, ok
}

func (g *recordingGauge) series() []string {
	g.mu.Lock()
	defer g.mu.Unlock()

	return slices.Sorted(maps.Keys(g.values))
}

type recordingRegistry struct {
	metrics.Registry

	peers   *recordingGauge
	linkUp  *recordingGauge
	linkRTT *recordingGauge
}

func (r *recordingRegistry) MeshPeersGauge() gokitmetrics.Gauge   { return r.peers }
func (r *recordingRegistry) MeshLinkUpGauge() gokitmetrics.Gauge  { return r.linkUp }
func (r *recordingRegistry) MeshLinkRTTGauge() gokitmetrics.Gauge { return r.linkRTT }

func TestMesh_metrics(t *testing.T) {
	registry := &recordingRegistry{
		Registry: metrics.NewVoidRegistry(),
		peers:    newRecordingGauge(),
		linkUp:   newRecordingGauge(),
		linkRTT:  newRecordingGauge(),
	}

	SetMetricsRegistry(registry)
	t.Cleanup(func() { metricsRegistry.Store(nil) })

	first := newTestNode(t, "srv1", testToken, nil)
	newTestNode(t, "srv2", testToken, nil, first.address, "127.0.0.1:1")

	value := func(gauge *recordingGauge, labels string) float64 {
		t.Helper()

		v, ok := gauge.get(labels)
		require.True(t, ok, labels)

		return v
	}

	require.Eventually(t, func() bool {
		up, _ := registry.peers.get("state=up;")
		down, _ := registry.peers.get("state=down;")

		return up == 1 && down == 1
	}, 10*time.Second, 10*time.Millisecond)

	assert.InDelta(t, 1, value(registry.linkUp, "peer=srv1=transport=quic;"), 0)
	assert.InDelta(t, 0, value(registry.linkUp, "peer=srv1=transport=tcp;"), 0)

	require.Eventually(t, func() bool {
		rtt, _ := registry.linkRTT.get("peer=srv1;")
		return rtt > 0
	}, 10*time.Second, 10*time.Millisecond)

	// The address nobody answers at has no name to be a series: it only counts among the links that are down.
	assert.Equal(t, []string{"peer=srv1=transport=quic;", "peer=srv1=transport=tcp;"}, registry.linkUp.series())

	first.stop()

	require.Eventually(t, func() bool {
		down, _ := registry.peers.get("state=down;")
		return down == 2
	}, 10*time.Second, 10*time.Millisecond)

	assert.InDelta(t, 0, value(registry.linkUp, "peer=srv1=transport=quic;"), 0)
	assert.InDelta(t, 0, value(registry.linkRTT, "peer=srv1;"), 0)
}
