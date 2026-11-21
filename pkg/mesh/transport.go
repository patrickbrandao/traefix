package mesh

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"golang.org/x/net/http2"
)

const (
	// alpnMesh is the protocol the instances negotiate on a link, over QUIC and over TLS on TCP:
	// HTTP/3 and HTTP/2 under a name of their own, so that nothing else answers on the port of the mesh.
	alpnMesh = "traefix-mesh/1"

	// controlHost is the authority of the requests the instances send to each other about the link itself.
	controlHost = "mesh.traefix.internal"

	// TransportQUIC is a link over QUIC, with HTTP/3.
	TransportQUIC = cluster.MeshTransportQUIC
	// TransportTCP is a link over TLS on TCP, with HTTP/2.
	TransportTCP = cluster.MeshTransportTCP
)

// session is a connection this instance opened to another one, once the other one proved it holds a token of the cluster.
// The instance that opened it sends the requests.
type session struct {
	transport string
	// peer is the identifier of the other instance, from its certificate.
	peer   string
	remote net.Addr

	h3    *http3.ClientConn
	qconn *quic.Conn
	// closeUDP closes the socket of the session, when it has one of its own.
	closeUDP func()

	h2   *http2.ClientConn
	conn net.Conn
	done chan struct{}

	closeOnce sync.Once
}

func (s *session) RoundTrip(req *http.Request) (*http.Response, error) {
	if s.h3 != nil {
		return s.h3.RoundTrip(req)
	}

	return s.h2.RoundTrip(req)
}

// Done is closed when the connection is, by either side.
func (s *session) Done() <-chan struct{} {
	if s.qconn != nil {
		return s.qconn.Context().Done()
	}

	return s.done
}

func (s *session) Close() {
	s.closeOnce.Do(func() {
		if s.qconn != nil {
			_ = s.qconn.CloseWithError(quic.ApplicationErrorCode(http3.ErrCodeNoError), "")
			s.closeUDP()

			return
		}

		_ = s.h2.Close()
		_ = s.conn.Close()
	})
}

// quicConfig is the QUIC configuration of both ends of a link.
func quicConfig(linkTimeout time.Duration) *quic.Config {
	return &quic.Config{
		// A link without any packet for the link timeout is lost, and keep-alive packets tell a quiet link from a lost one.
		MaxIdleTimeout:  linkTimeout,
		KeepAlivePeriod: linkTimeout / 3,
		// The datagrams carry the UDP packets.
		EnableDatagrams: true,
	}
}

// dialQUIC opens a link over QUIC, from a socket of its own.
func dialQUIC(ctx context.Context, identity *Identity, address string, linkTimeout time.Duration) (*session, error) {
	return dialQUICFrom(ctx, nil, identity, address, linkTimeout)
}

// dialQUICFrom opens a link over QUIC from the socket of a transport, or from a socket of its own without one.
// The socket of the listener of the mesh is the one the links leave from: their source port is the port of the mesh,
// which is what lets an instance reach back a satellite through the mapping its NAT opened.
func dialQUICFrom(ctx context.Context, transport *quic.Transport, identity *Identity, address string, linkTimeout time.Duration) (*session, error) {
	remote, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	// A socket of its own closes with the session; a shared one stays with the listener.
	var own *net.UDPConn

	if transport == nil {
		own, err = net.ListenUDP("udp", nil)
		if err != nil {
			return nil, err
		}

		transport = &quic.Transport{Conn: own}
	}

	closeOwn := func() {
		if own != nil {
			_ = transport.Close()
			_ = own.Close()
		}
	}

	qconn, err := transport.Dial(ctx, remote, identity.ClientTLS(alpnMesh), quicConfig(linkTimeout))
	if err != nil {
		closeOwn()
		return nil, err
	}

	peer, err := identity.PeerNode(qconn.ConnectionState().TLS)
	if err != nil {
		_ = qconn.CloseWithError(quic.ApplicationErrorCode(http3.ErrCodeNoError), "")
		closeOwn()

		return nil, err
	}

	h3 := &http3.Transport{EnableDatagrams: true}

	return &session{
		transport: TransportQUIC,
		peer:      peer,
		remote:    remote,
		h3:        h3.NewClientConn(qconn),
		qconn:     qconn,
		closeUDP:  closeOwn,
	}, nil
}

// dialTCP opens a link over TLS on TCP.
func dialTCP(ctx context.Context, identity *Identity, address string, linkTimeout time.Duration) (*session, error) {
	dialer := net.Dialer{Timeout: linkTimeout}

	raw, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}

	conn := &notifyConn{Conn: raw, closed: make(chan struct{})}

	tlsConn := tls.Client(conn, identity.ClientTLS(alpnMesh))

	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}

	state := tlsConn.ConnectionState()

	if state.NegotiatedProtocol != alpnMesh {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("the other end does not speak the mesh protocol (%q)", state.NegotiatedProtocol)
	}

	peer, err := identity.PeerNode(state)
	if err != nil {
		_ = tlsConn.Close()
		return nil, err
	}

	// TCP only tells that the other end is gone when something is sent: the pings do.
	transport := &http2.Transport{ReadIdleTimeout: linkTimeout / 3, PingTimeout: linkTimeout}

	h2, err := transport.NewClientConn(tlsConn)
	if err != nil {
		_ = tlsConn.Close()
		return nil, err
	}

	return &session{
		transport: TransportTCP,
		peer:      peer,
		remote:    raw.RemoteAddr(),
		h2:        h2,
		conn:      tlsConn,
		done:      conn.closed,
	}, nil
}

// notifyConn tells when a connection is closed: the HTTP/2 client closes the one it is given when it gives up on it.
type notifyConn struct {
	net.Conn

	closed chan struct{}
	once   sync.Once
}

func (c *notifyConn) Close() error {
	c.once.Do(func() { close(c.closed) })

	return c.Conn.Close()
}

var errNotMeshRequest = errors.New("not a request of an instance of the mesh")
