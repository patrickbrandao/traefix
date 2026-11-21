package mesh

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/tcp"
)

// ownsTCP makes the instance announce a TCP router named pg@docker, and serve it with the handler.
func (n *testNode) ownsTCP(handler tcp.Handler, priority int) {
	n.t.Helper()

	n.mesh.SwitchHandlers(nil, map[string]tcp.Handler{"pg@docker": handler}, nil)
	n.mesh.Transform(n.t.Context(), dynamic.Configurations{
		"docker": {TCP: &dynamic.TCPConfiguration{Routers: map[string]*dynamic.TCPRouter{
			"pg": {Rule: "HostSNI(`*`)", EntryPoints: []string{"web"}, Service: "pg", Mesh: &dynamic.RouterMesh{Priority: priority}},
		}}},
	})
}

// waitTCPRouters returns the TCP routers of the next configuration the mesh gives, skipping the ones before the expected one.
func (n *testNode) waitTCPRouters(expected ...string) map[string]*dynamic.TCPRouter {
	n.t.Helper()

	var last map[string]*dynamic.TCPRouter

	require.Eventually(n.t, func() bool {
		for {
			select {
			case message := <-n.configurations:
				last = message.Configuration.TCP.Routers
			default:
				if last == nil || len(last) != len(expected) {
					return false
				}

				for _, name := range expected {
					if _, ok := last[name]; !ok {
						return false
					}
				}

				return true
			}
		}
	}, 10*time.Second, 10*time.Millisecond, "expected the TCP routers %v, got %v", expected, last)

	return last
}

// tcpEntry returns the handler of the service that the instance made for the TCP router of the other instances.
func (n *testNode) tcpEntry() tcp.Handler {
	n.t.Helper()

	n.waitTCPRouters("pg-docker")

	handler, err := n.mesh.BuildTCP(n.t.Context(), "pg-docker@mesh")
	require.NoError(n.t, err)
	require.NotNil(n.t, handler)

	return handler
}

// tcpCandidates returns how many instances the instance knows to own the TCP router of the tests, as a route of its own
// or as one it shares.
func (n *testNode) tcpCandidates() int {
	tables := n.mesh.tables.Load()

	if target := tables.tcpTargets["pg-docker@mesh"]; target != nil {
		return len(target.candidates)
	}

	if target := tables.tcpShared["pg@docker"]; target != nil {
		return len(target.candidates)
	}

	return 0
}

// connect gives the handler the connection of a client, and returns the end of the client.
func connect(t *testing.T, handler tcp.Handler) *net.TCPConn {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	served := make(chan struct{})

	go func() {
		defer close(served)

		conn, err := listener.Accept()
		if err != nil {
			return
		}

		handler.ServeTCP(conn.(*net.TCPConn))
	}()

	client, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.Close()
		_ = listener.Close()

		<-served
	})

	require.NoError(t, client.SetDeadline(time.Now().Add(20*time.Second)))

	return client.(*net.TCPConn)
}

// tcpNamed is the server of a TCP router: it tells its name, and closes.
func tcpNamed(name string) tcp.Handler {
	return tcp.HandlerFunc(func(conn tcp.WriteCloser) {
		defer func() { _ = conn.Close() }()

		_, _ = io.WriteString(conn, name)
	})
}

// askTCP opens a connection of a client to the handler, and returns what the server says before it closes.
func askTCP(t *testing.T, handler tcp.Handler) string {
	t.Helper()

	answer, err := io.ReadAll(connect(t, handler))
	require.NoError(t, err)

	return string(answer)
}

func TestMesh_forwardTCP(t *testing.T) {
	for _, transport := range transports() {
		t.Run(transport, func(t *testing.T) {
			type addresses struct{ remote, local net.Addr }

			seen := make(chan addresses, 1)

			// The server speaks first, echoes what it gets, and still answers once the client is done sending.
			owner := newTestNode(t, "srv1", testToken, withTransport(transport))
			owner.ownsTCP(tcp.HandlerFunc(func(conn tcp.WriteCloser) {
				defer func() { _ = conn.Close() }()

				seen <- addresses{remote: conn.RemoteAddr(), local: conn.LocalAddr()}

				if _, err := io.WriteString(conn, "hello\n"); err != nil {
					return
				}

				if _, err := io.Copy(conn, conn); err != nil {
					return
				}

				_, _ = io.WriteString(conn, "bye\n")
			}), 0)

			ingress := newTestNode(t, "srv2", testToken, withTransport(transport), owner.address)

			client := connect(t, ingress.tcpEntry())
			reader := bufio.NewReader(client)

			line, err := reader.ReadString('\n')
			require.NoError(t, err)
			assert.Equal(t, "hello\n", line)

			// The instance that owns the router sees the connection the client opened to the other instance.
			got := <-seen
			assert.Equal(t, client.LocalAddr().String(), got.remote.String())
			assert.Equal(t, client.RemoteAddr().String(), got.local.String())
			assert.IsType(t, &net.TCPAddr{}, got.remote)
			assert.IsType(t, &net.TCPAddr{}, got.local)

			// More than the windows of a stream, in both directions at once.
			payload := make([]byte, 1<<20)
			_, err = rand.Read(payload)
			require.NoError(t, err)

			written := make(chan error, 1)

			go func() {
				_, err := client.Write(payload)
				written <- err
			}()

			echoed := make([]byte, len(payload))
			_, err = io.ReadFull(reader, echoed)
			require.NoError(t, err)
			require.NoError(t, <-written)
			assert.True(t, bytes.Equal(payload, echoed))

			// The client is done sending: the server learns it, and what it still says reaches the client.
			require.NoError(t, client.CloseWrite())

			rest, err := io.ReadAll(reader)
			require.NoError(t, err)
			assert.Equal(t, "bye\n", string(rest))
		})
	}
}

func TestMesh_forwardTCP_serverStopsSending(t *testing.T) {
	for _, transport := range transports() {
		t.Run(transport, func(t *testing.T) {
			type result struct {
				data string
				err  error
			}

			received := make(chan result, 1)

			// What the services of Traefik do when a server is done sending: the client is told, and has a while to finish.
			owner := newTestNode(t, "srv1", testToken, withTransport(transport))
			owner.ownsTCP(tcp.HandlerFunc(func(conn tcp.WriteCloser) {
				defer func() { _ = conn.Close() }()

				_, _ = io.WriteString(conn, "done\n")
				_ = conn.CloseWrite()

				if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
					received <- result{err: err}
					return
				}

				data, err := io.ReadAll(conn)
				received <- result{data: string(data), err: err}
			}), 0)

			ingress := newTestNode(t, "srv2", testToken, withTransport(transport), owner.address)

			client := connect(t, ingress.tcpEntry())
			reader := bufio.NewReader(client)

			line, err := reader.ReadString('\n')
			require.NoError(t, err)
			assert.Equal(t, "done\n", line)

			if transport == cluster.MeshTransportQUIC {
				// Over QUIC the client learns at once that the server is done sending, and can still send.
				require.NoError(t, client.SetReadDeadline(time.Now().Add(500*time.Millisecond)))

				_, err = reader.ReadByte()
				require.ErrorIs(t, err, io.EOF)

				require.NoError(t, client.SetReadDeadline(time.Now().Add(20*time.Second)))
			}

			_, err = io.WriteString(client, "late\n")
			require.NoError(t, err)

			got := <-received
			assert.Equal(t, "late\n", got.data)
			require.ErrorIs(t, got.err, os.ErrDeadlineExceeded)

			// The server gave the connection up: the one of the client ends, though the client did not close it.
			_, err = io.ReadAll(reader)
			if err != nil {
				_, timedOut := errors.AsType[net.Error](err)
				assert.False(t, timedOut && errors.Is(err, os.ErrDeadlineExceeded))
			}
		})
	}
}

func TestMesh_forwardTCP_unknownRouter(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	first.ownsTCP(tcpNamed("srv1"), 10)

	second := newTestNode(t, "srv3", testToken, nil, first.address)
	second.ownsTCP(tcpNamed("srv3"), 0)

	ingress := newTestNode(t, "srv2", testToken, nil, first.address)
	handler := ingress.tcpEntry()

	require.Eventually(t, func() bool { return ingress.tcpCandidates() == 2 }, 10*time.Second, 10*time.Millisecond)

	require.Equal(t, "srv1", askTCP(t, handler))

	// The instance that should serve the connection lost the router and did not say so yet: it refuses the stream,
	// and the connection goes to the other one, as nothing of it was sent.
	first.mesh.routing.Store(&routing{})

	require.Equal(t, "srv3", askTCP(t, handler))

	// After a few refusals, the instance is left alone for the route.
	for range quarantineAfter {
		require.Equal(t, "srv3", askTCP(t, handler))
	}

	assert.True(t, ingress.mesh.quarantined(routeKey(KindTCP, "pg@docker"), "srv1"))
	assert.False(t, ingress.mesh.quarantined(routeKey(KindHTTP, "pg@docker"), "srv1"))
}

func TestMesh_forwardTCP_noInstance(t *testing.T) {
	owner := newTestNode(t, "srv1", testToken, nil)
	owner.ownsTCP(tcpNamed("srv1"), 0)

	ingress := newTestNode(t, "srv2", testToken, nil, owner.address)
	handler := ingress.tcpEntry()

	require.Equal(t, "srv1", askTCP(t, handler))

	owner.stop()

	// Nothing can serve the connection: it is closed, which is all TCP has to say so.
	require.Eventually(t, func() bool { return askTCP(t, handler) == "" }, 10*time.Second, 10*time.Millisecond)
}

func TestMesh_sharedTCPRoute(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv2", testToken, nil, first.address)

	first.ownsTCP(tcpNamed("srv1"), 0)
	second.ownsTCP(tcpNamed("srv2"), 0)

	decided := first.mesh.DecideTCP("pg@docker", tcpNamed("srv1"))

	require.Eventually(t, func() bool { return first.tcpCandidates() == 1 }, 10*time.Second, 10*time.Millisecond)

	// With the default priority, an instance serves the connections it receives.
	require.Equal(t, "srv1", askTCP(t, decided))

	// The other instance takes the priority: the connections go to it.
	second.ownsTCP(tcpNamed("srv2"), 10)

	require.Eventually(t, func() bool { return askTCP(t, decided) == "srv2" }, 10*time.Second, 10*time.Millisecond)

	// Without it, the instance serves them again.
	second.stop()

	require.Eventually(t, func() bool { return askTCP(t, decided) == "srv1" }, 10*time.Second, 10*time.Millisecond)
}

func TestMesh_healthTCP(t *testing.T) {
	owner := newTestNode(t, "srv1", testToken, nil)
	owner.ownsTCP(tcpNamed("srv1"), 0)

	ingress := newTestNode(t, "srv2", testToken, nil, owner.address)
	handler := ingress.tcpEntry()

	require.Equal(t, "srv1", askTCP(t, handler))

	service := &runtime.TCPServiceInfo{TCPService: &dynamic.TCPService{LoadBalancer: &dynamic.TCPServersLoadBalancer{Servers: []dynamic.TCPServer{{Address: "10.0.0.1:5432"}}}}}
	service.UpdateServerStatus("10.0.0.1:5432", runtime.StatusUp)

	conf := &runtime.Configuration{TCPServices: map[string]*runtime.TCPServiceInfo{"pg@docker": service}}

	owner.mesh.SwitchHandlers(nil, map[string]tcp.Handler{"pg@docker": tcpNamed("srv1")}, conf)

	require.Equal(t, "srv1", askTCP(t, handler))

	// The health check of the owner takes its only server down: it says so, and the connections stop coming.
	service.UpdateServerStatus("10.0.0.1:5432", runtime.StatusDown)
	owner.mesh.checkHealth()

	assert.True(t, owner.mesh.isLocalDown(routeKey(KindTCP, "pg@docker")))

	require.Eventually(t, func() bool { return askTCP(t, handler) == "" }, 10*time.Second, 10*time.Millisecond)

	// The server is back.
	service.UpdateServerStatus("10.0.0.1:5432", runtime.StatusUp)
	owner.mesh.checkHealth()

	require.Eventually(t, func() bool { return askTCP(t, handler) == "srv1" }, 10*time.Second, 10*time.Millisecond)
}

func Test_tcpServiceUp(t *testing.T) {
	loadBalancer := func(servers int, statuses ...string) *runtime.TCPServiceInfo {
		service := &runtime.TCPServiceInfo{TCPService: &dynamic.TCPService{LoadBalancer: &dynamic.TCPServersLoadBalancer{}}}

		for i := range servers {
			service.LoadBalancer.Servers = append(service.LoadBalancer.Servers, dynamic.TCPServer{Address: net.JoinHostPort("10.0.0.1", string(rune('0'+i)))})
		}

		for i, status := range statuses {
			service.UpdateServerStatus(net.JoinHostPort("10.0.0.1", string(rune('0'+i))), status)
		}

		return service
	}

	conf := &runtime.Configuration{TCPServices: map[string]*runtime.TCPServiceInfo{
		"up@docker":       loadBalancer(2, runtime.StatusUp, runtime.StatusDown),
		"down@docker":     loadBalancer(2, runtime.StatusDown, runtime.StatusDown),
		"empty@docker":    loadBalancer(0),
		"unknown@docker":  loadBalancer(1),
		"weighted@docker": {TCPService: &dynamic.TCPService{Weighted: &dynamic.TCPWeightedRoundRobin{}}},
	}}

	assert.True(t, tcpServiceUp(conf, "up@docker"))
	assert.False(t, tcpServiceUp(conf, "down@docker"))
	assert.False(t, tcpServiceUp(conf, "empty@docker"))
	assert.True(t, tcpServiceUp(conf, "unknown@docker"))
	assert.True(t, tcpServiceUp(conf, "weighted@docker"))
	assert.False(t, tcpServiceUp(conf, "missing@docker"))
	assert.True(t, tcpServiceUp(nil, "missing@docker"))
}

func TestMesh_LocalTCPRoute(t *testing.T) {
	node := newTestNode(t, "srv1", testToken, nil)

	node.mesh.Transform(t.Context(), dynamic.Configurations{
		"docker": {TCP: &dynamic.TCPConfiguration{Routers: map[string]*dynamic.TCPRouter{
			"pg":      {Rule: "HostSNI(`*`)", EntryPoints: []string{"web"}, Service: "pg"},
			"private": {Rule: "HostSNI(`*`)", EntryPoints: []string{"web"}, Service: "private", Mesh: &dynamic.RouterMesh{Export: new(false)}},
		}}},
		ProviderName: {TCP: &dynamic.TCPConfiguration{Routers: map[string]*dynamic.TCPRouter{
			"mqtt-docker": {Rule: "HostSNI(`*`)", EntryPoints: []string{"web"}, Service: "mqtt-docker"},
		}}},
	})

	assert.True(t, node.mesh.LocalTCPRoute("pg@docker"))
	// A router that is not announced, and one that came from the mesh, are served as they are.
	assert.False(t, node.mesh.LocalTCPRoute("private@docker"))
	assert.False(t, node.mesh.LocalTCPRoute("mqtt-docker@mesh"))

	assert.Contains(t, node.mesh.LocalTCPRouters(), "pg@docker")
	assert.Len(t, node.mesh.LocalTCPRouters(), 1)
}

func TestMesh_localRoutes_tlsOptions(t *testing.T) {
	node := newTestNode(t, "srv1", testToken, nil)

	routes, _ := node.mesh.localRoutes(dynamic.Configurations{
		"file": {
			HTTP: &dynamic.HTTPConfiguration{Routers: map[string]*dynamic.Router{
				"app":   {Rule: "Host(`app.example.com`)", Service: "app", TLS: &dynamic.RouterTLSConfig{Options: "modern"}},
				"plain": {Rule: "Host(`plain.example.com`)", Service: "plain", TLS: &dynamic.RouterTLSConfig{Options: "default"}},
			}},
			TCP: &dynamic.TCPConfiguration{Routers: map[string]*dynamic.TCPRouter{
				"pg":   {Rule: "HostSNI(`pg.example.com`)", Service: "pg", TLS: &dynamic.RouterTCPTLSConfig{Options: "modern"}},
				"mqtt": {Rule: "HostSNI(`mqtt.example.com`)", Service: "mqtt", TLS: &dynamic.RouterTCPTLSConfig{Options: "strict@docker"}},
				"raw":  {Rule: "HostSNI(`raw.example.com`)", Service: "raw", TLS: &dynamic.RouterTCPTLSConfig{Passthrough: true}},
			}},
		},
	})

	options := map[string]string{}
	for _, route := range routes {
		options[routeKey(route.Kind, route.Name)] = route.TLS.Options
	}

	// The other instances look for the options where the router has them: in its provider, not in the one of the mesh.
	assert.Equal(t, map[string]string{
		"http/app@file":   "modern@file",
		"http/plain@file": "default",
		"tcp/pg@file":     "modern@file",
		"tcp/mqtt@file":   "strict@docker",
		"tcp/raw@file":    "",
	}, options)
}

func Test_carry_endsWithTheStream(t *testing.T) {
	// The client keeps its connection open and says nothing: the connection still ends when the stream does.
	near, far := net.Pipe()
	stream := &fakeStream{Conn: far, done: make(chan struct{})}

	var served atomic.Bool

	handler := tcp.HandlerFunc(func(conn tcp.WriteCloser) {
		carry(conn, stream)
		served.Store(true)
	})

	client := connect(t, handler)

	_, err := io.WriteString(near, "over\n")
	require.NoError(t, err)

	line, err := bufio.NewReader(client).ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "over\n", line)

	// The other instance is done with the stream, in both directions.
	require.NoError(t, near.Close())
	close(stream.done)

	_, err = io.ReadAll(client)
	require.NoError(t, err)

	require.Eventually(t, served.Load, 10*time.Second, 10*time.Millisecond)
}

// fakeStream is a stream of a link, made of an in-memory connection.
type fakeStream struct {
	net.Conn

	done chan struct{}
}

func (s *fakeStream) CloseWrite() error     { return nil }
func (s *fakeStream) Done() <-chan struct{} { return s.done }
