package mesh

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
)

// asSatellite makes an instance of the tests a satellite: it has no address to tell.
func asSatellite(conf *cluster.Configuration) {
	conf.Mesh.Satellite = true
	conf.Mesh.Advertise = nil
}

func (n *testNode) inboundRole(nodeID string) (string, bool) {
	for _, inbound := range n.mesh.Status().Inbound {
		if inbound.NodeID == nodeID {
			return inbound.Role, true
		}
	}

	return "", false
}

func TestMesh_satellite(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv2", testToken, nil, first.address)

	first.waitLinked("srv2")
	second.waitLinked("srv1")

	satellite := newTestNode(t, "casa1", testToken, asSatellite, first.address)

	// The satellite links to the instance it knows, and to the one it learns from it.
	satellite.waitLinked("srv1", "srv2")

	status := satellite.mesh.Status()
	assert.Equal(t, RoleSatellite, status.Role)
	assert.Empty(t, status.Advertise)

	require.Eventually(t, func() bool {
		role, ok := first.inboundRole("casa1")
		return ok && role == RoleSatellite
	}, 10*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		role, ok := second.inboundRole("casa1")
		return ok && role == RoleSatellite
	}, 10*time.Second, 10*time.Millisecond)

	// Both reach the satellite back, through the link it opened, and do not tell each other about it.
	first.waitLinked("srv2", "casa1")
	second.waitLinked("srv1", "casa1")

	link, _ := first.linkTo("casa1")
	assert.Equal(t, []string{sourceSatellite}, link.Sources)
	assert.Equal(t, RoleSatellite, link.Role)
	assert.Equal(t, TransportQUIC, link.Transport)

	for _, peer := range first.mesh.knownPeers() {
		assert.NotEqual(t, "casa1", peer.NodeID)
	}

	assert.Equal(t, RoleMember, first.mesh.Status().Role)
	assert.Equal(t, RoleSatellite, roleOf(hello{Role: RoleSatellite}))
	assert.Equal(t, RoleMember, roleOf(hello{}))

	// The satellite imports none of the routers of the others.
	first.mesh.Transform(t.Context(), testConfigurations())

	require.Eventually(t, func() bool { return len(second.mesh.importedFrom("srv1").routes) > 0 }, 10*time.Second, 10*time.Millisecond)

	time.Sleep(500 * time.Millisecond)

	assert.Nil(t, satellite.mesh.importedFrom("srv1"))

	conf, report := satellite.mesh.buildConfiguration()
	assert.Empty(t, conf.HTTP.Routers)
	assert.Empty(t, report)
}

func TestMesh_satellite_notDialed(t *testing.T) {
	satellite := newTestNode(t, "casa1", testToken, asSatellite, "127.0.0.1:1")

	// An instance that lists the address of a satellite among its peers drops it, and does not dial it there again.
	member := newTestNode(t, "srv1", testToken, nil, satellite.address)

	marked := func(n *testNode) bool {
		n.mesh.mu.Lock()
		defer n.mesh.mu.Unlock()

		_, ok := n.mesh.satelliteAddresses[satellite.address]

		return ok
	}

	require.Eventually(t, func() bool { return marked(member) }, 10*time.Second, 10*time.Millisecond)

	// The satellite learned the instance when it was dialed, and links to it: the instance reaches it back then,
	// as any satellite. Without a NAT in between, the way back is the address of the list, and the link is still
	// the one of a satellite, never one the list opened.
	onlyReachedBack := func() bool {
		for _, link := range member.mesh.Status().Links {
			if !slices.Equal(link.Sources, []string{sourceSatellite}) {
				return false
			}
		}

		return true
	}

	require.Eventually(t, onlyReachedBack, 10*time.Second, 10*time.Millisecond)

	member.mesh.learn(sourcePeers, "", []string{satellite.address})
	assert.True(t, onlyReachedBack())

	// A satellite does not link to another one, and the other one does not count it among the instances linked to it.
	other := newTestNode(t, "casa2", testToken, asSatellite, satellite.address)

	require.Eventually(t, func() bool { return marked(other) }, 10*time.Second, 10*time.Millisecond)

	_, linked := other.linkTo("casa1")
	assert.False(t, linked)

	_, counted := satellite.inboundRole("casa2")
	assert.False(t, counted)
}

func TestMesh_satellite_exports(t *testing.T) {
	central := newTestNode(t, "srv1", testToken, nil)
	satellite := newTestNode(t, "casa1", testToken, asSatellite, central.address)

	satellite.waitLinked("srv1")

	// The satellite announces its routers like any instance, and the central follows them through the link back.
	satellite.mesh.Transform(t.Context(), dynamic.Configurations{
		"docker": {HTTP: &dynamic.HTTPConfiguration{Routers: map[string]*dynamic.Router{
			"nas": {Rule: "Host(`nas.example.com`)", EntryPoints: []string{"websecure"}, Service: "nas"},
		}}},
	})

	assert.Equal(t, []string{"http/nas@docker"}, names(satellite.mesh.exported))

	central.waitRouters("nas-docker")
}

func TestMesh_satellite_behindNAT(t *testing.T) {
	central := newTestNode(t, "srv1", testToken, nil)
	nat := newTestNAT(t, central.address)

	satellite := newTestNode(t, "casa1", testToken, asSatellite, nat.address())
	satellite.owns(named("casa1"), 0)

	// The central sees the satellite at the address of the NAT, dials it back there, and serves its router.
	handler := central.entry()
	assert.Equal(t, "casa1", ask(t, handler, "198.51.100.7:1000"))

	link, ok := central.linkTo("casa1")
	require.True(t, ok)
	assert.True(t, link.Up)
	assert.Equal(t, []string{link.Address}, nat.mapped())
	assert.NotEqual(t, satellite.address, link.Address)

	// The NAT only lets back what comes from the central: another instance does not reach the satellite there.
	other := newTestIdentity(t, "srv9", testToken)

	_, err := dialQUIC(t.Context(), other, link.Address, 500*time.Millisecond)
	require.Error(t, err)
	assert.Positive(t, nat.dropped.Load())

	// The NAT maps the satellite to another port: the central follows, and serves the router again.
	before := link.Address
	nat.remap()

	require.Eventually(t, func() bool {
		link, ok := central.linkTo("casa1")
		return ok && link.Up && link.Address != before
	}, 15*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool { return answers(handler) == "casa1" }, 10*time.Second, 10*time.Millisecond)

	// The satellite goes away: the central gives it up, and forgets it.
	satellite.stop()

	require.Eventually(t, func() bool {
		_, known := central.linkTo("casa1")
		return !known
	}, 15*time.Second, 10*time.Millisecond)
}

// answers returns what the handler answers to a request, or nothing when it does not serve it.
func answers(handler http.Handler) string {
	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/", nil)
	req.RemoteAddr = "198.51.100.7:1000"

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		return ""
	}

	return recorder.Body.String()
}
