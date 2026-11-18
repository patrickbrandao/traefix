package fingerprint

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v3/pkg/config/static"
	"github.com/traefik/traefik/v3/pkg/tcp"
	"github.com/traefik/traefik/v3/pkg/tls/generate"
)

func TestNewEntryPoint_validation(t *testing.T) {
	testCases := []struct {
		desc   string
		change func(*static.Fingerprint)
		err    string
	}{
		{desc: "defaults"},
		{desc: "custom prefix", change: func(c *static.Fingerprint) { c.HeaderPrefix = "X-Edge-" }},
		{desc: "empty prefix", change: func(c *static.Fingerprint) { c.HeaderPrefix = "" }, err: "header prefix"},
		{desc: "prefix with space", change: func(c *static.Fingerprint) { c.HeaderPrefix = "X Fs-" }, err: "header prefix"},
		{desc: "export URL without scheme", change: func(c *static.Fingerprint) { c.Export.URL = "10.0.0.1:9000/ingest" }, err: "export URL"},
		{desc: "export URL with another scheme", change: func(c *static.Fingerprint) { c.Export.URL = "ftp://10.0.0.1/ingest" }, err: "export URL"},
		{desc: "batch size 0", change: func(c *static.Fingerprint) { c.Export.BatchSize = 0 }, err: "batchSize"},
		{desc: "batch size over the limit", change: func(c *static.Fingerprint) { c.Export.BatchSize = 2000 }, err: "batchSize"},
		{desc: "queue size 0", change: func(c *static.Fingerprint) { c.Export.QueueSize = 0 }, err: "queueSize"},
		{desc: "flush interval 0", change: func(c *static.Fingerprint) { c.Export.FlushInterval = 0 }, err: "flushInterval"},
		{desc: "send timeout 0", change: func(c *static.Fingerprint) { c.Export.SendTimeout = ptypes.Duration(0) }, err: "sendTimeout"},
		{desc: "invalid header", change: func(c *static.Fingerprint) { c.Export.Headers = map[string]string{"Bad Name": "x"} }, err: "header"},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			cfg := &static.Fingerprint{}
			cfg.SetDefaults()
			cfg.Export = &static.FingerprintExport{URL: "http://127.0.0.1:9000/ingest"}
			cfg.Export.SetDefaults()
			if test.change != nil {
				test.change(cfg)
			}

			ep, err := NewEntryPoint(t.Context(), "ep", cfg, nil, nil)
			if test.err != "" {
				require.ErrorContains(t, err, test.err)
				return
			}
			require.NoError(t, err)
			ep.Close(context.Background())
		})
	}
}

func TestEntryPoint_removeClientHeaders(t *testing.T) {
	cfg := &static.Fingerprint{HeaderPrefix: "X-Edge-"}
	ep, err := NewEntryPoint(t.Context(), "ep", cfg, nil, nil)
	require.NoError(t, err)

	h := http.Header{}
	for _, name := range []string{"X-Edge-Src-Ip", "x-edge-verdict", "X_EDGE_JA4H", "X-eDgE_cOnn-Id", "X_Edge-Proto"} {
		h[name] = []string{"forged"}
	}
	h.Set("X-Edgex", "kept")
	h.Set("X-Forwarded-For", "kept")
	h.Set("User-Agent", "kept")

	ep.removeClientHeaders(h)

	assert.Equal(t, http.Header{"X-Edgex": {"kept"}, "X-Forwarded-For": {"kept"}, "User-Agent": {"kept"}}, h)
}

// writeCloser adapts a net.Conn to tcp.WriteCloser.
type writeCloser struct {
	net.Conn
}

func (w writeCloser) CloseWrite() error {
	if cw, ok := w.Conn.(interface{ CloseWrite() error }); ok {
		return cw.CloseWrite()
	}
	return nil
}

// oneByteConn returns at most one byte per Read, like a client sending its ClientHello byte by byte.
type oneByteConn struct {
	tcp.WriteCloser
}

func (c oneByteConn) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return c.WriteCloser.Read(p)
}

// tlsEntryPoint serves HTTPS like an entry point: the accepted connection goes through Accept, the TLS termination,
// then WrapConn before net/http.
type tlsEntryPoint struct {
	net.Listener

	ep       *EntryPoint
	cfg      *tls.Config
	oneByte  bool
	accepted chan tcp.WriteCloser
}

func (l *tlsEntryPoint) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	var wc tcp.WriteCloser = writeCloser{Conn: c}
	if l.oneByte {
		wc = oneByteConn{WriteCloser: wc}
	}
	accepted := l.ep.Accept(wc)
	if l.accepted != nil {
		l.accepted <- accepted
	}

	return l.ep.WrapConn(tls.Server(accepted, l.cfg)), nil
}

func startTLSEntryPoint(t *testing.T, ep *EntryPoint, oneByte bool, handler http.Handler) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	serverCfg := &tls.Config{Certificates: []tls.Certificate{mustCert(t)}, NextProtos: []string{"h2", "http/1.1"}}

	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)

	srv := &http.Server{
		Protocols:   &protocols,
		Handler:     ep.Handler(handler),
		ConnContext: ep.ConnContext,
		ConnState: func(c net.Conn, state http.ConnState) {
			if state == http.StateClosed || state == http.StateHijacked {
				ep.ConnClosed(c)
			}
		},
	}
	go func() { _ = srv.Serve(&tlsEntryPoint{Listener: ln, ep: ep, cfg: serverCfg, oneByte: oneByte}) }()
	t.Cleanup(func() { _ = srv.Close() })

	return ln.Addr().String()
}

func mustCert(t *testing.T) tls.Certificate {
	t.Helper()

	certPEM, keyPEM, err := generate.KeyPair("example.com", time.Time{})
	require.NoError(t, err)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)

	return cert
}

func exportedRecords(c *collector) (conns, reqs []map[string]any) {
	for _, b := range c.snapshot() {
		for _, raw := range b.Records {
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			switch m["type"] {
			case recordConnection:
				conns = append(conns, m)
			case recordRequest:
				reqs = append(reqs, m)
			}
		}
	}
	return conns, reqs
}

func echoPrefixedHeaders(prefix string) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = io.Copy(io.Discard, req.Body)
		out := map[string]string{}
		for name := range req.Header {
			if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
				out[name] = req.Header.Get(name)
			}
		}
		_ = json.NewEncoder(rw).Encode(out)
	})
}

func TestEntryPoint_TLS(t *testing.T) {
	for _, proto := range []string{"http/1.1", "h2"} {
		for _, oneByte := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s oneByteReads=%t", proto, oneByte), func(t *testing.T) {
				c := &collector{}
				dest := httptest.NewServer(c)
				t.Cleanup(dest.Close)

				ep := newTestEntryPoint(t, &static.FingerprintExport{
					URL:           dest.URL,
					RawBytes:      true,
					Fields:        []string{"User-Agent", "Sec-Fetch-*", "Cookie"},
					QueueSize:     100,
					QueueMaxBytes: 1 << 20,
					BatchSize:     100,
					FlushInterval: ptypes.Duration(20 * time.Millisecond),
					SendTimeout:   ptypes.Duration(time.Second),
				})
				addr := startTLSEntryPoint(t, ep, oneByte, echoPrefixedHeaders("X-Traefix-"))

				transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, NextProtos: []string{proto}}}
				var protocols http.Protocols
				if proto == "h2" {
					protocols.SetHTTP2(true)
				} else {
					protocols.SetHTTP1(true)
				}
				transport.Protocols = &protocols
				t.Cleanup(transport.CloseIdleConnections)

				req, err := http.NewRequest(http.MethodGet, "https://"+addr+"/path?secret=1", nil)
				require.NoError(t, err)
				req.Header.Set("User-Agent", "test-agent")
				req.Header.Set("Sec-Fetch-Site", "none")
				req.Header.Set("Cookie", "sid=secret")
				req.Header.Set("Authorization", "Bearer visitor")
				// Forged values in every form the removal covers.
				req.Header.Set("X-Traefix-Src-Ip", "6.6.6.6")
				req.Header["x-traefix-ja4h"] = []string{"forged"}
				req.Header["X_Traefix_Conn_Id"] = []string{"forged"}

				resp, err := transport.RoundTrip(req)
				require.NoError(t, err)
				var got map[string]string
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
				_ = resp.Body.Close()

				short := map[string]string{"http/1.1": "h1", "h2": "h2"}[proto]
				assert.Equal(t, "127.0.0.1", got["X-Traefix-Src-Ip"])
				assert.NotEmpty(t, got["X-Traefix-Src-Port"])
				_, port, _ := net.SplitHostPort(addr)
				assert.Equal(t, port, got["X-Traefix-Dst-Port"])
				assert.Len(t, got["X-Traefix-Conn-Id"], 26)
				assert.Equal(t, short, got["X-Traefix-Proto"])
				assert.Regexp(t, `^[0-9a-f]{32}$`, got["X-Traefix-Ja3"])
				assert.Regexp(t, `^t13i`, got["X-Traefix-Ja4"])
				assert.Regexp(t, `^ge(11|20)[cn][rn]\d\d`, got["X-Traefix-Ja4h"])
				assert.NotContains(t, got, "X_traefix_conn_id")
				if short == "h2" {
					assert.NotEmpty(t, got["X-Traefix-H2"])
				} else {
					assert.NotContains(t, got, "X-Traefix-H2")
				}

				transport.CloseIdleConnections()

				var conns, reqs []map[string]any
				require.Eventually(t, func() bool {
					conns, reqs = exportedRecords(c)
					return len(conns) == 1 && len(reqs) == 1
				}, 5*time.Second, 10*time.Millisecond)

				conn, r := conns[0], reqs[0]
				assert.Equal(t, got["X-Traefix-Conn-Id"], conn["connId"])
				assert.Equal(t, conn["connId"], r["connId"])
				assert.Equal(t, short, conn["proto"])
				src := conn["src"].(map[string]any)
				assert.Equal(t, "127.0.0.1", src["ip"])
				assert.Equal(t, got["X-Traefix-Src-Port"], strconv.Itoa(int(src["port"].(float64))))

				tlsRec := conn["tls"].(map[string]any)
				assert.Equal(t, got["X-Traefix-Ja4"], tlsRec["ja4"])
				assert.Equal(t, got["X-Traefix-Ja3"], tlsRec["ja3"])
				assert.Equal(t, proto, tlsRec["alpn"])
				assert.Equal(t, "TLS 1.3", tlsRec["version"])
				assert.NotEmpty(t, tlsRec["cipher"])
				assert.NotEmpty(t, tlsRec["clientHello"])
				if short == "h2" {
					h2 := conn["http2"].(map[string]any)
					assert.Equal(t, got["X-Traefix-H2"], h2["akamai"])
					assert.NotEmpty(t, h2["prefix"])
				}

				assert.Equal(t, got["X-Traefix-Ja4h"], r["ja4h"])
				assert.Equal(t, "/path", r["path"])
				assert.InDelta(t, 200, r["status"], 0)
				assert.Equal(t, []any{"sid"}, r["cookieNames"])
				// The cookie and the authorization values are kept in the process.
				raw, _ := json.Marshal(r)
				assert.NotContains(t, string(raw), "secret")
				assert.NotContains(t, string(raw), "visitor")
				assert.NotContains(t, string(raw), "forged")
				assert.Contains(t, string(raw), `"value":"test-agent"`)
				assert.Contains(t, string(raw), `"value":"none"`)
			})
		}
	}
}

func TestEntryPoint_closedDuringHandshake(t *testing.T) {
	c := &collector{}
	dest := httptest.NewServer(c)
	t.Cleanup(dest.Close)

	ep := newTestEntryPoint(t, &static.FingerprintExport{
		URL: dest.URL, QueueSize: 10, QueueMaxBytes: 1 << 20, BatchSize: 1,
		FlushInterval: ptypes.Duration(time.Hour), SendTimeout: ptypes.Duration(time.Second),
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	done := make(chan struct{})
	go func() {
		defer close(done)
		raw, err := ln.Accept()
		if err != nil {
			return
		}
		accepted := ep.Accept(writeCloser{Conn: raw})
		// The server reads the ClientHello, then the client goes away before the handshake ends.
		_ = tls.Server(accepted, &tls.Config{Certificates: []tls.Certificate{mustCert(t)}}).Handshake()
		_ = accepted.Close()
	}()

	client, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	tc := tls.Client(client, &tls.Config{InsecureSkipVerify: true, ServerName: "example.com"})
	go func() { _ = tc.Handshake() }()
	time.Sleep(100 * time.Millisecond)
	_ = client.Close()
	<-done

	var conns []map[string]any
	require.Eventually(t, func() bool {
		conns, _ = exportedRecords(c)
		return len(conns) == 1
	}, 5*time.Second, 10*time.Millisecond)

	tlsRec := conns[0]["tls"].(map[string]any)
	assert.Equal(t, "example.com", tlsRec["sni"])
	assert.True(t, strings.HasPrefix(tlsRec["ja4"].(string), "t13d"))
	assert.NotContains(t, tlsRec, "version")
}

func TestEntryPoint_clientHelloTooLong(t *testing.T) {
	ep := newTestEntryPoint(t, nil)
	counter := newTestCounter()
	ep.truncatedCounter = counter

	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })

	accepted := ep.Accept(writeCloser{Conn: server}).(*acceptedConn)

	// A handshake record announcing a ClientHello longer than the capture limit.
	go func() {
		hello := []byte{recordTypeHandshake, 3, 1, 0x40, 0, 1, 0, 0x7f, 0xfc}
		_, _ = client.Write(append(hello, make([]byte, maxClientHello+1000)...))
	}()

	buf := make([]byte, 4096)
	for read := 0; read < maxClientHello+1000; {
		n, err := accepted.Read(buf)
		require.NoError(t, err)
		read += n
	}

	accepted.info.mu.Lock()
	assert.True(t, accepted.info.helloTruncated)
	assert.Len(t, accepted.info.hello, maxClientHello)
	assert.Nil(t, accepted.info.parsedHello)
	accepted.info.mu.Unlock()
	assert.InDelta(t, 1, counter.get("entrypoint,test,part,client_hello"), 0)
}

func TestEntryPoint_requestLatencyWithDestinationDown(t *testing.T) {
	// A destination accepting the connections and never answering.
	blackhole, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = blackhole.Close() })
	go func() {
		for {
			c, err := blackhole.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { _ = c.Close() })
		}
	}()

	ep := newTestEntryPoint(t, &static.FingerprintExport{
		URL: "http://" + blackhole.Addr().String(), QueueSize: 10, QueueMaxBytes: 1 << 20, BatchSize: 1,
		FlushInterval: ptypes.Duration(time.Millisecond), SendTimeout: ptypes.Duration(2 * time.Second),
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := &http.Server{Handler: ep.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})), ConnContext: ep.ConnContext}
	go func() { _ = srv.Serve(wrappingListener{Listener: ln, ep: ep}) }()
	t.Cleanup(func() { _ = srv.Close() })

	conn, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	br := bufio.NewReader(conn)

	// More requests than the queue holds, while the destination holds the first batch for the whole send timeout.
	start := time.Now()
	for range 50 {
		_, err := io.WriteString(conn, "GET / HTTP/1.1\r\nHost: h\r\n\r\n")
		require.NoError(t, err)
		resp, err := http.ReadResponse(br, nil)
		require.NoError(t, err)
		_ = resp.Body.Close()
	}
	assert.Less(t, time.Since(start), time.Second)
}

func TestWrapRouterHandler(t *testing.T) {
	c := &collector{}
	dest := httptest.NewServer(c)
	t.Cleanup(dest.Close)

	ep := newTestEntryPoint(t, &static.FingerprintExport{
		URL: dest.URL, QueueSize: 10, QueueMaxBytes: 1 << 20, BatchSize: 1,
		FlushInterval: ptypes.Duration(time.Hour), SendTimeout: ptypes.Duration(time.Second),
	})

	inner, err := WrapRouterHandler("child@file", "svc@file")(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusTeapot)
	}))
	require.NoError(t, err)
	outer, err := WrapRouterHandler("parent@file", "parent-svc@file")(inner)
	require.NoError(t, err)

	ep.Handler(outer).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	var reqs []map[string]any
	require.Eventually(t, func() bool {
		_, reqs = exportedRecords(c)
		return len(reqs) == 1
	}, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, "child@file", reqs[0]["router"])
	assert.Equal(t, "svc@file", reqs[0]["service"])
	assert.InDelta(t, http.StatusTeapot, reqs[0]["status"], 0)
}
