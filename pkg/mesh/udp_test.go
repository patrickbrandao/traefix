package mesh

import (
	"bytes"
	"crypto/rand"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/udp"
)

// ownsUDP makes the instance announce a UDP router named dns@docker, and serve it with the handler.
func (n *testNode) ownsUDP(handler udp.Handler, priority int) {
	n.t.Helper()

	n.mesh.SwitchHandlers(Handlers{UDP: map[string]udp.Handler{"dns@docker": handler}}, nil)
	n.mesh.Transform(n.t.Context(), dynamic.Configurations{
		"docker": {UDP: &dynamic.UDPConfiguration{Routers: map[string]*dynamic.UDPRouter{
			"dns": {EntryPoints: []string{"dns"}, Service: "dns", Mesh: &dynamic.RouterMesh{Priority: priority}},
		}}},
	})
}

// udpEntry returns the handler of the service that the instance made for the UDP router of the other instances.
func (n *testNode) udpEntry() udp.Handler {
	n.t.Helper()

	require.Eventually(n.t, func() bool {
		return n.mesh.tables.Load().udpTargets["dns-docker@mesh"] != nil
	}, 10*time.Second, 10*time.Millisecond)

	handler, err := n.mesh.BuildUDP(n.t.Context(), "dns-docker@mesh")
	require.NoError(n.t, err)
	require.NotNil(n.t, handler)

	return handler
}

func (n *testNode) udpCandidates() int {
	tables := n.mesh.tables.Load()

	if target := tables.udpTargets["dns-docker@mesh"]; target != nil {
		return len(target.candidates)
	}

	if target := tables.udpShared["dns@docker"]; target != nil {
		return len(target.candidates)
	}

	return 0
}

// udpClient starts a UDP entry point whose sessions the handler serves, and returns a client connected to it.
func udpClient(t *testing.T, timeout time.Duration, handler udp.Handler) *net.UDPConn {
	t.Helper()

	listener, err := udp.Listen(net.ListenConfig{}, "udp", "127.0.0.1:0", timeout)
	require.NoError(t, err)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			go handler.ServeUDP(conn)
		}
	}()

	client, err := net.DialUDP("udp", nil, listener.Addr().(*net.UDPAddr))
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.Close()
		_ = listener.Close()
	})

	return client
}

// exchange sends a packet and returns the answer, or an empty string when none comes.
func exchange(t *testing.T, client *net.UDPConn, payload []byte) []byte {
	t.Helper()

	_, err := client.Write(payload)
	require.NoError(t, err)

	// The first packet of a session waits for a flow to open: a loaded machine takes a while.
	require.NoError(t, client.SetReadDeadline(time.Now().Add(5*time.Second)))

	buf := make([]byte, maxPacketSize)

	n, err := client.Read(buf)
	if err != nil {
		return nil
	}

	return buf[:n]
}

// udpNamed is the service of a UDP router: it answers each packet with its name and the packet.
// The addresses of the sessions it serves go to the channel, when there is one.
func udpNamed(name string, remotes chan<- net.Addr) udp.Handler {
	return udp.HandlerFunc(func(conn *udp.Conn) {
		defer func() { _ = conn.Close() }()

		if remotes != nil {
			remotes <- conn.RemoteAddr()
		}

		buf := make([]byte, maxPacketSize)

		for {
			n, err := conn.Read(buf)
			if err != nil {
				return
			}

			if _, err := conn.Write(append([]byte(name+":"), buf[:n]...)); err != nil {
				return
			}
		}
	})
}

func TestMesh_forwardUDP(t *testing.T) {
	for _, transport := range transports() {
		t.Run(transport, func(t *testing.T) {
			remotes := make(chan net.Addr, 1)

			owner := newTestNode(t, "srv1", testToken, withTransport(transport))
			owner.ownsUDP(udpNamed("srv1", remotes), 0)

			ingress := newTestNode(t, "srv2", testToken, withTransport(transport), owner.address)

			client := udpClient(t, time.Minute, ingress.udpEntry())

			// Small packets go in datagrams over QUIC, the ones that do not fit go in the stream of the flow.
			// macOS sends no UDP packet larger than 9216 bytes by default.
			for _, size := range []int{1, 100, 1200, 1400, 9000} {
				payload := make([]byte, size)
				_, err := rand.Read(payload)
				require.NoError(t, err)

				answer := exchange(t, client, payload)
				require.NotNil(t, answer, "no answer for a packet of %d bytes", size)
				assert.True(t, bytes.Equal(append([]byte("srv1:"), payload...), answer), "packet of %d bytes", size)
			}

			// The service of the instance that owns the router sees the address of the client, and one session for it.
			assert.Equal(t, client.LocalAddr().String(), (<-remotes).String())
		})
	}
}

func TestMesh_forwardUDP_idleTimeout(t *testing.T) {
	served := make(chan struct{}, 2)

	owner := newTestNode(t, "srv1", testToken, nil)
	owner.ownsUDP(udp.HandlerFunc(func(conn *udp.Conn) {
		udpNamed("srv1", nil).ServeUDP(conn)
		served <- struct{}{}
	}), 0)

	ingress := newTestNode(t, "srv2", testToken, nil, owner.address)

	client := udpClient(t, time.Second, ingress.udpEntry())

	assert.Equal(t, "srv1:a", string(exchange(t, client, []byte("a"))))

	// Without a packet for the timeout of the entry point, the session ends on both instances.
	select {
	case <-served:
	case <-time.After(5 * time.Second):
		t.Fatal("the session should have ended on the instance that owns the router")
	}

	// The next packet starts another session.
	assert.Equal(t, "srv1:b", string(exchange(t, client, []byte("b"))))
}

func TestMesh_forwardUDP_unknownRouter(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	first.ownsUDP(udpNamed("srv1", nil), 10)

	second := newTestNode(t, "srv3", testToken, nil, first.address)
	second.ownsUDP(udpNamed("srv3", nil), 0)

	ingress := newTestNode(t, "srv2", testToken, nil, first.address)
	handler := ingress.udpEntry()

	require.Eventually(t, func() bool { return ingress.udpCandidates() == 2 }, 10*time.Second, 10*time.Millisecond)

	assert.Equal(t, "srv1:a", string(exchange(t, udpClient(t, time.Minute, handler), []byte("a"))))

	// The instance that should serve the session lost the router: it refuses the flow, and the session goes to the
	// other one, with the packet that started it.
	first.mesh.routing.Store(&routing{})

	assert.Equal(t, "srv3:b", string(exchange(t, udpClient(t, time.Minute, handler), []byte("b"))))
}

func TestMesh_forwardUDP_noInstance(t *testing.T) {
	owner := newTestNode(t, "srv1", testToken, nil)
	owner.ownsUDP(udpNamed("srv1", nil), 0)

	ingress := newTestNode(t, "srv2", testToken, nil, owner.address)
	handler := ingress.udpEntry()

	assert.Equal(t, "srv1:a", string(exchange(t, udpClient(t, time.Minute, handler), []byte("a"))))

	owner.stop()

	// Nothing can serve the session: its packets are dropped, as UDP does.
	require.Eventually(t, func() bool {
		return exchange(t, udpClient(t, time.Minute, handler), []byte("b")) == nil
	}, 10*time.Second, 10*time.Millisecond)
}

func TestMesh_sharedUDPRoute(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv2", testToken, nil, first.address)

	first.ownsUDP(udpNamed("srv1", nil), 0)
	second.ownsUDP(udpNamed("srv2", nil), 0)

	decided := first.mesh.DecideUDP("dns@docker", udpNamed("srv1", nil))

	require.Eventually(t, func() bool { return first.udpCandidates() == 1 }, 10*time.Second, 10*time.Millisecond)

	// With the default priority, an instance serves the sessions it receives.
	assert.Equal(t, "srv1:a", string(exchange(t, udpClient(t, time.Minute, decided), []byte("a"))))

	// The other instance takes the priority: the sessions go to it.
	second.ownsUDP(udpNamed("srv2", nil), 10)

	require.Eventually(t, func() bool {
		return string(exchange(t, udpClient(t, time.Minute, decided), []byte("b"))) == "srv2:b"
	}, 10*time.Second, 10*time.Millisecond)
}

func TestMesh_buildConfiguration_udpEntryPoints(t *testing.T) {
	node := newTestNode(t, "srv1", testToken, nil)
	node.mesh.SetEntryPoints([]string{"web"}, []string{"dns", "syslog", "ntp"}, []string{"web"})

	// The instance has a UDP router of its own on one entry point, which it does not announce.
	node.mesh.Transform(t.Context(), dynamic.Configurations{
		"file": {UDP: &dynamic.UDPConfiguration{Routers: map[string]*dynamic.UDPRouter{
			"local": {EntryPoints: []string{"syslog"}, Service: "local", Mesh: &dynamic.RouterMesh{Export: new(false)}},
		}}},
	})

	node.mesh.mu.Lock()
	node.mesh.imported = map[string]*importedTable{
		"srv2": {routes: []Route{
			{Kind: KindUDP, Name: "a@docker", EntryPoints: []string{"dns"}},
			// The same entry point as the router before it: an entry point has one UDP router only.
			{Kind: KindUDP, Name: "b@docker", EntryPoints: []string{"dns", "ntp"}},
			{Kind: KindUDP, Name: "c@docker", EntryPoints: []string{"syslog"}},
			{Kind: KindUDP, Name: "d@docker", EntryPoints: []string{"dns"}},
		}},
	}
	node.mesh.mu.Unlock()

	conf, report := node.mesh.buildConfiguration()

	assert.Equal(t, map[string]*dynamic.UDPRouter{
		"a-docker": {EntryPoints: []string{"dns"}, Service: "a-docker"},
		"b-docker": {EntryPoints: []string{"ntp"}, Service: "b-docker"},
	}, conf.UDP.Routers)

	status := map[string]string{}
	for _, imported := range report {
		status[imported.Name] = imported.Status
	}

	assert.Equal(t, map[string]string{"a@docker": ImportImported, "b@docker": ImportImported, "c@docker": ImportSkipped, "d@docker": ImportSkipped}, status)
}

func TestMesh_healthUDP(t *testing.T) {
	owner := newTestNode(t, "srv1", testToken, nil)
	owner.ownsUDP(udpNamed("srv1", nil), 0)

	service := &runtime.UDPServiceInfo{UDPService: &dynamic.UDPService{LoadBalancer: &dynamic.UDPServersLoadBalancer{}}}
	conf := &runtime.Configuration{UDPServices: map[string]*runtime.UDPServiceInfo{"dns@docker": service}}

	// A load balancer without servers has nothing to send the packets to.
	owner.mesh.SwitchHandlers(Handlers{UDP: map[string]udp.Handler{"dns@docker": udpNamed("srv1", nil)}}, conf)

	assert.True(t, owner.mesh.isLocalDown(routeKey(KindUDP, "dns@docker")))

	service.LoadBalancer.Servers = []dynamic.UDPServer{{Address: "10.0.0.1:53"}}
	owner.mesh.checkHealth()

	assert.False(t, owner.mesh.isLocalDown(routeKey(KindUDP, "dns@docker")))
}

func Test_udpServiceUp(t *testing.T) {
	conf := &runtime.Configuration{UDPServices: map[string]*runtime.UDPServiceInfo{
		"up@docker":       {UDPService: &dynamic.UDPService{LoadBalancer: &dynamic.UDPServersLoadBalancer{Servers: []dynamic.UDPServer{{Address: "10.0.0.1:53"}}}}},
		"empty@docker":    {UDPService: &dynamic.UDPService{LoadBalancer: &dynamic.UDPServersLoadBalancer{}}},
		"weighted@docker": {UDPService: &dynamic.UDPService{Weighted: &dynamic.UDPWeightedRoundRobin{}}},
	}}

	assert.True(t, udpServiceUp(conf, "up@docker"))
	assert.False(t, udpServiceUp(conf, "empty@docker"))
	assert.True(t, udpServiceUp(conf, "weighted@docker"))
	assert.False(t, udpServiceUp(conf, "missing@docker"))
	assert.True(t, udpServiceUp(nil, "missing@docker"))
}

func TestMesh_LocalUDPRoute(t *testing.T) {
	node := newTestNode(t, "srv1", testToken, nil)

	node.mesh.Transform(t.Context(), dynamic.Configurations{
		"docker": {UDP: &dynamic.UDPConfiguration{Routers: map[string]*dynamic.UDPRouter{
			"dns":     {EntryPoints: []string{"dns"}, Service: "dns"},
			"private": {EntryPoints: []string{"dns"}, Service: "private", Mesh: &dynamic.RouterMesh{Export: new(false)}},
		}}},
	})

	assert.True(t, node.mesh.LocalUDPRoute("dns@docker"))
	assert.False(t, node.mesh.LocalUDPRoute("private@docker"))
	assert.Len(t, node.mesh.LocalUDPRouters(), 1)
}
