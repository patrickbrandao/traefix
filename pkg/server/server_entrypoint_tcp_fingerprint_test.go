//go:build !windows

package server

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/quic-go/quic-go/http3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/static"
	"github.com/traefik/traefik/v3/pkg/fingerprint"
	tcprouter "github.com/traefik/traefik/v3/pkg/server/router/tcp"
	traefiktls "github.com/traefik/traefik/v3/pkg/tls"
	"github.com/traefik/traefik/v3/pkg/tls/generate"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

// The expected fingerprints are the ones of the FoxIO Python reference (python/ja4h.py) for the same fields.
const (
	fingerprintH1Request = "GET /a?b=c HTTP/1.1\r\nHost: example.com\r\nConnection: keep-alive\r\nsec-ch-ua: \"Chromium\";v=\"131\"\r\nUser-Agent: Mozilla/5.0\r\nAccept: text/html\r\nReferer: https://example.com/\r\nAccept-Encoding: gzip, deflate, br\r\nAccept-Language: en-US,en;q=0.9\r\nCookie: sid=abc; _ga=GA1.2; theme=dark\r\n\r\n"
	fingerprintH1JA4H    = "ge11cr07enus_3c666a2bb02e_86aa86fc6cd2_760a8ef12d1d"
	fingerprintH2JA4H    = "ge20cr08enus_308e4456ec31_b50932941748_1a7d4e15bf8f"
)

var fingerprintH2Fields = []hpack.HeaderField{
	{Name: ":method", Value: "GET"},
	{Name: ":authority", Value: "example.com"},
	{Name: ":scheme", Value: "https"},
	{Name: ":path", Value: "/"},
	{Name: "sec-ch-ua", Value: `"Chromium";v="131"`},
	{Name: "sec-ch-ua-mobile", Value: "?0"},
	{Name: "user-agent", Value: "Mozilla/5.0"},
	{Name: "accept", Value: "text/html"},
	{Name: "sec-fetch-site", Value: "none"},
	{Name: "referer", Value: "https://example.com/"},
	{Name: "accept-encoding", Value: "gzip, deflate, br, zstd"},
	{Name: "accept-language", Value: "en-US,en;q=0.9"},
	{Name: "cookie", Value: "_ga=GA1.2"},
	{Name: "cookie", Value: "sid=xyz"},
	{Name: "priority", Value: "u=0, i"},
}

func fingerprintHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ja4h := "none"
		if r, ok := fingerprint.FromRequest(req); ok {
			ja4h = r.JA4H()
		}
		_, _ = fmt.Fprintf(rw, "%d %t %s", req.ProtoMajor, req.TLS != nil, ja4h)
	})
}

func fingerprintTLSConfig(tb testing.TB) *tls.Config {
	tb.Helper()

	certPEM, keyPEM, err := generate.KeyPair("example.com", time.Time{})
	require.NoError(tb, err)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(tb, err)

	return &tls.Config{Certificates: []tls.Certificate{cert}, NextProtos: []string{"h2", "http/1.1"}}
}

func startFingerprintEntryPoint(tb testing.TB, enabled bool) string {
	tb.Helper()

	var fp *static.Fingerprint
	if enabled {
		fp = &static.Fingerprint{}
		fp.SetDefaults()
		if url := os.Getenv("FINGERPRINT_EXPORT_URL"); url != "" {
			fp.Export = &static.FingerprintExport{URL: url, RawBytes: true}
			fp.Export.SetDefaults()
		}
	}

	return startEntryPointWithFingerprint(tb, fp)
}

func startEntryPointWithFingerprint(tb testing.TB, fp *static.Fingerprint) string {
	tb.Helper()

	epConfig := &static.EntryPointsTransport{}
	epConfig.SetDefaults()

	config := &static.EntryPoint{
		Address:          "127.0.0.1:0",
		Transport:        epConfig,
		ForwardedHeaders: &static.ForwardedHeaders{},
		HTTP2:            &static.HTTP2Config{},
	}
	config.Fingerprint = fp

	entryPoint, err := NewTCPEntryPoint(tb.Context(), "", config, nil, nil)
	require.NoError(tb, err)

	router, err := tcprouter.NewRouter(nil)
	require.NoError(tb, err)
	router.SetHTTPHandler(fingerprintHandler())
	router.SetHTTPSHandler(fingerprintHandler(), fingerprintTLSConfig(tb))

	go entryPoint.Start(tb.Context())
	entryPoint.SwitchRouter(router)
	tb.Cleanup(func() { entryPoint.Shutdown(tb.Context()) })

	return entryPoint.listener.Addr().String()
}

func TestFingerprintEntryPoint_headerOrder(t *testing.T) {
	addr := startFingerprintEntryPoint(t, true)

	testCases := []struct {
		desc     string
		useTLS   bool
		h2       bool
		expected string
	}{
		{desc: "h1 over TLS", useTLS: true, expected: "1 true " + fingerprintH1JA4H},
		{desc: "h1", expected: "1 false " + fingerprintH1JA4H},
		{desc: "h2 over TLS", useTLS: true, h2: true, expected: "2 true " + fingerprintH2JA4H},
		{desc: "h2c", h2: true, expected: "2 false " + fingerprintH2JA4H},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			var conn net.Conn
			var err error
			if test.useTLS {
				proto := "http/1.1"
				if test.h2 {
					proto = "h2"
				}
				var tc *tls.Conn
				tc, err = tls.Dial("tcp", addr, &tls.Config{InsecureSkipVerify: true, ServerName: "example.com", NextProtos: []string{proto}})
				require.NoError(t, err)
				require.Equal(t, proto, tc.ConnectionState().NegotiatedProtocol)
				conn = tc
			} else {
				conn, err = net.Dial("tcp", addr)
				require.NoError(t, err)
			}
			t.Cleanup(func() { _ = conn.Close() })
			require.NoError(t, conn.SetDeadline(time.Now().Add(10*time.Second)))

			// Two requests on the same connection, to cover the parsing past the first one.
			for range 2 {
				var body string
				if test.h2 {
					body = fingerprintH2Roundtrip(t, conn)
				} else {
					body = fingerprintH1Roundtrip(t, conn)
				}
				assert.Equal(t, test.expected, body)
			}
		})
	}
}

func TestFingerprintEntryPoint_disabled(t *testing.T) {
	addr := startFingerprintEntryPoint(t, false)

	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	assert.Equal(t, "1 false none", fingerprintH1Roundtrip(t, conn))
}

func fingerprintH1Roundtrip(t *testing.T, conn net.Conn) string {
	t.Helper()

	_, err := io.WriteString(conn, fingerprintH1Request)
	require.NoError(t, err)

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return string(body)
}

type h2TestConn struct {
	fr   *http2.Framer
	enc  *hpack.Encoder
	hb   bytes.Buffer
	next uint32
}

var h2TestConns = map[net.Conn]*h2TestConn{}

func fingerprintH2Roundtrip(t *testing.T, conn net.Conn) string {
	t.Helper()

	c, ok := h2TestConns[conn]
	if !ok {
		c = &h2TestConn{fr: http2.NewFramer(conn, conn), next: 1}
		c.enc = hpack.NewEncoder(&c.hb)
		c.fr.ReadMetaHeaders = hpack.NewDecoder(4096, nil)
		h2TestConns[conn] = c
		t.Cleanup(func() { delete(h2TestConns, conn) })

		_, err := io.WriteString(conn, http2.ClientPreface)
		require.NoError(t, err)
		require.NoError(t, c.fr.WriteSettings())
	}

	c.hb.Reset()
	for _, f := range fingerprintH2Fields {
		require.NoError(t, c.enc.WriteField(f))
	}
	streamID := c.next
	c.next += 2
	require.NoError(t, c.fr.WriteHeaders(http2.HeadersFrameParam{StreamID: streamID, BlockFragment: c.hb.Bytes(), EndStream: true, EndHeaders: true}))

	var body bytes.Buffer
	for {
		f, err := c.fr.ReadFrame()
		require.NoError(t, err)

		switch f := f.(type) {
		case *http2.SettingsFrame:
			if !f.IsAck() {
				require.NoError(t, c.fr.WriteSettingsAck())
			}
		case *http2.DataFrame:
			if f.StreamID != streamID {
				continue
			}
			body.Write(f.Data())
			if f.StreamEnded() {
				return body.String()
			}
		case *http2.GoAwayFrame:
			t.Fatalf("GOAWAY: %v", f.ErrCode)
		}
	}
}

// BenchmarkFingerprintEntryPoint measures an HTTPS entry point with the capture off and on.
// The entry point runs in a child process, so that server-cpu-ns/op is the CPU time of the server alone
// (user and system, from the child rusage, also reported apart), apart from the client load generated here.
// With the capture on, the entry point also adds the headers and exports the records to a destination run here.
func BenchmarkFingerprintEntryPoint(b *testing.B) {
	for _, proto := range []string{"h2", "h1"} {
		for _, enabled := range []bool{false, true} {
			name := fmt.Sprintf("%s/capture=%t", proto, enabled)
			b.Run(name, func(b *testing.B) {
				addr, server := startFingerprintServerProcess(b, enabled)

				transport := &http.Transport{
					TLSClientConfig:     &tls.Config{InsecureSkipVerify: true, ServerName: "example.com"},
					MaxIdleConnsPerHost: 100,
				}
				var protocols http.Protocols
				if proto == "h2" {
					protocols.SetHTTP2(true)
				} else {
					protocols.SetHTTP1(true)
				}
				transport.Protocols = &protocols
				b.Cleanup(transport.CloseIdleConnections)

				client := &http.Client{Transport: transport}
				req, err := http.NewRequest(http.MethodGet, "https://"+addr+"/bench?x=1", nil)
				require.NoError(b, err)
				req.Host = "example.com"
				for _, f := range fingerprintH2Fields[4:] {
					req.Header.Add(f.Name, f.Value)
				}

				// Warm up the connections outside of the measure.
				for range 100 {
					benchRoundtrip(b, client, req)
				}

				startUser, startSys := server.cpu(b)
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						benchRoundtrip(b, client, req)
					}
				})
				b.StopTimer()
				endUser, endSys := server.cpu(b)
				b.ReportMetric(float64(endUser-startUser+endSys-startSys)/float64(b.N), "server-cpu-ns/op")
				b.ReportMetric(float64(endUser-startUser)/float64(b.N), "server-user-ns/op")
				b.ReportMetric(float64(endSys-startSys)/float64(b.N), "server-sys-ns/op")
			})
		}
	}
}

// TestFingerprintServerProcess is the entry point process of BenchmarkFingerprintEntryPoint.
// It answers each line read on stdin with its CPU time so far, and exits when stdin is closed.
func TestFingerprintServerProcess(t *testing.T) {
	capture := os.Getenv("FINGERPRINT_SERVER_PROCESS")
	if capture == "" {
		t.Skip("only run as the child process of BenchmarkFingerprintEntryPoint")
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	addr := startFingerprintEntryPoint(t, capture == "on")
	fmt.Println("ADDR", addr)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var ru syscall.Rusage
		require.NoError(t, syscall.Getrusage(syscall.RUSAGE_SELF, &ru))
		fmt.Println("CPU", ru.Utime.Nano(), ru.Stime.Nano())
	}
}

type fingerprintServerProcess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
}

func startFingerprintServerProcess(b *testing.B, enabled bool) (string, *fingerprintServerProcess) {
	b.Helper()

	capture := "off"
	if enabled {
		capture = "on"
	}

	args := []string{"-test.run=^TestFingerprintServerProcess$", "-test.count=1"}
	if profile := os.Getenv("FINGERPRINT_SERVER_CPUPROFILE"); profile != "" {
		args = append(args, "-test.cpuprofile="+profile+"."+capture)
	}
	// The export destination discards the batches, outside of the measured process.
	dest := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = io.Copy(io.Discard, req.Body)
	}))
	b.Cleanup(dest.Close)

	cmd := exec.Command(os.Args[0], args...)
	exportURL := dest.URL
	if os.Getenv("FINGERPRINT_BENCH_NO_EXPORT") != "" {
		exportURL = ""
	}
	cmd.Env = append(os.Environ(), "FINGERPRINT_SERVER_PROCESS="+capture, "FINGERPRINT_EXPORT_URL="+exportURL)
	stdin, err := cmd.StdinPipe()
	require.NoError(b, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(b, err)
	require.NoError(b, cmd.Start())

	p := &fingerprintServerProcess{cmd: cmd, stdin: stdin, scanner: bufio.NewScanner(stdout)}
	b.Cleanup(func() {
		_ = stdin.Close()
		_ = cmd.Wait()
	})

	for p.scanner.Scan() {
		if addr, ok := strings.CutPrefix(p.scanner.Text(), "ADDR "); ok {
			return addr, p
		}
	}

	b.Fatal("the entry point process did not start")
	return "", nil
}

// cpu returns the user and system CPU time used so far by the entry point process.
func (p *fingerprintServerProcess) cpu(b *testing.B) (time.Duration, time.Duration) {
	b.Helper()

	_, err := io.WriteString(p.stdin, "cpu\n")
	require.NoError(b, err)

	for p.scanner.Scan() {
		if v, ok := strings.CutPrefix(p.scanner.Text(), "CPU "); ok {
			userTime, sysTime, _ := strings.Cut(v, " ")
			u, err := strconv.ParseInt(userTime, 10, 64)
			require.NoError(b, err)
			s, err := strconv.ParseInt(sysTime, 10, 64)
			require.NoError(b, err)
			return time.Duration(u), time.Duration(s)
		}
	}

	b.Fatal("the entry point process did not report its CPU time")
	return 0, 0
}

func benchRoundtrip(b *testing.B, client *http.Client, req *http.Request) {
	b.Helper()

	resp, err := client.Do(req)
	if err != nil {
		b.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

func echoFingerprintHeaders(rw http.ResponseWriter, req *http.Request) {
	out := map[string]string{}
	for name := range req.Header {
		if strings.HasPrefix(name, "X-Fs-") {
			out[name] = req.Header.Get(name)
		}
	}
	if err := json.NewEncoder(rw).Encode(out); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
	}
}

// TestFingerprintEntryPoint_routers covers the whole entry point: the accept wrapper, the TCP router reading the
// ClientHello for the SNI routing, the TLS termination, net/http, and the HTTP/3 server.
func TestFingerprintEntryPoint_routers(t *testing.T) {
	certContent, err := localhostCert.Read()
	require.NoError(t, err)
	keyContent, err := localhostKey.Read()
	require.NoError(t, err)
	tlsCert, err := tls.X509KeyPair(certContent, keyContent)
	require.NoError(t, err)

	// The same free port serves TCP and UDP.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	epConfig := &static.EntryPointsTransport{}
	epConfig.SetDefaults()
	config := &static.EntryPoint{
		Address:          addr,
		Transport:        epConfig,
		ForwardedHeaders: &static.ForwardedHeaders{},
		HTTP2:            &static.HTTP2Config{},
		HTTP3:            &static.HTTP3Config{},
		Fingerprint:      &static.Fingerprint{HeaderPrefix: "X-Fs-"},
	}

	entryPoint, err := NewTCPEntryPoint(t.Context(), "websecure", config, nil, nil)
	require.NoError(t, err)

	router, err := tcprouter.NewRouter(nil)
	require.NoError(t, err)
	router.AddHTTPTLSConfig("example.com", &tls.Config{Certificates: []tls.Certificate{tlsCert}, NextProtos: []string{"h2", "http/1.1"}}, traefiktls.DefaultTLSConfigName)
	router.SetHTTPSHandler(http.HandlerFunc(echoFingerprintHeaders), nil)

	go entryPoint.Start(t.Context())
	entryPoint.SwitchRouter(router)
	t.Cleanup(func() { entryPoint.Shutdown(context.Background()) })

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(certContent)
	tlsConf := &tls.Config{RootCAs: certPool, ServerName: "example.com"}

	get := func(t *testing.T, rt http.RoundTripper) map[string]string {
		t.Helper()

		req, err := http.NewRequest(http.MethodGet, "https://"+addr+"/", nil)
		require.NoError(t, err)
		req.Host = "example.com"
		req.Header.Set("X-Fs-Src-Ip", "6.6.6.6")

		resp, err := rt.RoundTrip(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		var got map[string]string
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		return got
	}

	t.Run("h2", func(t *testing.T) {
		var protocols http.Protocols
		protocols.SetHTTP2(true)
		transport := &http.Transport{TLSClientConfig: tlsConf.Clone(), Protocols: &protocols}
		t.Cleanup(transport.CloseIdleConnections)

		got := get(t, transport)
		assert.Equal(t, "127.0.0.1", got["X-Fs-Src-Ip"])
		assert.Equal(t, "h2", got["X-Fs-Proto"])
		assert.Len(t, got["X-Fs-Conn-Id"], 26)
		// The SNI is in the ClientHello read by the accept wrapper.
		assert.Regexp(t, `^t13d\d{4}h2_`, got["X-Fs-Ja4"])
		assert.Regexp(t, `^[0-9a-f]{32}$`, got["X-Fs-Ja3"])
		assert.NotEmpty(t, got["X-Fs-H2"])
		assert.Regexp(t, `^ge20`, got["X-Fs-Ja4h"])
	})

	t.Run("h3", func(t *testing.T) {
		transport := &http3.Transport{TLSClientConfig: tlsConf.Clone()}
		t.Cleanup(func() { _ = transport.Close() })

		var got map[string]string
		// The HTTP/3 server starts with the entry point, in its own goroutine.
		require.Eventually(t, func() bool {
			req, err := http.NewRequest(http.MethodGet, "https://"+addr+"/", nil)
			require.NoError(t, err)
			req.Host = "example.com"
			req.Header.Set("X-Fs-Src-Ip", "6.6.6.6")
			resp, err := transport.RoundTrip(req)
			if err != nil {
				return false
			}
			defer func() { _ = resp.Body.Close() }()
			return json.NewDecoder(resp.Body).Decode(&got) == nil
		}, 5*time.Second, 100*time.Millisecond)

		assert.Equal(t, "127.0.0.1", got["X-Fs-Src-Ip"])
		assert.Equal(t, "h3", got["X-Fs-Proto"])
		assert.Len(t, got["X-Fs-Conn-Id"], 26)
		// The HTTP/3 header order and the QUIC ClientHello are not captured yet.
		assert.NotContains(t, got, "X-Fs-Ja4h")
		assert.NotContains(t, got, "X-Fs-Ja4")
	})
}
