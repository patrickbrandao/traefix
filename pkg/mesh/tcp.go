package mesh

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/tcp"
)

// forwardTCP is a connection of a TCP router: its bytes travel in a stream of the link.
const forwardTCP = "tcp"

// meshTCPService forwards the connections of a TCP router learned from the mesh to an instance that owns it.
type meshTCPService struct {
	mesh *Mesh
	// name is the name of the service, which is the one of the router of this instance.
	name string
}

func (s *meshTCPService) ServeTCP(conn tcp.WriteCloser) {
	target := s.mesh.tables.Load().tcpTargets[s.name]
	if target == nil {
		s.mesh.refuseTCP(conn, "no instance of the mesh announces the router")
		return
	}

	s.mesh.dispatchTCP(conn, target, nil)
}

// sharedTCPRoute decides, for a TCP router that this instance and other instances have, where a connection is served.
// The middlewares of the router run where the connection is served.
type sharedTCPRoute struct {
	mesh *Mesh
	name string
	// local runs the middlewares of the router and its service on this instance.
	local tcp.Handler
}

func (s *sharedTCPRoute) ServeTCP(conn tcp.WriteCloser) {
	// Most routers live on one instance: nothing is decided for them.
	target := s.mesh.tables.Load().tcpShared[s.name]
	if target == nil {
		s.local.ServeTCP(conn)
		return
	}

	s.mesh.dispatchTCP(conn, target, s.local)
}

// LocalTCPRouters returns the TCP routers the other instances may send connections for, as their providers wrote them.
func (m *Mesh) LocalTCPRouters() map[string]*dynamic.TCPRouter {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.localTCP
}

// LocalTCPRoute tells whether a TCP router of the routing configuration is one this instance announces to the mesh.
func (m *Mesh) LocalTCPRoute(routerName string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.localTCP[routerName]

	return ok
}

// DecideTCP returns the handler that serves a connection for a TCP router this instance announces:
// on this instance, with the given handler, or on another instance that has the router, as the priorities say.
func (m *Mesh) DecideTCP(name string, local tcp.Handler) tcp.Handler {
	return &sharedTCPRoute{mesh: m, name: name, local: local}
}

// dispatchTCP serves a connection where the priorities say, and falls back to the next place when one cannot take it.
// A stream that opens means that the other instance has the router and took the connection: until then,
// nothing of the connection was sent, and any other place can serve it.
func (m *Mesh) dispatchTCP(conn tcp.WriteCloser, target *forwardTarget, local tcp.Handler) {
	remote := conn.RemoteAddr().String()

	client := remote
	if host, _, err := net.SplitHostPort(remote); err == nil {
		client = host
	}

	info := forwardInfo{Version: 1, Kind: forwardTCP, Router: target.router, Remote: remote, Local: conn.LocalAddr().String()}

	for _, next := range m.plan(target, client) {
		if next.local {
			local.ServeTCP(conn)
			return
		}

		session := m.sessionTo(next.nodeID)
		if session == nil {
			// The link went down since the places were chosen.
			continue
		}

		stream, release, err := openTunnel(m, session.OpenStream, http.Header{headerForward: {info.encode()}})
		if err != nil {
			m.failed(target.key(), next.nodeID)
			m.countRequest(next.nodeID, forwardTCP, resultError)
			m.logger.Debug().Err(err).Str("peer", next.nodeID).Str("router", target.router).
				Msg("Unable to open a stream for a connection through the mesh")

			continue
		}

		m.succeeded(target.key(), next.nodeID)
		m.countRequest(next.nodeID, forwardTCP, resultForwarded)

		carry(conn, stream)
		release()

		return
	}

	m.refuseTCP(conn, "no instance that owns the router can be reached")
}

// refuseTCP closes a connection that could not be forwarded: TCP has no other way to say so.
func (m *Mesh) refuseTCP(conn tcp.WriteCloser, cause string) {
	m.countRequest("", forwardTCP, resultUnavailable)

	m.logger.Debug().Str("client", conn.RemoteAddr().String()).Str("cause", cause).Msg("Unable to forward a connection through the mesh")

	_ = conn.Close()
}

// openTunnel opens a stream or a flow that lives for as long as the mesh runs, or until it is released.
// A connection or a UDP session outlives the routing configuration that accepted it, as it does with the services
// of Traefik: its tunnel is not bound to the context the routing was built with.
func openTunnel[T io.Closer](m *Mesh, open func(ctx context.Context, header http.Header) (T, error), header http.Header) (T, func(), error) {
	var zero T

	m.mu.Lock()
	parent := m.runCtx
	m.mu.Unlock()

	if parent == nil {
		return zero, nil, context.Canceled
	}

	ctx, cancel := context.WithCancel(parent)

	// The context of a tunnel bounds its life: the time to open it is bounded apart.
	opening := time.AfterFunc(m.linkTimeout, cancel)

	tunnel, err := open(ctx, header)
	if err == nil && !opening.Stop() {
		_ = tunnel.Close()
		err = context.DeadlineExceeded
	}

	if err != nil {
		opening.Stop()
		cancel()

		return zero, nil, err
	}

	return tunnel, cancel, nil
}

// carry copies the bytes between the connection of a client and a stream to the instance that serves it,
// until both directions are done, and closes both.
// Each side may stop sending and keep receiving, as the two ends of a TCP connection may.
func carry(conn tcp.WriteCloser, stream Stream) {
	sent := make(chan struct{})

	go func() {
		defer close(sent)

		_, _ = io.Copy(stream, conn)
		// The client is done sending: the server is told so, and may still answer.
		_ = stream.CloseWrite()
	}()

	_, _ = io.Copy(conn, stream)
	// The server is done sending: the client is told so, and may still send, for as long as the other instance listens.
	_ = conn.CloseWrite()

	select {
	case <-sent:
	case <-stream.Done():
	}

	// Closing both ends a copy that still waits for the client.
	_ = conn.Close()
	_ = stream.Close()

	<-sent
}

// serveTCP serves a connection that another instance received and forwarded: the stream is the connection of the client,
// for the middlewares of the router and its service.
func (m *Mesh) serveTCP(rw http.ResponseWriter, req *http.Request, info forwardInfo, handler tcp.Handler) {
	stream, err := acceptStream(rw, req)
	if err != nil {
		return
	}

	conn := newTunnelConn(stream, info.Local, info.Remote)

	// The handler returns once the connection is over, and nothing may use the stream afterwards.
	handler.ServeTCP(conn)

	_ = conn.Close()
}

// tunnelConn makes a stream of a link the connection of a client, for the instance that serves it.
// It has the addresses of the connection the client opened to the instance that received it, which is what
// the allow lists read, and what the PROXY protocol tells the server.
type tunnelConn struct {
	stream Stream
	local  net.Addr
	remote net.Addr

	once sync.Once

	// The services of Traefik set the deadlines of a connection after they say they are done with it, e.g. for the
	// termination delay. The stream belongs to the request of the link, which is over once its handler returns:
	// no deadline reaches it after Close, and Close waits for the ones under way.
	deadlines sync.RWMutex
	closed    bool
}

func newTunnelConn(stream Stream, local, remote string) *tunnelConn {
	return &tunnelConn{stream: stream, local: tcpAddr(local), remote: tcpAddr(remote)}
}

// tcpAddr reads an address as the one of a TCP connection, which is what the handlers expect of a client.
func tcpAddr(address string) net.Addr {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return meshAddr(address)
	}

	ip := net.ParseIP(host)
	number, err := strconv.Atoi(port)

	if ip == nil || err != nil {
		return meshAddr(address)
	}

	return &net.TCPAddr{IP: ip, Port: number}
}

func (c *tunnelConn) Read(p []byte) (int, error)  { return c.stream.Read(p) }
func (c *tunnelConn) Write(p []byte) (int, error) { return c.stream.Write(p) }
func (c *tunnelConn) CloseWrite() error           { return c.stream.CloseWrite() }
func (c *tunnelConn) LocalAddr() net.Addr         { return c.local }
func (c *tunnelConn) RemoteAddr() net.Addr        { return c.remote }

// Close may be called more than once, by the handlers of the chain: the stream is closed once.
func (c *tunnelConn) Close() error {
	var err error

	c.once.Do(func() {
		c.deadlines.Lock()
		c.closed = true
		c.deadlines.Unlock()

		err = c.stream.Close()
	})

	return err
}

func (c *tunnelConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}

	return c.SetWriteDeadline(t)
}

func (c *tunnelConn) SetReadDeadline(t time.Time) error {
	return c.setDeadline(func(stream deadliner) error { return stream.SetReadDeadline(t) })
}

func (c *tunnelConn) SetWriteDeadline(t time.Time) error {
	return c.setDeadline(func(stream deadliner) error { return stream.SetWriteDeadline(t) })
}

func (c *tunnelConn) setDeadline(set func(stream deadliner) error) error {
	stream, ok := c.stream.(deadliner)
	if !ok {
		return os.ErrNoDeadline
	}

	c.deadlines.RLock()
	defer c.deadlines.RUnlock()

	if c.closed {
		return net.ErrClosed
	}

	return set(stream)
}
