package mesh

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
)

func Test_compareEntryPoints(t *testing.T) {
	ours := map[string]EntryPointSummary{
		"web":       {Protocol: "tcp", Middlewares: []string{"waf@file", "auth@file"}},
		"websecure": {Protocol: "tcp", Fingerprint: true},
		"dns":       {Protocol: "udp"},
		"admin":     {Protocol: "tcp"},
	}

	theirs := map[string]EntryPointSummary{
		// The same middlewares in another order run in another order.
		"web":       {Protocol: "tcp", Middlewares: []string{"auth@file", "waf@file"}},
		"websecure": {Protocol: "tcp"},
		"dns":       {Protocol: "tcp"},
		"metrics":   {Protocol: "tcp"},
	}

	// An entry point that only one of the instances has is not a difference.
	assert.Equal(t, []string{
		"dns: protocol udp here, tcp there",
		"web: middlewares [waf@file, auth@file] here, [auth@file, waf@file] there",
		"websecure: fingerprint on here, off there",
	}, compareEntryPoints(ours, theirs))

	assert.Empty(t, compareEntryPoints(ours, ours))
}

func TestMesh_configMismatch(t *testing.T) {
	registry := newRecordingRegistry()

	first := newIdleTestNode(t, "srv1", testToken, nil)
	first.entryPoints = map[string]EntryPointSummary{
		"web": {Protocol: "tcp", Middlewares: []string{"waf@file"}},
		"dns": {Protocol: "udp"},
	}
	first.start()

	second := newIdleTestNode(t, "srv2", testToken, nil, first.address)
	second.registry = registry
	second.entryPoints = map[string]EntryPointSummary{
		"web": {Protocol: "tcp"},
		"dns": {Protocol: "udp"},
	}
	second.start()

	expected := []string{"web: middlewares [] here, [waf@file] there"}

	// The instance that opens a link compares what it is answered, and the other one what it is told.
	require.Eventually(t, func() bool { return len(second.mesh.mismatchOf("srv1")) == 1 }, 10*time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool { return len(first.mesh.mismatchOf("srv2")) == 1 }, 10*time.Second, 10*time.Millisecond)

	assert.Equal(t, expected, second.mesh.mismatchOf("srv1"))
	assert.Equal(t, []string{"web: middlewares [waf@file] here, [] there"}, first.mesh.mismatchOf("srv2"))

	value, ok := registry.mismatch.get("peer=srv1;")
	require.True(t, ok)
	assert.InDelta(t, 1, value, 0)

	// The second instance restarts with the same entry points: the difference is gone.
	second.stop()

	same := newIdleTestNode(t, "srv2", testToken, func(conf *cluster.Configuration) { conf.Mesh.Address = second.address }, first.address)
	same.registry = registry
	same.entryPoints = first.entryPoints
	same.start()

	require.Eventually(t, func() bool {
		value, _ := registry.mismatch.get("peer=srv1;")
		return value == 0 && len(same.mesh.mismatchOf("srv1")) == 0
	}, 10*time.Second, 10*time.Millisecond)
}

func TestMesh_routeMetrics(t *testing.T) {
	registry := newRecordingRegistry()

	owner := newTestNode(t, "srv1", testToken, nil)

	ingress := newIdleTestNode(t, "srv2", testToken, nil, owner.address)
	ingress.registry = registry
	ingress.start()

	owner.mesh.Transform(t.Context(), testConfigurations())

	require.Eventually(t, func() bool {
		value, _ := registry.routes.get("peer=srv1=kind=http;")
		return value == 3
	}, 10*time.Second, 10*time.Millisecond)

	tcp, _ := registry.routes.get("peer=srv1=kind=tcp;")
	udp, _ := registry.routes.get("peer=srv1=kind=udp;")
	assert.InDelta(t, 1, tcp, 0)
	assert.InDelta(t, 1, udp, 0)

	// The routes of an instance leave with its link.
	owner.stop()

	require.Eventually(t, func() bool {
		value, _ := registry.routes.get("peer=srv1=kind=http;")
		return value == 0
	}, 10*time.Second, 10*time.Millisecond)

	// The instance counts the routers it announces.
	ingress.mesh.Transform(t.Context(), testConfigurations())

	exported, _ := registry.exported.get("kind=http;")
	assert.InDelta(t, 3, exported, 0)
}

func TestMesh_udpPacketMetrics(t *testing.T) {
	for _, transport := range transports() {
		t.Run(transport, func(t *testing.T) {
			ingressRegistry := newRecordingRegistry()
			ownerRegistry := newRecordingRegistry()

			owner := newIdleTestNode(t, "srv1", testToken, withTransport(transport))
			owner.registry = ownerRegistry
			owner.start()
			owner.ownsUDP(udpNamed("srv1", nil), 0)

			ingress := newIdleTestNode(t, "srv2", testToken, withTransport(transport), owner.address)
			ingress.registry = ingressRegistry
			ingress.start()

			client := udpClient(t, 300*time.Millisecond, ingress.udpEntry())

			// One packet that fits in a datagram, one that does not.
			assert.Equal(t, "srv1:a", string(exchange(t, client, []byte("a"))))

			large := bytes.Repeat([]byte("b"), 4000)
			assert.Equal(t, append([]byte("srv1:"), large...), exchange(t, client, large))

			// The packets are counted when the session ends.
			small, inStream := 1.0, 1.0
			if transport == cluster.MeshTransportTCP {
				small, inStream = 0, 2
			}

			require.Eventually(t, func() bool {
				sent, _ := ingressRegistry.packets.get("peer=srv1=path=stream=direction=sent;")
				return sent == inStream
			}, 10*time.Second, 10*time.Millisecond)

			datagrams, _ := ingressRegistry.packets.get("peer=srv1=path=datagram=direction=sent;")
			assert.InDelta(t, small, datagrams, 0)

			received, _ := ingressRegistry.packets.get("peer=srv1=path=stream=direction=received;")
			assert.InDelta(t, inStream, received, 0)

			require.Eventually(t, func() bool {
				received, _ := ownerRegistry.packets.get("peer=srv2=path=stream=direction=received;")
				return received == inStream
			}, 10*time.Second, 10*time.Millisecond)
		})
	}
}
