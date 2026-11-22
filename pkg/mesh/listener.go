package mesh

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"golang.org/x/net/http2"
)

type peerKey struct{}

// peerFrom returns the identifier of the instance that sent a request, as its certificate tells.
func peerFrom(ctx context.Context) string {
	peer, _ := ctx.Value(peerKey{}).(string)

	return peer
}

// listener accepts the links of the other instances, in UDP for QUIC and in TCP on the same port.
type listener struct {
	identity    *Identity
	linkTimeout time.Duration

	udp *net.UDPConn
	// transport listens and dials on the UDP socket: the links this instance opens leave from the port of the mesh.
	transport *quic.Transport
	quic      *quic.Listener
	tcp       net.Listener

	mu    sync.Mutex
	conns map[net.Conn]struct{}

	// reverse takes the links over TCP that satellites open for this instance to send requests through.
	reverse func(tlsConn *tls.Conn, notify *notifyConn, peer string)
}

// newListener binds the address. With one transport only, the other one is not listened on.
func newListener(address, transport string, identity *Identity, linkTimeout time.Duration) (*listener, error) {
	l := &listener{identity: identity, linkTimeout: linkTimeout, conns: map[net.Conn]struct{}{}}

	if transport != cluster.MeshTransportTCP {
		addr, err := net.ResolveUDPAddr("udp", address)
		if err != nil {
			return nil, fmt.Errorf("resolving the mesh address: %w", err)
		}

		l.udp, err = net.ListenUDP("udp", addr)
		if err != nil {
			return nil, fmt.Errorf("listening on the mesh address in UDP: %w", err)
		}

		l.transport = &quic.Transport{Conn: l.udp}

		l.quic, err = l.transport.Listen(identity.ServerTLS(alpnMesh), quicConfig(linkTimeout))
		if err != nil {
			l.close()
			return nil, fmt.Errorf("listening on the mesh address in QUIC: %w", err)
		}
	}

	if transport != cluster.MeshTransportQUIC {
		var err error

		l.tcp, err = net.Listen("tcp", address)
		if err != nil {
			l.close()
			return nil, fmt.Errorf("listening on the mesh address in TCP: %w", err)
		}
	}

	return l, nil
}

// serve hands the requests of the other instances to the handler until the context is done.
// The handler finds the instance that sent a request with peerFrom.
func (l *listener) serve(ctx context.Context, handler http.Handler, reverse func(tlsConn *tls.Conn, notify *notifyConn, peer string)) {
	l.reverse = reverse
	handler = l.authenticated(handler)

	var wg sync.WaitGroup

	var h3 *http3.Server

	if l.quic != nil {
		h3 = &http3.Server{Handler: handler, EnableDatagrams: true}

		wg.Go(func() {
			for {
				conn, err := l.quic.Accept(ctx)
				if err != nil {
					return
				}

				go func() { _ = h3.ServeQUICConn(conn) }()
			}
		})
	}

	if l.tcp != nil {
		h2 := &http2.Server{ReadIdleTimeout: l.linkTimeout / 3, PingTimeout: l.linkTimeout}

		wg.Go(func() {
			for {
				conn, err := l.tcp.Accept()
				if err != nil {
					return
				}

				go l.serveTCP(ctx, conn, h2, handler)
			}
		})
	}

	<-ctx.Done()

	if h3 != nil {
		_ = h3.Close()
	}

	l.close()

	wg.Wait()
}

func (l *listener) serveTCP(ctx context.Context, conn net.Conn, h2 *http2.Server, handler http.Handler) {
	logger := log.Ctx(ctx)

	notify := &notifyConn{Conn: conn, closed: make(chan struct{})}
	tlsConn := tls.Server(notify, l.identity.ServerTLS(alpnMesh, alpnReverse))

	handshakeCtx, cancel := context.WithTimeout(ctx, l.linkTimeout)
	err := tlsConn.HandshakeContext(handshakeCtx)
	cancel()

	if err != nil {
		// Anyone can reach the port: a failed handshake is not worth more than a debug line.
		logger.Debug().Err(err).Stringer("remote", conn.RemoteAddr()).Msg("Refused a link to the mesh")
		_ = conn.Close()

		return
	}

	state := tlsConn.ConnectionState()

	if state.NegotiatedProtocol != alpnMesh && (state.NegotiatedProtocol != alpnReverse || l.reverse == nil) {
		_ = tlsConn.Close()
		return
	}

	if !l.track(tlsConn) {
		_ = tlsConn.Close()
		return
	}

	// A satellite that opens a link for this instance to send requests through: the instance is the client of HTTP/2.
	if state.NegotiatedProtocol == alpnReverse {
		peer, err := l.identity.PeerNode(state)
		if err != nil {
			l.untrack(tlsConn)
			_ = tlsConn.Close()

			return
		}

		context.AfterFunc(ctx, func() { _ = tlsConn.Close() })

		go func() {
			<-notify.closed
			l.untrack(tlsConn)
		}()

		l.reverse(tlsConn, notify, peer)

		return
	}

	defer l.untrack(tlsConn)

	h2.ServeConn(tlsConn, &http2.ServeConnOpts{Context: ctx, Handler: handler})
}

// authenticated gives the handler the instance that sent the request.
// The handshake already refused the ones without a token of the cluster: this only reads the name.
func (l *listener) authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.TLS == nil {
			http.Error(rw, errNotMeshRequest.Error(), http.StatusForbidden)
			return
		}

		peer, err := l.identity.PeerNode(*req.TLS)
		if err != nil {
			http.Error(rw, errNotMeshRequest.Error(), http.StatusForbidden)
			return
		}

		next.ServeHTTP(rw, req.WithContext(context.WithValue(req.Context(), peerKey{}, peer)))
	})
}

// track records a TCP connection to close it with the listener, and reports false when the listener is closed.
func (l *listener) track(conn net.Conn) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.conns == nil {
		return false
	}

	l.conns[conn] = struct{}{}

	return true
}

func (l *listener) untrack(conn net.Conn) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.conns, conn)
}

func (l *listener) close() {
	if l.tcp != nil {
		_ = l.tcp.Close()
	}

	if l.quic != nil {
		_ = l.quic.Close()
	}

	// The links this instance opened over QUIC end with the socket they share.
	if l.transport != nil {
		_ = l.transport.Close()
	}

	if l.udp != nil {
		_ = l.udp.Close()
	}

	l.mu.Lock()
	conns := l.conns
	l.conns = nil
	l.mu.Unlock()

	for conn := range conns {
		_ = conn.Close()
	}
}
