package fingerprint

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/static"
	"github.com/traefik/traefik/v3/pkg/tls/generate"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

type wrappingListener struct {
	net.Listener

	ep *EntryPoint
}

func (l wrappingListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return l.ep.WrapConn(c), nil
}

func newTestEntryPoint(t *testing.T, export *static.FingerprintExport) *EntryPoint {
	t.Helper()

	cfg := &static.Fingerprint{}
	cfg.SetDefaults()
	cfg.Export = export

	ep, err := NewEntryPoint(t.Context(), "test", cfg, nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() { ep.Close(context.Background()) })

	return ep
}

// startServer serves the given listener with net/http as the entry point does: HTTP/1, HTTP/2 and unencrypted HTTP/2.
// Each response body tells the protocol, whether the TLS state was set, and the JA4H of the captured request.
func startServer(t *testing.T, ln net.Listener) {
	t.Helper()

	ep := newTestEntryPoint(t, nil)

	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	protocols.SetUnencryptedHTTP2(true)

	srv := &http.Server{
		Protocols: &protocols,
		Handler: ep.Handler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = io.Copy(io.Discard, req.Body)

			ja4h := "none"
			if r, ok := FromRequest(req); ok {
				ja4h = r.JA4H()
			}
			_, _ = fmt.Fprintf(rw, "%d %t %s", req.ProtoMajor, req.TLS != nil, ja4h)
		})),
		ConnContext: ep.ConnContext,
	}

	go func() { _ = srv.Serve(wrappingListener{Listener: ln, ep: ep}) }()
	t.Cleanup(func() { _ = srv.Close() })
}

func tlsListener(t *testing.T) (net.Listener, *tls.Config) {
	t.Helper()

	certPEM, keyPEM, err := generate.KeyPair("example.com", time.Time{})
	require.NoError(t, err)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	return tls.NewListener(ln, &tls.Config{Certificates: []tls.Certificate{cert}, NextProtos: []string{"h2", "http/1.1"}}),
		&tls.Config{InsecureSkipVerify: true}
}

func TestServer_h1(t *testing.T) {
	testCases := []struct {
		desc   string
		useTLS bool
	}{
		{desc: "TLS", useTLS: true},
		{desc: "plain"},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			var conn net.Conn
			if test.useTLS {
				ln, clientCfg := tlsListener(t)
				startServer(t, ln)
				clientCfg.NextProtos = []string{"http/1.1"}
				c, err := tls.Dial("tcp", ln.Addr().String(), clientCfg)
				require.NoError(t, err)
				conn = c
			} else {
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				startServer(t, ln)
				conn, err = net.Dial("tcp", ln.Addr().String())
				require.NoError(t, err)
			}
			t.Cleanup(func() { _ = conn.Close() })

			br := bufio.NewReader(conn)
			// All the requests go over a single keep-alive connection, written one byte at a time for the first one.
			for i, v := range vectorsFor("h1") {
				if v.version == "HTTP/1.0" {
					continue
				}
				wire := v.h1Bytes()
				if i == 0 {
					for _, b := range wire {
						_, err := conn.Write([]byte{b})
						require.NoError(t, err)
					}
				} else {
					_, err := conn.Write(wire)
					require.NoError(t, err)
				}

				resp, err := http.ReadResponse(br, &http.Request{Method: v.method})
				require.NoError(t, err)
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				_ = resp.Body.Close()

				assert.Equal(t, fmt.Sprintf("1 %t %s", test.useTLS, v.expected), string(body), v.name)
			}
		})
	}
}

func TestServer_h2(t *testing.T) {
	testCases := []struct {
		desc   string
		useTLS bool
	}{
		{desc: "TLS", useTLS: true},
		{desc: "h2c"},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			var conn net.Conn
			if test.useTLS {
				ln, clientCfg := tlsListener(t)
				startServer(t, ln)
				clientCfg.NextProtos = []string{"h2"}
				c, err := tls.Dial("tcp", ln.Addr().String(), clientCfg)
				require.NoError(t, err)
				require.Equal(t, "h2", c.ConnectionState().NegotiatedProtocol)
				conn = c
			} else {
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				startServer(t, ln)
				conn, err = net.Dial("tcp", ln.Addr().String())
				require.NoError(t, err)
			}
			t.Cleanup(func() { _ = conn.Close() })

			vectors := vectorsFor("h2")
			shapes := []h2Shape{{}, {continuation: true}, {padded: true, priority: true}}

			w := newH2Writer()
			w.buf.Write(h2Preface)
			require.NoError(t, w.fr.WriteSettings())

			expected := map[uint32]string{}
			streamID := uint32(1)
			for range 2 {
				for i, v := range vectors {
					w.request(t, streamID, v, shapes[i%len(shapes)])
					expected[streamID] = fmt.Sprintf("2 %t %s", test.useTLS, v.expected)
					streamID += 2
				}
			}

			// The whole client side goes in one write, so that the server reads several streams at once.
			_, err := conn.Write(w.buf.Bytes())
			require.NoError(t, err)

			got := readH2Bodies(t, conn, len(expected))
			assert.Equal(t, expected, got)
		})
	}
}

// readH2Bodies reads the server frames until n streams have ended, and returns their bodies.
func readH2Bodies(t *testing.T, conn net.Conn, n int) map[uint32]string {
	t.Helper()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(10*time.Second)))

	fr := http2.NewFramer(conn, conn)
	fr.ReadMetaHeaders = hpack.NewDecoder(4096, nil)

	bodies := map[uint32]*bytes.Buffer{}
	ended := map[uint32]string{}
	for len(ended) < n {
		f, err := fr.ReadFrame()
		require.NoError(t, err)

		switch f := f.(type) {
		case *http2.SettingsFrame:
			if !f.IsAck() {
				require.NoError(t, fr.WriteSettingsAck())
			}
		case *http2.MetaHeadersFrame:
			require.Equal(t, "200", f.PseudoValue("status"))
			bodies[f.StreamID] = &bytes.Buffer{}
		case *http2.DataFrame:
			bodies[f.StreamID].Write(f.Data())
			if f.StreamEnded() {
				ended[f.StreamID] = bodies[f.StreamID].String()
			}
		case *http2.GoAwayFrame:
			t.Fatalf("GOAWAY: %v %s", f.ErrCode, f.DebugData())
		}
	}

	return ended
}

func TestServer_noCaptureWithoutWrap(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ep := newTestEntryPoint(t, nil)
	srv := &http.Server{Handler: ep.Handler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, ok := FromRequest(req)
		_, _ = fmt.Fprint(rw, ok)
	}))}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	resp, err := http.Get("http://" + ln.Addr().String())
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	_ = resp.Body.Close()

	assert.Equal(t, "false", strings.TrimSpace(string(body)))
}

func TestUnwrapConn(t *testing.T) {
	c1, c2 := net.Pipe()
	t.Cleanup(func() { _ = c1.Close(); _ = c2.Close() })

	ep := newTestEntryPoint(t, nil)

	tc := tls.Server(c1, &tls.Config{})
	wrapped := ep.WrapConn(tc)

	_, isStater := wrapped.(interface{ ConnectionState() tls.ConnectionState })
	assert.True(t, isStater)
	assert.Same(t, tc, UnwrapConn(wrapped))

	plain := ep.WrapConn(c1)
	_, isStater = plain.(interface{ ConnectionState() tls.ConnectionState })
	assert.False(t, isStater)
	assert.Equal(t, c1, UnwrapConn(plain))

	assert.NotNil(t, connInfoFromContext(ep.ConnContext(context.Background(), wrapped)))
	assert.Nil(t, connInfoFromContext(ep.ConnContext(context.Background(), c1)))
}
