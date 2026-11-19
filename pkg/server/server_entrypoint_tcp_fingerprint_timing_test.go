//go:build !windows

package server

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/static"
	tcprouter "github.com/traefik/traefik/v3/pkg/server/router/tcp"
	traefiktls "github.com/traefik/traefik/v3/pkg/tls"
)

// relayConn delays the client side like a relay between the client and Traefix: before the first write, and
// before each write that answers a read, as the answer crosses the relay once more.
type relayConn struct {
	net.Conn

	firstDelay, answerDelay time.Duration

	mu      sync.Mutex
	written bool
	read    bool
}

func (c *relayConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	c.mu.Lock()
	c.read = true
	c.mu.Unlock()
	return n, err
}

func (c *relayConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	delay := time.Duration(0)
	switch {
	case !c.written:
		delay = c.firstDelay
	case c.read:
		delay = c.answerDelay
	}
	c.written = true
	c.read = false
	c.mu.Unlock()

	time.Sleep(delay)
	return c.Conn.Write(p)
}

func parseKV(t *testing.T, s string) map[string]int64 {
	t.Helper()

	out := map[string]int64{}
	for pair := range strings.SplitSeq(s, ",") {
		k, v, ok := strings.Cut(pair, "=")
		require.True(t, ok, pair)
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			out[k] = n
		}
	}
	return out
}

// TestFingerprintEntryPoint_timing checks the timing header of a TLS connection through a simulated relay:
// the wait between the connection and the ClientHello, and the TLS round trip, grow with the relay delays,
// while the TCP round trip, measured by the kernel with the peer of the TCP connection, stays the one of loopback.
func TestFingerprintEntryPoint_timing(t *testing.T) {
	certContent, err := localhostCert.Read()
	require.NoError(t, err)
	keyContent, err := localhostKey.Read()
	require.NoError(t, err)
	tlsCert, err := tls.X509KeyPair(certContent, keyContent)
	require.NoError(t, err)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	epConfig := &static.EntryPointsTransport{}
	epConfig.SetDefaults()
	entryPoint, err := NewTCPEntryPoint(t.Context(), "websecure", &static.EntryPoint{
		Address:          addr,
		Transport:        epConfig,
		ForwardedHeaders: &static.ForwardedHeaders{},
		HTTP2:            &static.HTTP2Config{},
		Fingerprint:      &static.Fingerprint{HeaderPrefix: "X-Edge-", TCP: true},
	}, nil, nil)
	require.NoError(t, err)

	router, err := tcprouter.NewRouter(nil)
	require.NoError(t, err)
	router.AddHTTPTLSConfig("example.com", &tls.Config{Certificates: []tls.Certificate{tlsCert}, NextProtos: []string{"http/1.1"}}, traefiktls.DefaultTLSConfigName)
	router.SetHTTPSHandler(http.HandlerFunc(echoFingerprintHeaders), nil)

	go entryPoint.Start(t.Context())
	entryPoint.SwitchRouter(router)
	t.Cleanup(func() { entryPoint.Shutdown(t.Context()) })

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(certContent)

	get := func(t *testing.T, firstDelay, answerDelay time.Duration) map[string]int64 {
		t.Helper()

		raw, err := net.Dial("tcp", addr)
		require.NoError(t, err)
		relay := &relayConn{Conn: raw, firstDelay: firstDelay, answerDelay: answerDelay}
		conn := tls.Client(relay, &tls.Config{RootCAs: certPool, ServerName: "example.com", NextProtos: []string{"http/1.1"}})
		t.Cleanup(func() { _ = conn.Close() })

		req, err := http.NewRequest(http.MethodGet, "https://example.com/", nil)
		require.NoError(t, err)
		require.NoError(t, req.Write(conn))
		resp, err := http.ReadResponse(bufio.NewReader(conn), req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		var got map[string]string
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		t.Log(got["X-Edge-Timing"])
		return parseKV(t, got["X-Edge-Timing"])
	}

	direct := get(t, 0, 0)
	relayed := get(t, 60*time.Millisecond, 40*time.Millisecond)

	// The relay delays are 60 and 40 ms: the differences with the direct connection hold whatever the load of the
	// test host, which only adds to both.
	assert.GreaterOrEqual(t, relayed["est_to_first_us"], int64(55_000))
	assert.GreaterOrEqual(t, relayed["est_to_first_us"]-direct["est_to_first_us"], int64(40_000))

	assert.GreaterOrEqual(t, relayed["tls_rtt_us"], int64(35_000))
	assert.GreaterOrEqual(t, relayed["tls_rtt_us"]-direct["tls_rtt_us"], int64(25_000))
	assert.GreaterOrEqual(t, relayed["tls_us"], relayed["tls_rtt_us"])
	assert.Contains(t, relayed, "ch_us")
	assert.Contains(t, relayed, "srv_us")
	assert.Contains(t, relayed, "req_us")

	if runtime.GOOS == "linux" {
		// The kernel times the arrivals, and the TCP round trip is the one of loopback.
		assert.Less(t, relayed["tcp_rtt_us"], int64(5_000))
		assert.GreaterOrEqual(t, relayed["rtt_gap_us"], int64(30_000))
		assert.GreaterOrEqual(t, relayed["rtt_gap_us"]-direct["rtt_gap_us"], int64(25_000))
	}
}
