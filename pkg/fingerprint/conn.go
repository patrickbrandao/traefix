package fingerprint

import (
	"context"
	"crypto/tls"
	"net"
	"net/netip"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/traefik/traefik/v3/pkg/tcp"
)

// Config configures the capture of a connection.
type Config struct {
	// MaxDecoderHeaderTableSize is the HPACK dynamic table size the HTTP/2 server advertises,
	// which bounds the table the client encoder uses.
	MaxDecoderHeaderTableSize uint32

	fields *fieldMatcher
}

const (
	// maxClientHello bounds the ClientHello bytes kept; a longer ClientHello is kept truncated and not fingerprinted.
	maxClientHello = 16 << 10

	recordTypeHandshake = 0x16
)

// connInfo holds what is known of a connection, shared by its accept and HTTP wrappers and by its requests.
type connInfo struct {
	id       string
	accepted time.Time
	src, dst netip.AddrPort

	// The header values of the connection, formatted once.
	srcIP, srcPort, dstPort, dstIP string

	mu             sync.Mutex
	hello          []byte
	helloTruncated bool
	parsedHello    *ClientHello
	claimed        bool

	// Set by the HTTP layer before the connection is served.
	capture *Capture
	tls     *tls.Conn
	quic    *quic.Conn
	h3      *quicTrace

	quicParams []uint64

	// syn is the SYN the kernel saved for the connection, and rtt the round-trip time of the handshake.
	syn []byte
	rtt time.Duration
	// synView is the parsed SYN, and tcpConn the socket whose TCP_INFO the transport header reads.
	synView *SYN
	tcpConn *net.TCPConn

	quicInitial quicInitialInfo
	// quicIP is the IP header of the first Initial, as the QUIC library reports it.
	quicIP quic.IPHeaderInfo

	// detail holds the detail header values, formatted once.
	detail *connDetail

	// timing holds the milestones of the start of the connection, protected by mu.
	timing connTiming

	// requests counts the requests of the connection, for the transport header.
	requests atomic.Int64

	// finalize computes the connection record once, on the first request or when the connection closes.
	finalize sync.Once
	proto    string
	ja3      string
	ja3Str   string
	ja4      string
	alpn     string
	ja3n     string
	ja4t     string
	akamai   string
	record   *connectionRecord
}

func newConnInfo(c interface {
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
},
) *connInfo {
	now := time.Now()
	info := &connInfo{
		id:       newULID(now),
		accepted: now,
		src:      addrPort(c.RemoteAddr()),
		dst:      addrPort(c.LocalAddr()),
	}
	info.srcIP = info.src.Addr().String()
	info.srcPort = strconv.Itoa(int(info.src.Port()))
	info.dstPort = strconv.Itoa(int(info.dst.Port()))
	// A socket bound to all the addresses gives no destination address, e.g. the UDP socket of HTTP/3.
	if a := info.dst.Addr(); a.IsValid() && !a.IsUnspecified() {
		info.dstIP = a.String()
	}

	return info
}

func addrPort(a net.Addr) netip.AddrPort {
	var ap netip.AddrPort
	switch addr := a.(type) {
	case *net.TCPAddr:
		ap = addr.AddrPort()
	case *net.UDPAddr:
		ap = addr.AddrPort()
	default:
		if a != nil {
			ap, _ = netip.ParseAddrPort(a.String())
		}
	}

	// An IPv4 client of a dual-stack socket shows as an IPv4-mapped IPv6 address.
	return netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port())
}

func connKey(c net.Conn) string {
	return c.RemoteAddr().String() + "|" + c.LocalAddr().String()
}

// acceptedConn is the bottom wrapper of a connection: it records the bytes of the ClientHello as they are read,
// before the TLS termination and the protocol detection read them again from their own buffers.
type acceptedConn struct {
	tcp.WriteCloser

	ep        *EntryPoint
	info      *connInfo
	key       string
	capturing bool
	closeOnce sync.Once

	// tc is the socket read with its kernel times while the handshake is timed (kernelTimed).
	tc          *net.TCPConn
	kernelTimed bool
	timing      bool
}

func (c *acceptedConn) Read(p []byte) (int, error) {
	if !c.timing {
		n, err := c.WriteCloser.Read(p)
		if n > 0 && c.capturing {
			c.captureHello(p[:n])
		}
		return n, err
	}

	var n int
	var at time.Time
	var err error
	if c.kernelTimed {
		n, at, err = readTimed(c.tc, p)
	} else {
		n, err = c.WriteCloser.Read(p)
	}
	if n > 0 {
		if at.IsZero() {
			at = time.Now()
		}
		helloDone := false
		if c.capturing {
			helloDone = c.captureHello(p[:n])
		}
		c.observeRead(at, helloDone)
	}

	return n, err
}

func (c *acceptedConn) Write(p []byte) (int, error) {
	n, err := c.WriteCloser.Write(p)
	if c.timing && n > 0 {
		c.observeWrite(time.Now())
	}
	return n, err
}

func (c *acceptedConn) Close() error {
	c.closeOnce.Do(func() {
		c.ep.forget(c.key, c.info)
	})
	return c.WriteCloser.Close()
}

// observeRead records the arrival milestones, and stops the timing once the server flight got its answer.
func (c *acceptedConn) observeRead(at time.Time, helloDone bool) {
	info := c.info
	info.mu.Lock()
	t := &info.timing
	if t.firstByte.IsZero() {
		t.firstByte = at
	}
	switch {
	case helloDone:
		t.helloDone = at
	case !t.serverFlight.IsZero():
		t.clientFlight = at
		t.done = true
	case t.helloDone.IsZero() && !c.capturing:
		// Not a TLS connection: only the first byte is timed.
		t.done = true
	}
	done := t.done
	info.mu.Unlock()

	if done {
		c.stopTiming()
	}
}

// observeWrite records the end of the server flight: the last write before the client answers.
func (c *acceptedConn) observeWrite(at time.Time) {
	info := c.info
	info.mu.Lock()
	if t := &info.timing; !t.helloDone.IsZero() && t.clientFlight.IsZero() {
		t.serverFlight = at
	}
	info.mu.Unlock()
}

func (c *acceptedConn) stopTiming() {
	if c.kernelTimed {
		disableKernelTimestamps(c.tc)
		c.kernelTimed = false
	}
	c.timing = false
}

// captureHello appends the bytes to the ClientHello, and reports whether they complete it.
func (c *acceptedConn) captureHello(b []byte) bool {
	info := c.info
	info.mu.Lock()
	defer info.mu.Unlock()

	if len(info.hello) == 0 && b[0] != recordTypeHandshake {
		c.capturing = false
		return false
	}

	room := maxClientHello - len(info.hello)
	if len(b) > room {
		b = b[:room]
	}
	info.hello = append(info.hello, b...)

	if n, ok := ClientHelloLen(info.hello); ok {
		info.hello = info.hello[:n:n]
		c.capturing = false
		if hello, err := ParseClientHello(info.hello); err == nil {
			info.parsedHello = hello
		}
		return true
	}

	if len(info.hello) >= maxClientHello {
		info.helloTruncated = true
		c.capturing = false
		c.ep.truncated("client_hello")
	}
	return false
}

// conn wraps a connection handed to the HTTP server so that the bytes the server reads from it are observed.
// The bytes are only read, never modified, delayed or consumed.
type conn struct {
	net.Conn

	info *connInfo
}

func (c *conn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		c.info.capture.observe(p[:n])
	}

	return n, err
}

func (c *conn) CloseWrite() error {
	if cw, ok := c.Conn.(interface{ CloseWrite() error }); ok {
		return cw.CloseWrite()
	}

	return nil
}

// NetConn returns the wrapped connection, like tls.Conn.NetConn does.
func (c *conn) NetConn() net.Conn {
	return c.Conn
}

// tlsConn is the TLS flavor of conn: net/http recognizes a TLS connection through the ConnectionState and
// HandshakeContext methods, which only this flavor exposes, so that a plain connection is never mistaken for a TLS one.
type tlsConn struct {
	conn

	tls *tls.Conn
}

func (c *tlsConn) ConnectionState() tls.ConnectionState {
	return c.tls.ConnectionState()
}

func (c *tlsConn) HandshakeContext(ctx context.Context) error {
	err := c.tls.HandshakeContext(ctx)
	if err == nil {
		c.info.mu.Lock()
		if c.info.timing.handshakeDone.IsZero() {
			c.info.timing.handshakeDone = time.Now()
		}
		c.info.mu.Unlock()
	}
	return err
}

func wrapConn(c net.Conn, info *connInfo) net.Conn {
	if tc, ok := c.(*tls.Conn); ok {
		info.tls = tc
		return &tlsConn{conn{Conn: c, info: info}, tc}
	}

	return &conn{Conn: c, info: info}
}

// UnwrapConn returns the connection given to the HTTP wrapper, or c itself when it is not a wrapped connection.
func UnwrapConn(c net.Conn) net.Conn {
	switch wc := c.(type) {
	case *conn:
		return wc.Conn
	case *tlsConn:
		return wc.Conn
	default:
		return c
	}
}

func connInfoOf(c net.Conn) *connInfo {
	switch wc := c.(type) {
	case *conn:
		return wc.info
	case *tlsConn:
		return wc.info
	default:
		return nil
	}
}

type connInfoKey struct{}

func connInfoFromContext(ctx context.Context) *connInfo {
	info, _ := ctx.Value(connInfoKey{}).(*connInfo)
	return info
}
