package mesh

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/traefik/traefik/v3/pkg/cluster"
)

const (
	// quicDialTimeout is how long an instance waits for QUIC before it falls back to TCP.
	defaultQUICDialTimeout = 3 * time.Second
	// quicRetryInterval is how often an instance linked over TCP checks whether QUIC gets through by now.
	defaultQUICRetryInterval = 5 * time.Minute

	defaultRedialFirst   = time.Second
	defaultRedialLongest = 30 * time.Second
)

// LinkStatus is the state of the link this instance opens to an address.
type LinkStatus struct {
	Address string
	// NodeID is empty until the instance at the address was reached once.
	NodeID    string
	Up        bool
	Transport string
	// Since is when the link went up or down.
	Since time.Time
	RTT   time.Duration
	// Error is why the link is down.
	Error string
	// Self is true for an address that turned out to be this instance, which is not dialed again.
	Self bool
}

// errSelf stops the link to an address that is this instance: a list of peers shared by all the instances names each of them.
var errSelf = errors.New("the address is the one of this instance")

// link keeps a session open to an address: it dials, introduces itself, watches the session and dials again when it is lost.
type link struct {
	mesh    *Mesh
	address string
	logger  zerolog.Logger

	mu      sync.Mutex
	status  LinkStatus
	session *session
}

func newLink(m *Mesh, address string) *link {
	return &link{
		mesh:    m,
		address: address,
		logger:  m.logger.With().Str("address", address).Logger(),
		status:  LinkStatus{Address: address, Since: time.Now()},
	}
}

func (l *link) run(ctx context.Context) {
	wait := l.mesh.redialFirst

	for ctx.Err() == nil {
		s, err := l.connect(ctx)

		switch {
		case errors.Is(err, errSelf):
			l.setSelf()
			return

		case err != nil:
			l.setDown(ctx, err)

			select {
			case <-time.After(wait):
				wait = min(wait*2, l.mesh.redialLongest)
			case <-ctx.Done():
			}

			continue
		}

		wait = l.mesh.redialFirst

		l.setUp(s, 0)

		err = l.watch(ctx)

		l.close()
		l.setDown(ctx, err)
	}
}

// connect opens a session with the transport the configuration asks for, and introduces this instance.
func (l *link) connect(ctx context.Context) (*session, error) {
	switch l.mesh.conf.Transport {
	case cluster.MeshTransportQUIC:
		return l.open(ctx, TransportQUIC, l.mesh.linkTimeout)

	case cluster.MeshTransportTCP:
		return l.open(ctx, TransportTCP, l.mesh.linkTimeout)

	default:
		s, err := l.open(ctx, TransportQUIC, l.mesh.quicDialTimeout)
		if err == nil || errors.Is(err, errSelf) || ctx.Err() != nil {
			return s, err
		}

		// QUIC does not get through, e.g. a network that drops UDP: TCP is slower under loss, but it gets there.
		// An instance that refuses the token over QUIC refuses it over TCP too, and that error is the one reported.
		s, tcpErr := l.open(ctx, TransportTCP, l.mesh.linkTimeout)
		if tcpErr != nil {
			return nil, errors.Join(tcpErr, err)
		}

		return s, nil
	}
}

func (l *link) open(ctx context.Context, transport string, timeout time.Duration) (*session, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dial := dialQUIC
	if transport == TransportTCP {
		dial = dialTCP
	}

	s, err := dial(ctx, l.mesh.identity, l.address, l.mesh.linkTimeout)
	if err != nil {
		return nil, err
	}

	theirs, err := l.mesh.sayHello(ctx, s)
	if err != nil {
		s.Close()
		return nil, err
	}

	if theirs.NodeID == l.mesh.member.NodeID() {
		s.Close()

		if theirs.Instance == l.mesh.member.Instance() {
			return nil, errSelf
		}

		return nil, errors.New("another instance uses the nodeID of this one: give each instance its own cluster nodeID")
	}

	return s, nil
}

// watch pings the other end until the session is lost or the context is done.
// On a link over TCP that QUIC could serve, it also tries QUIC again now and then.
func (l *link) watch(ctx context.Context) error {
	pings := time.NewTicker(l.mesh.linkTimeout / 3)
	defer pings.Stop()

	var upgrade <-chan time.Time
	if l.mesh.conf.Transport == cluster.MeshTransportAuto {
		ticker := time.NewTicker(l.mesh.quicRetryInterval)
		defer ticker.Stop()

		upgrade = ticker.C
	}

	for {
		s := l.current()

		select {
		case <-pings.C:
			rtt, err := ping(ctx, s, l.mesh.linkTimeout)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}

				return fmt.Errorf("the other instance does not answer: %w", err)
			}

			l.setRTT(rtt)

		case <-upgrade:
			if s.transport != TransportTCP {
				continue
			}

			if better, err := l.open(ctx, TransportQUIC, l.mesh.quicDialTimeout); err == nil {
				l.setUp(better, 0)
				s.Close()
			}

		case <-s.Done():
			if ctx.Err() != nil {
				return ctx.Err()
			}

			return errors.New("the connection was closed")

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (l *link) current() *session {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.session
}

func (l *link) close() {
	l.mu.Lock()
	s := l.session
	l.session = nil
	l.mu.Unlock()

	if s != nil {
		s.Close()
	}
}

func (l *link) snapshot() LinkStatus {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.status
}

func (l *link) setUp(s *session, rtt time.Duration) {
	l.mu.Lock()
	previous := l.status
	l.session = s
	l.status = LinkStatus{Address: l.address, NodeID: s.peer, Up: true, Transport: s.transport, Since: time.Now(), RTT: rtt}
	l.mu.Unlock()

	l.logger.Info().Str("peer", s.peer).Str("transport", s.transport).Msg("Linked to an instance of the mesh")

	l.mesh.linkChanged(previous, l.snapshot())
}

func (l *link) setRTT(rtt time.Duration) {
	l.mu.Lock()
	l.status.RTT = rtt
	status := l.status
	l.mu.Unlock()

	l.mesh.linkMeasured(status)
}

func (l *link) setDown(ctx context.Context, cause error) {
	if ctx.Err() != nil {
		// The instance is stopping.
		return
	}

	l.mu.Lock()
	previous := l.status
	l.status.Up = false
	l.status.RTT = 0
	l.status.Error = cause.Error()
	if previous.Up {
		l.status.Since = time.Now()
	}
	status := l.status
	l.mu.Unlock()

	switch {
	case previous.Up:
		l.logger.Warn().Err(cause).Str("peer", previous.NodeID).Msg("Lost the link to an instance of the mesh")
	case previous.Error != status.Error:
		// The attempts repeat for as long as the other instance is away: only a new reason is worth a line.
		l.logger.Warn().Err(cause).Msg("Unable to link to an instance of the mesh")
	default:
		l.logger.Debug().Err(cause).Msg("Unable to link to an instance of the mesh")
	}

	l.mesh.linkChanged(previous, status)
}

func (l *link) setSelf() {
	l.mu.Lock()
	previous := l.status
	l.status = LinkStatus{Address: l.address, NodeID: l.mesh.member.NodeID(), Self: true, Since: time.Now()}
	status := l.status
	l.mu.Unlock()

	l.logger.Debug().Msg("The address is the one of this instance: it is not linked to")

	l.mesh.linkChanged(previous, status)
}
