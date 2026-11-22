package mesh

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
)

func satelliteOver(transport string) func(conf *cluster.Configuration) {
	return func(conf *cluster.Configuration) {
		asSatellite(conf)
		conf.Mesh.Transport = transport
	}
}

func TestMesh_satellite_linkBackOverTCP(t *testing.T) {
	central := newTestNode(t, "srv1", testToken, nil)

	// The satellite only gets out over TCP: the central cannot reach it back over QUIC, and uses the link it opens.
	satellite := newTestNode(t, "casa1", testToken, satelliteOver(cluster.MeshTransportTCP), central.address)
	satellite.owns(named("casa1"), 0)

	handler := central.entry()
	assert.Equal(t, "casa1", ask(t, handler, "198.51.100.7:1000"))

	link, ok := central.linkTo("casa1")
	require.True(t, ok)
	assert.True(t, link.Up)
	assert.Equal(t, TransportTCPReverse, link.Transport)
	assert.Equal(t, []string{sourceSatellite}, link.Sources)

	// The link back is lost: the satellite opens another one, and the central uses it.
	central.mesh.mu.Lock()
	back := central.mesh.byNode["casa1"].link
	central.mesh.mu.Unlock()

	back.current().Close()

	require.Eventually(t, func() bool { return answers(handler) == "casa1" }, 10*time.Second, 10*time.Millisecond)

	// UDP travels in the stream of the flow. The routers of the satellite are its UDP one only from here on.
	satellite.ownsUDP(udpNamed("casa1", nil), 0)

	client := udpClient(t, time.Minute, central.udpEntry())
	assert.Equal(t, "casa1:ping", string(exchange(t, client, []byte("ping"))))

	// The satellite goes away: the central forgets it.
	satellite.stop()

	require.Eventually(t, func() bool {
		_, known := central.linkTo("casa1")
		return !known
	}, 15*time.Second, 10*time.Millisecond)
}

func TestMesh_satellite_noQUICBack(t *testing.T) {
	// The central cannot reach the satellite back over QUIC, though the link of the satellite comes over QUIC:
	// e.g. a firewall that lets the replies of a QUIC connection through, and nothing else.
	central := newIdleTestNode(t, "srv1", testToken, nil)
	central.configure = func(m *Mesh) { m.quicDialTimeout = time.Nanosecond }
	central.start()

	satellite := newTestNode(t, "casa1", testToken, asSatellite, central.address)
	satellite.owns(named("casa1"), 0)

	satellite.waitLinked("srv1")

	out, _ := satellite.linkTo("srv1")
	assert.Equal(t, TransportQUIC, out.Transport)

	// After the time it gives the central to reach it back over QUIC, the satellite opens a link over TCP.
	handler := central.entry()
	assert.Equal(t, "casa1", ask(t, handler, "198.51.100.7:1000"))

	back, ok := central.linkTo("casa1")
	require.True(t, ok)
	assert.Equal(t, TransportTCPReverse, back.Transport)
}

func TestMesh_satellite_prefersQUIC(t *testing.T) {
	central := newTestNode(t, "srv1", testToken, nil)
	satellite := newTestNode(t, "casa1", testToken, asSatellite, central.address)

	require.Eventually(t, func() bool {
		link, ok := central.linkTo("casa1")
		return ok && link.Up && link.Transport == TransportQUIC
	}, 10*time.Second, 10*time.Millisecond)

	// The central reaches the satellite back over QUIC: the satellite opens no link over TCP, and one that arrives
	// anyway is refused.
	time.Sleep(3 * satellite.mesh.reverseGrace)

	link, _ := central.linkTo("casa1")
	assert.Equal(t, TransportQUIC, link.Transport)
	assert.True(t, satellite.mesh.reachedBackOverQUIC("srv1"))

	require.Error(t, satellite.mesh.serveReverse(t.Context(), central.address, "srv1"))

	link, _ = central.linkTo("casa1")
	assert.Equal(t, TransportQUIC, link.Transport)
	assert.True(t, link.Up)
}

func TestMesh_reverse_refusesUnknownInstances(t *testing.T) {
	central := newTestNode(t, "srv1", testToken, nil)

	// An instance that did not introduce itself as a satellite cannot hand the central a link back.
	member := newTestNode(t, "srv2", testToken, nil)

	require.Error(t, member.mesh.serveReverse(t.Context(), central.address, "srv1"))

	_, known := central.linkTo("srv2")
	assert.False(t, known)
}
