package fingerprint

import (
	"context"
	"crypto/tls"
	"net"
	"net/netip"
	"strconv"
	"sync"
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
	srcIP, srcPort, dstPort string

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

	// finalize computes the connection record once, on the first request or when the connection closes.
	finalize sync.Once
	proto    string
	ja3      string
	ja3Str   string
	ja4      string
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
}

func (c *acceptedConn) Read(p []byte) (int, error) {
	n, err := c.WriteCloser.Read(p)
	if n > 0 && c.capturing {
		c.captureHello(p[:n])
	}

	return n, err
}

func (c *acceptedConn) Close() error {
	c.closeOnce.Do(func() {
		c.ep.forget(c.key, c.info)
	})
	return c.WriteCloser.Close()
}

func (c *acceptedConn) captureHello(b []byte) {
	info := c.info
	info.mu.Lock()
	defer info.mu.Unlock()

	if len(info.hello) == 0 && b[0] != recordTypeHandshake {
		c.capturing = false
		return
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
		return
	}

	if len(info.hello) >= maxClientHello {
		info.helloTruncated = true
		c.capturing = false
		c.ep.truncated("client_hello")
	}
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
	return c.tls.HandshakeContext(ctx)
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
