package fingerprint

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/textproto"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gokitmetrics "github.com/go-kit/kit/metrics"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/config/static"
	"github.com/traefik/traefik/v3/pkg/observability/metrics"
	"github.com/traefik/traefik/v3/pkg/tcp"
	"golang.org/x/net/http/httpguts"
)

// enabled tells whether an entry point has the fingerprint option, so that the routers skip their hook otherwise.
var enabled atomic.Bool

// Enabled reports whether an entry point has the fingerprint option.
func Enabled() bool {
	return enabled.Load()
}

type headerNames struct {
	srcIP, srcPort, dstPort, connID, proto, ja3, ja4, h2, ja4h, alpn, ja4t string
	serverIP, serverPort, timestamp, ja3n, ttl, transport                  string
}

// EntryPoint identifies and fingerprints the connections and requests of an entry point.
type EntryPoint struct {
	name string

	prefix  string
	headers headerNames

	captureCfg       Config
	rawBytes         bool
	http3HeaderOrder bool
	tcp              bool
	exporter         *exporter

	quic quicPending

	truncatedCounter gokitmetrics.Counter

	mu sync.Mutex
	// pending holds the connections accepted and not yet handed to the HTTP server, by address pair.
	pending map[string]*connInfo
}

// NewEntryPoint validates the fingerprint option of an entry point, and builds what applies it.
func NewEntryPoint(ctx context.Context, name string, cfg *static.Fingerprint, h2 *static.HTTP2Config, registry metrics.Registry) (*EntryPoint, error) {
	if cfg == nil {
		return nil, errors.New("missing fingerprint configuration")
	}
	if registry == nil {
		registry = metrics.NewVoidRegistry()
	}

	prefix := cfg.HeaderPrefix
	if prefix == "" || !httpguts.ValidHeaderFieldName(prefix) {
		return nil, fmt.Errorf("invalid fingerprint header prefix %q: it must be a non-empty header name", prefix)
	}

	e := &EntryPoint{
		name:             name,
		prefix:           strings.ToLower(strings.ReplaceAll(prefix, "_", "-")),
		truncatedCounter: registry.FingerprintCaptureTruncatedCounter(),
		pending:          make(map[string]*connInfo),
		http3HeaderOrder: cfg.HTTP3HeaderOrder,
		tcp:              cfg.TCP,
	}
	if e.tcp && !savedSYNSupported {
		log.Ctx(ctx).Warn().Str("entryPointName", name).
			Msg("The fingerprint tcp option needs the saved SYN of Linux: the connections get no TCP fingerprint")
		e.tcp = false
	}
	e.headers = headerNames{
		srcIP:   textproto.CanonicalMIMEHeaderKey(prefix + "Src-Ip"),
		srcPort: textproto.CanonicalMIMEHeaderKey(prefix + "Src-Port"),
		dstPort: textproto.CanonicalMIMEHeaderKey(prefix + "Dst-Port"),
		connID:  textproto.CanonicalMIMEHeaderKey(prefix + "Conn-Id"),
		proto:   textproto.CanonicalMIMEHeaderKey(prefix + "Proto"),
		ja3:     textproto.CanonicalMIMEHeaderKey(prefix + "Ja3"),
		ja4:     textproto.CanonicalMIMEHeaderKey(prefix + "Ja4"),
		h2:      textproto.CanonicalMIMEHeaderKey(prefix + "H2"),
		ja4h:    textproto.CanonicalMIMEHeaderKey(prefix + "Ja4h"),
		alpn:    textproto.CanonicalMIMEHeaderKey(prefix + "Alpn"),
		ja4t:    textproto.CanonicalMIMEHeaderKey(prefix + "Ja4t"),

		serverIP:   textproto.CanonicalMIMEHeaderKey(prefix + "Server-Ip"),
		serverPort: textproto.CanonicalMIMEHeaderKey(prefix + "Server-Port"),
		timestamp:  textproto.CanonicalMIMEHeaderKey(prefix + "Time"),
		ja3n:       textproto.CanonicalMIMEHeaderKey(prefix + "Ja3n"),
		ttl:        textproto.CanonicalMIMEHeaderKey(prefix + "Ttl"),
		transport:  textproto.CanonicalMIMEHeaderKey(prefix + "Transport"),
	}

	if h2 != nil && h2.MaxDecoderHeaderTableSize > 0 {
		e.captureCfg.MaxDecoderHeaderTableSize = uint32(h2.MaxDecoderHeaderTableSize)
	}

	if cfg.Export != nil {
		xcfg, err := exportConfig(name, cfg.Export)
		if err != nil {
			return nil, err
		}

		e.rawBytes = cfg.Export.RawBytes
		e.captureCfg.fields = newFieldMatcher(cfg.Export.Fields, cfg.Export.AllowSensitiveFields)
		if sensitive := e.captureCfg.fields.sensitive(); cfg.Export.AllowSensitiveFields && len(sensitive) > 0 {
			log.Ctx(ctx).Warn().Strs("fields", sensitive).
				Msg("The fingerprint export sends the values of headers carrying credentials, as allowSensitiveFields is set")
		}

		e.exporter = newExporter(xcfg, registry)
	}

	enabled.Store(true)
	entryPoints.Store(name, struct{}{})

	return e, nil
}

func exportConfig(name string, cfg *static.FingerprintExport) (exporterConfig, error) {
	u, err := url.Parse(cfg.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return exporterConfig{}, fmt.Errorf("invalid fingerprint export URL %q: it must be an absolute http or https URL", cfg.URL)
	}

	switch {
	case cfg.BatchSize < 1 || cfg.BatchSize > maxBatchRecords:
		return exporterConfig{}, fmt.Errorf("invalid fingerprint export batchSize %d: it must be between 1 and %d", cfg.BatchSize, maxBatchRecords)
	case cfg.QueueSize < 1:
		return exporterConfig{}, fmt.Errorf("invalid fingerprint export queueSize %d: it must be positive", cfg.QueueSize)
	case cfg.QueueMaxBytes < 1:
		return exporterConfig{}, fmt.Errorf("invalid fingerprint export queueMaxBytes %d: it must be positive", cfg.QueueMaxBytes)
	case cfg.FlushInterval <= 0:
		return exporterConfig{}, errors.New("invalid fingerprint export flushInterval: it must be positive")
	case cfg.SendTimeout <= 0:
		return exporterConfig{}, errors.New("invalid fingerprint export sendTimeout: it must be positive")
	}

	headers := make(http.Header, len(cfg.Headers))
	for k, v := range cfg.Headers {
		if !httpguts.ValidHeaderFieldName(k) || !httpguts.ValidHeaderFieldValue(v) {
			return exporterConfig{}, fmt.Errorf("invalid fingerprint export header %q", k)
		}
		headers.Set(k, v)
	}

	return exporterConfig{
		EntryPoint:    name,
		URL:           cfg.URL,
		Headers:       headers,
		QueueSize:     cfg.QueueSize,
		QueueMaxBytes: cfg.QueueMaxBytes,
		BatchSize:     cfg.BatchSize,
		FlushInterval: time.Duration(cfg.FlushInterval),
		SendTimeout:   time.Duration(cfg.SendTimeout),
	}, nil
}

// Close stops the export after a last attempt to send the queued records, bounded by ctx.
func (e *EntryPoint) Close(ctx context.Context) {
	if e.exporter != nil {
		e.exporter.close(ctx)
	}
}

// SaveSYN asks the kernel to keep the SYN of the connections the listener accepts, when the tcp option is set.
func (e *EntryPoint) SaveSYN(l *net.TCPListener) error {
	if !e.tcp {
		return nil
	}
	return saveSYN(l)
}

// Accept wraps a connection as it is accepted, before the TCP router reads its first bytes.
func (e *EntryPoint) Accept(c tcp.WriteCloser) tcp.WriteCloser {
	info := newConnInfo(c)
	key := connKey(c)

	// A connection that went through the PROXY protocol is not a *net.TCPConn: its SYN, the one of the proxy,
	// would not tell about the client.
	if tc, ok := c.(*net.TCPConn); ok && e.tcp {
		syn, rtt, err := readSYN(tc)
		if err != nil {
			log.Debug().Err(err).Msg("Reading the saved SYN")
		}
		info.syn, info.rtt = syn, rtt
		info.tcpConn = tc
	}

	e.mu.Lock()
	e.pending[key] = info
	e.mu.Unlock()

	return &acceptedConn{WriteCloser: c, ep: e, info: info, key: key, capturing: true}
}

// WrapConn wraps a connection handed to the HTTP server, whose decrypted bytes are captured.
func (e *EntryPoint) WrapConn(c net.Conn) net.Conn {
	key := connKey(c)

	e.mu.Lock()
	info := e.pending[key]
	delete(e.pending, key)
	e.mu.Unlock()

	if info == nil {
		info = newConnInfo(c)
	}

	info.mu.Lock()
	info.claimed = true
	info.mu.Unlock()

	info.capture = newCapture(e.captureCfg)

	return wrapConn(c, info)
}

// ConnContext attaches the connection data to the context of its requests.
func (e *EntryPoint) ConnContext(ctx context.Context, c net.Conn) context.Context {
	if info := connInfoOf(c); info != nil {
		return context.WithValue(ctx, connInfoKey{}, info)
	}
	return ctx
}

// ConnClosed is called when the HTTP server is done with a connection.
func (e *EntryPoint) ConnClosed(c net.Conn) {
	if info := connInfoOf(c); info != nil {
		proto := ""
		if info.capture != nil {
			proto = info.capture.Proto()
		}
		e.finishConn(info, proto)
	}
}

// HTTP3HeaderOrder reports whether the QUIC connections need the qlog tracer of the fingerprint.
func (e *EntryPoint) HTTP3HeaderOrder() bool {
	return e.http3HeaderOrder
}

// QUICConnContext identifies an HTTP/3 connection, and attaches its data to the context of its requests.
// The connection keeps its identifier when it migrates to another path, as it is bound to the connection.
func (e *EntryPoint) QUICConnContext(ctx context.Context, c *quic.Conn) context.Context {
	info := newConnInfo(c)
	info.quic = c
	info.claimed = true

	if h := e.quic.take(c.RemoteAddr().String()); h != nil {
		info.parsedHello = h.fromInfo
		if h.fromRaw != nil {
			info.parsedHello = h.fromRaw
		}
		info.hello = h.raw
		info.helloTruncated = h.truncated
		info.quicParams = h.params
		info.quicInitial = h.initial
	}
	if t, ok := c.QlogTrace().(*quicTrace); ok {
		info.h3 = t
	}

	context.AfterFunc(c.Context(), func() { e.finishConn(info, "h3") })

	return context.WithValue(ctx, connInfoKey{}, info)
}

func (e *EntryPoint) truncated(part string) {
	e.truncatedCounter.With("entrypoint", e.name, "part", part).Add(1)
}

// forget is called when an accepted connection closes. A TLS connection that never reached the HTTP server,
// e.g. closed during the handshake, still gets its connection record.
func (e *EntryPoint) forget(key string, info *connInfo) {
	e.mu.Lock()
	if e.pending[key] == info {
		delete(e.pending, key)
	}
	e.mu.Unlock()

	info.mu.Lock()
	export := !info.claimed && len(info.hello) > 0
	info.mu.Unlock()

	if export {
		e.finishConn(info, "")
	}
}

// finishConn computes the connection data once, and exports its record.
func (e *EntryPoint) finishConn(info *connInfo, proto string) {
	info.finalize.Do(func() {
		e.finalizeConn(info, proto)
	})
}

func (e *EntryPoint) finalizeConn(info *connInfo, proto string) {
	info.proto = proto

	rec := &connectionRecord{
		Type:       recordConnection,
		ConnID:     info.id,
		Time:       info.accepted.UTC(),
		EntryPoint: e.name,
		Src:        addrRecord{IP: info.src.Addr().String(), Port: info.src.Port()},
		Dst:        addrRecord{IP: info.dst.Addr().String(), Port: info.dst.Port()},
		Proto:      proto,
	}

	info.mu.Lock()
	hello, raw, truncated := info.parsedHello, info.hello, info.helloTruncated
	info.mu.Unlock()

	var state tls.ConnectionState
	switch {
	case info.tls != nil:
		state = info.tls.ConnectionState()
	case info.quic != nil:
		state = info.quic.ConnectionState().TLS
	}

	if hello != nil || len(raw) > 0 || state.HandshakeComplete {
		t := &tlsRecord{ClientHelloTruncated: truncated}
		if hello != nil {
			info.ja3Str = hello.JA3()
			info.ja3 = md5Hex(info.ja3Str)
			ja4 := hello.JA4Variants(info.quic != nil)
			info.ja4 = ja4.JA4
			info.alpn = alpnHeaderValue(hello.ALPN)
			t.SNI = hello.ServerName
			t.ALPNOffered = hello.ALPN
			t.JA3, t.JA3String, t.JA4 = info.ja3, info.ja3Str, info.ja4
			t.JA3NString = hello.JA3N()
			t.JA3N = md5Hex(t.JA3NString)
			info.ja3n = t.JA3N
			t.JA4R, t.JA4O = ja4.R, ja4.O
		}
		if state.HandshakeComplete {
			if t.SNI == "" {
				t.SNI = state.ServerName
			}
			t.ALPN = state.NegotiatedProtocol
			t.Version = tls.VersionName(state.Version)
			t.Cipher = tls.CipherSuiteName(state.CipherSuite)
		}
		if e.rawBytes {
			t.ClientHello = raw
		}
		t.QUICTransportParameters = info.quicParams
		rec.TLS = t
	}

	if info.capture != nil {
		if prefix, prefixTruncated := info.capture.H2Prefix(); prefix != nil {
			info.akamai = AkamaiH2(prefix, info.capture.FirstPseudoNames())
			h := &http2Record{Akamai: info.akamai, HeadersPriority: H2HeadersPriority(prefix), PrefixTruncated: prefixTruncated}
			if e.rawBytes {
				h.Prefix = prefix
			}
			rec.HTTP2 = h
			if prefixTruncated {
				e.truncated("h2_prefix")
			}
		}
	}

	if info.h3 != nil {
		if settings := info.h3.Settings(); settings != "" {
			rec.HTTP3 = &http3Record{Settings: settings}
		}
	}

	if len(info.syn) > 0 {
		rec.TCP = e.tcpRecord(info)
		info.syn = nil
	}

	info.record = rec

	// The raw bytes are only kept for the record: the queued record holds them until it is sent.
	info.mu.Lock()
	info.hello = nil
	info.parsedHello = nil
	info.mu.Unlock()
	if info.capture != nil {
		info.capture.releaseH2Prefix()
	}

	if e.exporter != nil {
		e.exporter.enqueue(recordConnection, rec, rec.size())
	}
}

// alpnHeaderValue joins the offered protocols in the client order. A protocol name is an opaque byte string:
// the bytes that are not visible ASCII, the comma and the percent sign are percent-encoded, so that the list
// stays a valid header value and splits on the commas.
func alpnHeaderValue(protocols []string) string {
	var b strings.Builder
	for i, p := range protocols {
		if i > 0 {
			b.WriteByte(',')
		}
		for j := range len(p) {
			c := p[j]
			if c <= ' ' || c >= 0x7f || c == ',' || c == '%' {
				const upperHex = "0123456789ABCDEF"
				b.Write([]byte{'%', upperHex[c>>4], upperHex[c&0xf]})
				continue
			}
			b.WriteByte(c)
		}
	}
	return b.String()
}

// tcpRecord reads the saved SYN of the connection. A SYN from another address than the one of the connection,
// which no known setup gives, is left out.
func (e *EntryPoint) tcpRecord(info *connInfo) *tcpRecord {
	syn, err := ParseSYN(info.syn)
	if err != nil {
		log.Debug().Err(err).Msg("Parsing the saved SYN")
		return nil
	}
	if src, ok := netip.AddrFromSlice(syn.Src); !ok || src.Unmap() != info.src.Addr() {
		log.Debug().Str("src", info.src.Addr().String()).Msg("Ignoring a saved SYN from another address")
		return nil
	}

	info.ja4t = syn.JA4T
	info.synView = syn
	rec := &tcpRecord{JA4T: syn.JA4T, P0F: syn.P0F, TTL: syn.TTL, RTTMicros: info.rtt.Microseconds()}
	if e.rawBytes {
		rec.SYN = info.syn
	}

	return rec
}
