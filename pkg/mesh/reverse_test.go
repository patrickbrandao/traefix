package mesh

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
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

func TestMesh_satellite_metricsAndAPI(t *testing.T) {
	registry := newRecordingRegistry()

	store := &fakeCertificates{certificates: map[string][]Certificate{}}
	store.set("le", testCertificate("nas", time.Now().Add(30*24*time.Hour), "nas.example.com"))

	central := newIdleTestNode(t, "srv1", testToken, nil)
	central.registry = registry
	central.start()
	central.mesh.SetCertificateSource(store.source)

	overTCP := newTestNode(t, "casa1", testToken, satelliteOver(cluster.MeshTransportTCP), central.address)
	overTCP.mesh.Transform(t.Context(), dynamic.Configurations{
		"docker": {HTTP: &dynamic.HTTPConfiguration{Routers: map[string]*dynamic.Router{
			"nas": {Rule: "Host(`nas.example.com`)", EntryPoints: []string{"websecure"}, Service: "nas", TLS: &dynamic.RouterTLSConfig{CertResolver: "le"}},
		}}},
	})

	overQUIC := newTestNode(t, "casa2", testToken, asSatellite, central.address)

	gauge := func(g *recordingGauge, labels string) float64 {
		v, _ := g.get(labels)
		return v
	}

	// The links back to the satellites count apart from the instances the central looks for.
	require.Eventually(t, func() bool {
		return gauge(registry.satellites, "state=up;") == 2 &&
			gauge(registry.linkUp, "peer=casa1=transport=tcp-reverse;") == 1 &&
			gauge(registry.linkUp, "peer=casa2=transport=quic;") == 1 &&
			gauge(registry.delivered, "peer=casa1;") == 1
	}, 10*time.Second, 10*time.Millisecond)

	assert.InDelta(t, 0, gauge(registry.peers, "state=up;"), 0)
	assert.InDelta(t, 0, gauge(registry.delivered, "peer=casa2;"), 0)

	// The API tells where the link of each satellite comes from, and how the central reaches it back.
	recorder := httptest.NewRecorder()
	central.mesh.APIHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/mesh", nil))

	var status apiStatus
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&status))

	links := map[string]apiLink{}
	for _, link := range status.Links {
		links[link.NodeID] = link
	}

	assert.Equal(t, RoleSatellite, links["casa1"].Role)
	assert.Equal(t, TransportTCPReverse, links["casa1"].Transport)
	assert.Empty(t, links["casa1"].Observed)
	assert.Equal(t, RoleSatellite, links["casa2"].Role)
	assert.Equal(t, TransportQUIC, links["casa2"].Transport)
	assert.Equal(t, overQUIC.address, links["casa2"].Observed)

	// A satellite that goes away leaves nothing up behind it.
	overTCP.stop()

	require.Eventually(t, func() bool {
		return gauge(registry.satellites, "state=up;") == 1 &&
			gauge(registry.linkUp, "peer=casa1=transport=tcp-reverse;") == 0 &&
			gauge(registry.delivered, "peer=casa1;") == 0
	}, 15*time.Second, 10*time.Millisecond)
}
