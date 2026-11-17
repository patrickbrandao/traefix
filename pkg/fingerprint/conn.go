package fingerprint

import (
	"context"
	"crypto/tls"
	"net"
)

// Config configures the capture of a connection.
type Config struct {
	// MaxDecoderHeaderTableSize is the HPACK dynamic table size the HTTP/2 server advertises,
	// which bounds the table the client encoder uses.
	MaxDecoderHeaderTableSize uint32
}

// WrapConn wraps a connection handed to the HTTP server so that the bytes the server reads from it are observed.
// The bytes are only read, never modified, delayed or consumed.
func WrapConn(c net.Conn, cfg Config) net.Conn {
	capture := newCapture(cfg)

	// net/http recognizes a TLS connection through the ConnectionState and HandshakeContext methods,
	// which only the TLS flavor exposes, so that a plain connection is never mistaken for a TLS one.
	if tc, ok := c.(*tls.Conn); ok {
		return &tlsConn{conn{Conn: c, capture: capture}, tc}
	}

	return &conn{Conn: c, capture: capture}
}

// UnwrapConn returns the connection given to WrapConn, or c itself when it is not a wrapped connection.
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

type captureKey struct{}

// ContextWithConn attaches the capture of a wrapped connection to the context.
func ContextWithConn(ctx context.Context, c net.Conn) context.Context {
	var capture *Capture
	switch wc := c.(type) {
	case *conn:
		capture = wc.capture
	case *tlsConn:
		capture = wc.capture
	default:
		return ctx
	}

	return context.WithValue(ctx, captureKey{}, capture)
}

func captureFromContext(ctx context.Context) *Capture {
	capture, _ := ctx.Value(captureKey{}).(*Capture)
	return capture
}

type conn struct {
	net.Conn

	capture *Capture
}

func (c *conn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		c.capture.observe(p[:n])
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
