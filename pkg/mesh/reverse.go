package mesh

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/http2"
)

const (
	// alpnReverse is a link over TCP that a satellite opens to an instance, for the instance to send requests through it:
	// the satellite is the client of TLS and the server of HTTP/2.
	alpnReverse = "traefix-mesh-reverse/1"

	// TransportTCPReverse is a link over TCP that the other end opened, with the roles of HTTP/2 the other way around.
	TransportTCPReverse = "tcp-reverse"

	// reverseGrace is how long a satellite waits for an instance to reach it back over QUIC, before it opens a link
	// over TCP for the instance to use.
	defaultReverseGrace = 3 * time.Second
)

// errAwaitingReverse is the state of the link to a satellite that the instance cannot reach back over QUIC:
// it waits for the satellite to open a link over TCP.
var errAwaitingReverse = errors.New("waiting for the satellite to open a link back over TCP")

// acceptReverse makes a session of a link over TCP that a satellite opened, and offers it to the link to the satellite.
func (m *Mesh) acceptReverse(tlsConn *tls.Conn, notify *notifyConn, peer string) {
	transport := &http2.Transport{ReadIdleTimeout: m.linkTimeout / 3, PingTimeout: m.linkTimeout}

	h2, err := transport.NewClientConn(tlsConn)
	if err != nil {
		_ = tlsConn.Close()
		return
	}

	s := &session{
		transport: TransportTCPReverse,
		peer:      peer,
		remote:    notify.RemoteAddr(),
		h2:        h2,
		conn:      tlsConn,
		done:      notify.closed,
	}

	m.mu.Lock()
	entry, known := m.byNode[peer]
	m.mu.Unlock()

	// A satellite introduces itself through its own link first: the link back of an unknown instance is not used.
	if !known || !entry.satellite {
		m.logger.Debug().Str("peer", peer).Msg("Refused a link back over TCP from an instance that is not a known satellite")
		s.Close()

		return
	}

	entry.link.offer(s)
}

// offer hands the link to a satellite a session that the satellite opened over TCP.
// A link back over QUIC is better: the session is refused while one is up. A newer session replaces an older one,
// which the satellite only opens again when it lost the older one.
func (l *link) offer(s *session) {
	l.mu.Lock()

	current := l.session
	if current != nil && current.transport == TransportQUIC && l.status.Up {
		l.mu.Unlock()
		s.Close()

		return
	}

	previous := l.offered
	l.offered = s
	l.mu.Unlock()

	if previous != nil {
		previous.Close()
	}

	if current != nil && current.transport == TransportTCPReverse {
		current.Close()
	}

	l.wake()
}

func (l *link) takeOffered() *session {
	l.mu.Lock()
	defer l.mu.Unlock()

	s := l.offered
	l.offered = nil

	if s == nil {
		return nil
	}

	select {
	case <-s.Done():
		return nil
	default:
		return s
	}
}

// connectSatellite opens the link back to a satellite: the session the satellite opened over TCP, when there is one,
// or a link over QUIC to the address its own link came from.
func (l *link) connectSatellite(ctx context.Context) (*session, string, error) {
	if s := l.takeOffered(); s != nil {
		helloCtx, cancel := context.WithTimeout(ctx, l.mesh.linkTimeout)
		theirs, err := l.mesh.sayHello(helloCtx, s)
		cancel()

		if err == nil {
			err = l.mesh.connected(l.entry, s.remote.String(), theirs)
		}

		if err != nil {
			s.Close()
			return nil, "", err
		}

		l.setRole(theirs)

		return s, s.remote.String(), nil
	}

	addresses := l.mesh.addressesOf(l.entry)
	if len(addresses) == 0 {
		return nil, "", errAwaitingReverse
	}

	var failure error

	for _, address := range addresses {
		l.tried(address)

		s, theirs, err := l.open(ctx, address, TransportQUIC, l.mesh.quicDialTimeout)
		if err == nil {
			if err = l.mesh.connected(l.entry, address, theirs); err != nil {
				s.Close()
			}
		}

		if err == nil {
			l.setRole(theirs)

			return s, address, nil
		}

		if ctx.Err() != nil {
			return nil, "", err
		}

		failure = fmt.Errorf("%s: %w", address, err)
	}

	return nil, "", fmt.Errorf("%w (QUIC: %w)", errAwaitingReverse, failure)
}

// reverseLoop keeps a link over TCP open to an instance that does not reach this satellite back over QUIC,
// for as long as the link of the satellite to it is up.
func (l *link) reverseLoop(ctx context.Context, address, nodeID string) {
	transport := l.mesh.conf.Transport

	// With QUIC, the instance reaches the satellite back by itself, right after the hello.
	if transport != TransportTCP {
		select {
		case <-time.After(l.mesh.reverseGrace):
		case <-ctx.Done():
			return
		}
	}

	wait := l.mesh.redialFirst
	warned := false

	for ctx.Err() == nil {
		reachedBack := transport != TransportTCP && l.mesh.reachedBackOverQUIC(nodeID)

		switch {
		case reachedBack:
			wait = l.mesh.redialFirst

		case transport == TransportQUIC:
			if !warned {
				warned = true
				l.logger.Warn().Str("peer", nodeID).Msg("The instance does not reach this satellite back over QUIC, and the transport of the mesh is quic: it cannot reach it over TCP")
			}

		default:
			if transport != TransportTCP && !warned {
				warned = true
				l.logger.Info().Str("peer", nodeID).Msg("The instance does not reach this satellite back over QUIC: the satellite opens a link back over TCP")
			}

			started := l.mesh.now()

			if err := l.mesh.serveReverse(ctx, address, nodeID); err != nil && ctx.Err() == nil {
				l.logger.Debug().Err(err).Str("peer", nodeID).Str("address", address).Msg("The link back over TCP to an instance of the mesh ended")
			}

			// A link that lasted is not a reason to wait long before the next one.
			if l.mesh.now().Sub(started) > l.mesh.linkTimeout {
				wait = l.mesh.redialFirst
			}
		}

		select {
		case <-time.After(min(wait, l.mesh.linkTimeout/3)):
			wait = min(2*wait, l.mesh.redialLongest)
		case <-ctx.Done():
		}
	}
}

// serveReverse opens a link over TCP to an instance, and serves the requests of the instance in it until it ends.
func (m *Mesh) serveReverse(ctx context.Context, address, nodeID string) error {
	dialer := net.Dialer{Timeout: m.linkTimeout}

	raw, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}

	tlsConn := tls.Client(raw, m.identity.ClientTLS(alpnReverse))

	handshakeCtx, cancel := context.WithTimeout(ctx, m.linkTimeout)
	err = tlsConn.HandshakeContext(handshakeCtx)
	cancel()

	if err != nil {
		_ = raw.Close()
		return err
	}

	state := tlsConn.ConnectionState()

	if state.NegotiatedProtocol != alpnReverse {
		_ = tlsConn.Close()
		return fmt.Errorf("the instance does not take links back over TCP (%q)", state.NegotiatedProtocol)
	}

	peer, err := m.identity.PeerNode(state)
	if err != nil || peer != nodeID {
		_ = tlsConn.Close()
		return fmt.Errorf("another instance answers at the address: %q", peer)
	}

	stop := context.AfterFunc(ctx, func() { _ = tlsConn.Close() })
	defer stop()

	server := &http2.Server{ReadIdleTimeout: m.linkTimeout / 3, PingTimeout: m.linkTimeout}
	server.ServeConn(tlsConn, &http2.ServeConnOpts{Context: ctx, Handler: m.listener.authenticated(m)})

	return errors.New("the link back over TCP was closed")
}

// markQUIC records that an instance sent a request over QUIC: on a satellite, that the instance reaches it back.
func (m *Mesh) markQUIC(nodeID string) {
	now := m.now()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.quicSeen[nodeID] = now
}

// reachedBackOverQUIC tells whether an instance reached this satellite back over QUIC lately.
func (m *Mesh) reachedBackOverQUIC(nodeID string) bool {
	now := m.now()

	m.mu.Lock()
	defer m.mu.Unlock()

	seen, ok := m.quicSeen[nodeID]

	return ok && now.Sub(seen) <= m.linkTimeout
}
