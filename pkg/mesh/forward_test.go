package mesh

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/fingerprint"
)

// seen is what the handler of a router saw of a request it served.
type seen struct {
	remote      string
	host        string
	tls         *tls.ConnectionState
	header      http.Header
	fingerprint fingerprint.Fingerprints
	measured    bool
}

// owns makes the instance announce a router named app@docker, and serve it with the handler.
func (n *testNode) owns(handler http.Handler, priority int) {
	n.t.Helper()

	n.mesh.SwitchHandlers(Handlers{HTTP: map[string]http.Handler{"app@docker": handler}}, nil)
	n.mesh.Transform(n.t.Context(), dynamic.Configurations{
		"docker": {HTTP: &dynamic.HTTPConfiguration{Routers: map[string]*dynamic.Router{
			"app": {Rule: "Host(`app.example.com`)", EntryPoints: []string{"web"}, Service: "app", Mesh: &dynamic.RouterMesh{Priority: priority}},
		}}},
	})
}

// entry returns the handler of the service that the instance made for the router of the other instances.
func (n *testNode) entry() http.Handler {
	n.t.Helper()

	n.waitRouters("app-docker")

	handler, err := n.mesh.ServiceBuilder().BuildHTTP(n.t.Context(), "app-docker@mesh")
	require.NoError(n.t, err)
	require.NotNil(n.t, handler)

	return handler
}

// candidates returns how many instances the instance knows to own the router of the tests.
func (n *testNode) candidates() int {
	tables := n.mesh.tables.Load()

	if target := tables.targets["app-docker@mesh"]; target != nil {
		return len(target.candidates)
	}

	if target := tables.shared["app@docker"]; target != nil {
		return len(target.candidates)
	}

	return 0
}

func withTransport(transport string) func(conf *cluster.Configuration) {
	return func(conf *cluster.Configuration) { conf.Mesh.Transport = transport }
}

func TestMesh_forward(t *testing.T) {
	for _, transport := range transports() {
		t.Run(transport, func(t *testing.T) {
			var mu sync.Mutex
			var got seen

			owner := newTestNode(t, "srv1", testToken, withTransport(transport))
			owner.owns(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				body, _ := io.ReadAll(req.Body)

				mu.Lock()
				got = seen{remote: req.RemoteAddr, host: req.Host, tls: req.TLS, header: req.Header.Clone()}
				got.fingerprint, got.measured = fingerprint.FingerprintsFromRequest(req)
				mu.Unlock()

				rw.Header().Set("X-App", "yes")
				rw.WriteHeader(http.StatusCreated)
				_, _ = rw.Write(append([]byte("echo: "), body...))
			}), 0)

			ingress := newTestNode(t, "srv2", testToken, withTransport(transport), owner.address)
			handler := ingress.entry()

			req := httptest.NewRequest(http.MethodPost, "https://app.example.com/path?x=1", strings.NewReader("hello"))
			req.RemoteAddr = "198.51.100.7:51234"
			req.TLS = &tls.ConnectionState{Version: tls.VersionTLS13, CipherSuite: tls.TLS_AES_128_GCM_SHA256, ServerName: "app.example.com", NegotiatedProtocol: "h2"}
			req.Header.Set("X-Forwarded-For", "203.0.113.9")
			req.Header.Set("X-Forwarded-Proto", "https")
			req.Header.Set("X-Traefix-Ja4", "t13d1516h2_8daaf6152771_e5627efa2ab1")
			req.Header.Set(headerForward, "forged")
			req = req.WithContext(fingerprint.ContextWithForwarded(req.Context(), fingerprint.Forwarded{
				ConnID:       "01JCONN",
				Fingerprints: fingerprint.Fingerprints{JA3: "ja3", JA4: "ja4", JA4H: "ja4h", HTTP2: "akamai", JA4T: "ja4t"},
			}))

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())
			assert.Equal(t, "echo: hello", recorder.Body.String())
			assert.Equal(t, "yes", recorder.Header().Get("X-App"))

			mu.Lock()
			defer mu.Unlock()

			// The handler of the router sees the client, not the instance that forwarded the request.
			assert.Equal(t, "198.51.100.7:51234", got.remote)
			assert.Equal(t, "app.example.com", got.host)

			require.NotNil(t, got.tls)
			assert.Equal(t, uint16(tls.VersionTLS13), got.tls.Version)
			assert.Equal(t, "app.example.com", got.tls.ServerName)
			assert.Equal(t, "h2", got.tls.NegotiatedProtocol)

			// The headers the entry point settled reach the router as they are, and the one of the mesh stays in the mesh.
			assert.Equal(t, []string{"203.0.113.9"}, got.header["X-Forwarded-For"])
			assert.Equal(t, "https", got.header.Get("X-Forwarded-Proto"))
			assert.Equal(t, "t13d1516h2_8daaf6152771_e5627efa2ab1", got.header.Get("X-Traefix-Ja4"))
			assert.Empty(t, got.header.Get(headerForward))

			assert.True(t, got.measured)
			assert.Equal(t, fingerprint.Fingerprints{JA3: "ja3", JA4: "ja4", JA4H: "ja4h", HTTP2: "akamai", JA4T: "ja4t"}, got.fingerprint)
		})
	}
}

func TestMesh_forward_plainHTTP(t *testing.T) {
	var mu sync.Mutex
	var got seen

	owner := newTestNode(t, "srv1", testToken, nil)
	owner.owns(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		got = seen{tls: req.TLS}
		_, got.measured = fingerprint.FingerprintsFromRequest(req)
	}), 0)

	ingress := newTestNode(t, "srv2", testToken, nil, owner.address)

	recorder := httptest.NewRecorder()
	ingress.entry().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://app.example.com/", nil))

	require.Equal(t, http.StatusOK, recorder.Code)

	mu.Lock()
	defer mu.Unlock()

	// The link between the instances is encrypted, the connection of the client was not: the router sees the latter.
	assert.Nil(t, got.tls)
	assert.False(t, got.measured)
}

func TestMesh_forward_streams(t *testing.T) {
	for _, transport := range transports() {
		t.Run(transport, func(t *testing.T) {
			owner := newTestNode(t, "srv1", testToken, withTransport(transport))
			owner.owns(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				controller := http.NewResponseController(rw)

				rw.Header().Set("Trailer", "X-Count")
				rw.WriteHeader(http.StatusOK)
				_ = controller.Flush()

				count := 0
				reader := bufio.NewReader(req.Body)

				for {
					line, err := reader.ReadString('\n')
					if line != "" {
						count++
						_, _ = rw.Write([]byte("got " + line))
						_ = controller.Flush()
					}

					if err != nil {
						break
					}
				}

				rw.Header().Set("X-Count", strconv.Itoa(count))
			}), 0)

			ingress := newTestNode(t, "srv2", testToken, withTransport(transport), owner.address)

			// The client talks HTTP/2 to the instance it reached, as a gRPC client does.
			front := httptest.NewUnstartedServer(ingress.entry())
			front.EnableHTTP2 = true
			front.StartTLS()
			t.Cleanup(front.Close)

			reader, writer := io.Pipe()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, front.URL, reader)
			require.NoError(t, err)

			resp, err := front.Client().Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			require.Equal(t, http.StatusOK, resp.StatusCode)

			answers := bufio.NewReader(resp.Body)

			// Each message is answered before the next one is sent: the request and the response flow at the same time.
			for _, message := range []string{"one\n", "two\n", "three\n"} {
				_, err = writer.Write([]byte(message))
				require.NoError(t, err)

				answer, err := answers.ReadString('\n')
				require.NoError(t, err)
				assert.Equal(t, "got "+message, answer)
			}

			require.NoError(t, writer.Close())

			_, err = io.ReadAll(answers)
			require.NoError(t, err)

			assert.Equal(t, "3", resp.Trailer.Get("X-Count"))
		})
	}
}

func TestMesh_forward_upgrade(t *testing.T) {
	for _, transport := range transports() {
		t.Run(transport, func(t *testing.T) {
			var mu sync.Mutex
			var remote string

			owner := newTestNode(t, "srv1", testToken, withTransport(transport))
			owner.owns(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if req.URL.Path != "/ws" {
					http.Error(rw, "no upgrade here", http.StatusBadRequest)
					return
				}

				mu.Lock()
				remote = req.RemoteAddr
				mu.Unlock()

				// What a WebSocket server does: it takes the connection over and speaks its own protocol on it.
				conn, buffered, err := http.NewResponseController(rw).Hijack()
				if err != nil {
					http.Error(rw, err.Error(), http.StatusInternalServerError)
					return
				}
				defer func() { _ = conn.Close() }()

				_, _ = buffered.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: echo\r\n\r\n")
				_ = buffered.Flush()

				for {
					line, err := buffered.ReadString('\n')
					if err != nil {
						return
					}

					_, _ = buffered.WriteString("echo " + line)
					_ = buffered.Flush()
				}
			}), 0)

			ingress := newTestNode(t, "srv2", testToken, withTransport(transport), owner.address)

			front := httptest.NewServer(ingress.entry())
			t.Cleanup(front.Close)

			dial := func(path string) (net.Conn, *bufio.Reader, *http.Response) {
				conn, err := net.Dial("tcp", front.Listener.Addr().String())
				require.NoError(t, err)
				t.Cleanup(func() { _ = conn.Close() })

				_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

				_, err = fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: app.example.com\r\nConnection: Upgrade\r\nUpgrade: echo\r\n\r\n", path)
				require.NoError(t, err)

				reader := bufio.NewReader(conn)

				resp, err := http.ReadResponse(reader, nil)
				require.NoError(t, err)

				return conn, reader, resp
			}

			conn, reader, resp := dial("/ws")

			require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
			assert.Equal(t, "echo", resp.Header.Get("Upgrade"))

			for _, message := range []string{"one\n", "two\n"} {
				_, err := conn.Write([]byte(message))
				require.NoError(t, err)

				answer, err := reader.ReadString('\n')
				require.NoError(t, err)
				assert.Equal(t, "echo "+message, answer)
			}

			mu.Lock()
			assert.Equal(t, conn.LocalAddr().String(), remote)
			mu.Unlock()

			// A server that does not switch answers as usual.
			_, _, resp = dial("/other")

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Contains(t, string(body), "no upgrade here")
		})
	}
}

func TestMesh_forward_unknownRouter(t *testing.T) {
	owner := newTestNode(t, "srv1", testToken, nil)
	owner.owns(http.NotFoundHandler(), 0)

	ingress := newTestNode(t, "srv2", testToken, nil, owner.address)
	handler := ingress.entry()

	// The router is announced, and its handler is gone: the container stopped, and the news has not arrived yet.
	owner.mesh.routing.Store(&routing{})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://app.example.com/", nil))

	// The answer says that it comes from the mesh, unlike a 404 of the service.
	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Proxy-Status"), "traefix-mesh")
}

func TestMesh_forward_noInstance(t *testing.T) {
	owner := newTestNode(t, "srv1", testToken, nil)
	owner.owns(http.NotFoundHandler(), 0)

	ingress := newTestNode(t, "srv2", testToken, nil, owner.address)
	handler := ingress.entry()

	owner.stop()
	ingress.waitRouters()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://app.example.com/", nil))

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Proxy-Status"), "traefix-mesh")
}

// served returns a handler that answers with the name of its instance.
func named(name string) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) { _, _ = rw.Write([]byte(name)) })
}

func ask(t *testing.T, handler http.Handler, remote string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/", nil)
	req.RemoteAddr = remote

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	return recorder.Body.String()
}

func TestMesh_forward_priority(t *testing.T) {
	primary := newTestNode(t, "srv1", testToken, nil)
	primary.owns(named("srv1"), 10)

	backup := newTestNode(t, "srv2", testToken, nil, primary.address)
	backup.owns(named("srv2"), 0)

	ingress := newTestNode(t, "srv3", testToken, nil, primary.address)
	ingress.waitLinked("srv1", "srv2")

	handler := ingress.entry()

	require.Eventually(t, func() bool { return ingress.candidates() == 2 }, 10*time.Second, 10*time.Millisecond)

	// Every request goes to the instance with the highest priority.
	for i := range 20 {
		assert.Equal(t, "srv1", ask(t, handler, fmt.Sprintf("198.51.100.%d:1000", i)))
	}

	// It goes away: the other one takes over.
	primary.stop()

	require.Eventually(t, func() bool {
		link, _ := ingress.linkTo("srv1")
		return !link.Up
	}, 10*time.Second, 10*time.Millisecond)

	assert.Equal(t, "srv2", ask(t, handler, "198.51.100.1:1000"))
}

func TestMesh_forward_balance(t *testing.T) {
	testCases := []struct {
		desc    string
		balance string
	}{
		{desc: "client IP", balance: cluster.MeshBalanceClientIP},
		{desc: "round robin", balance: cluster.MeshBalanceRoundRobin},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			withBalance := func(conf *cluster.Configuration) { conf.Mesh.Balance = test.balance }

			first := newTestNode(t, "srv1", testToken, withBalance)
			first.owns(named("srv1"), 5)

			second := newTestNode(t, "srv2", testToken, withBalance, first.address)
			second.owns(named("srv2"), 5)

			ingress := newTestNode(t, "srv3", testToken, withBalance, first.address)
			other := newTestNode(t, "srv4", testToken, withBalance, first.address)

			ingress.waitLinked("srv1", "srv2", "srv4")
			other.waitLinked("srv1", "srv2", "srv3")

			handler, otherHandler := ingress.entry(), other.entry()

			for _, node := range []*testNode{ingress, other} {
				require.Eventually(t, func() bool { return node.candidates() == 2 }, 10*time.Second, 10*time.Millisecond)
			}

			counts := map[string]int{}

			for i := range 200 {
				remote := fmt.Sprintf("198.51.100.%d:%d", i, 1000+i)

				answer := ask(t, handler, remote)
				counts[answer]++

				if test.balance == cluster.MeshBalanceClientIP {
					// The same client reaches the same instance again, from another port and through another instance.
					assert.Equal(t, answer, ask(t, handler, fmt.Sprintf("198.51.100.%d:9", i)))
					assert.Equal(t, answer, ask(t, otherHandler, remote))
				}
			}

			if test.balance == cluster.MeshBalanceRoundRobin {
				assert.Equal(t, 100, counts["srv1"])
				assert.Equal(t, 100, counts["srv2"])

				return
			}

			// The clients spread over the instances, without being split exactly.
			assert.Greater(t, counts["srv1"], 60)
			assert.Greater(t, counts["srv2"], 60)
		})
	}
}

func TestForwardInfo(t *testing.T) {
	info := forwardInfo{
		Version: 1, Kind: forwardHTTP, Router: "app@docker", Remote: "198.51.100.7:51234",
		TLS:         &forwardTLS{Version: tls.VersionTLS13, ServerName: "app.example.com"},
		Fingerprint: &forwardFingerprint{JA4: "ja4"},
	}

	decoded, err := decodeForwardInfo(info.encode())
	require.NoError(t, err)
	assert.Equal(t, info, decoded)

	for _, value := range []string{"", "forged", "e30"} {
		_, err := decodeForwardInfo(value)
		require.Error(t, err, value)
	}
}
