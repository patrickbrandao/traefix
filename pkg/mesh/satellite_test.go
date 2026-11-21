package mesh

import (
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

	// The other instances do not dial the satellite, nor tell each other about it.
	time.Sleep(500 * time.Millisecond)

	first.waitLinked("srv2")
	second.waitLinked("srv1")

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

	// An instance that lists the address of a satellite among its peers drops it, and does not dial it again.
	member := newTestNode(t, "srv1", testToken, nil, satellite.address)

	require.Eventually(t, func() bool { return len(member.mesh.Status().Links) == 0 }, 10*time.Second, 10*time.Millisecond)

	member.mesh.mu.Lock()
	_, marked := member.mesh.satelliteAddresses[satellite.address]
	member.mesh.mu.Unlock()
	assert.True(t, marked)

	member.mesh.learn(sourcePeers, "", []string{satellite.address})
	assert.Empty(t, member.mesh.Status().Links)

	// A satellite does not link to another one.
	other := newTestNode(t, "casa2", testToken, asSatellite, satellite.address)

	require.Eventually(t, func() bool { return len(other.mesh.Status().Links) == 0 }, 10*time.Second, 10*time.Millisecond)

	other.mesh.mu.Lock()
	_, marked = other.mesh.satelliteAddresses[satellite.address]
	other.mesh.mu.Unlock()
	assert.True(t, marked)

	_, counted := satellite.inboundRole("casa2")
	assert.False(t, counted)
}

func TestMesh_satellite_exports(t *testing.T) {
	central := newTestNode(t, "srv1", testToken, nil)
	satellite := newTestNode(t, "casa1", testToken, asSatellite, central.address)

	satellite.waitLinked("srv1")

	// The satellite announces its routers like any instance: the way back that lets the central follow them
	// comes with a later stage.
	satellite.mesh.Transform(t.Context(), dynamic.Configurations{
		"docker": {HTTP: &dynamic.HTTPConfiguration{Routers: map[string]*dynamic.Router{
			"nas": {Rule: "Host(`nas.example.com`)", EntryPoints: []string{"websecure"}, Service: "nas"},
		}}},
	})

	assert.Equal(t, []string{"http/nas@docker"}, names(satellite.mesh.exported))
}
