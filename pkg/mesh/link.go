package mesh

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// LinkStatus is the state of the link this instance opens to another one.
type LinkStatus struct {
	// NodeID is empty until the instance was reached once.
	NodeID string
	// Address is the address the link is open at, or the last one that was tried.
	Address string
	// Addresses are the addresses known for the instance, in the order they are tried.
	Addresses []string
	// Sources are where the instance was learned from.
	Sources []string
	// Role is the one the instance told when the link opened.
	Role      string
	Up        bool
	Transport string
	// Since is when the link went up or down.
	Since time.Time
	RTT   time.Duration
	// Error is why the link is down.
	Error       string
	LastContact time.Time
}

// link keeps a session open to an instance: it dials its addresses, introduces itself, watches the session
// and dials again when it is lost.
type link struct {
	mesh   *Mesh
	entry  *peerEntry
	logger zerolog.Logger

	// wakeCh ends the wait before the next dial, when there is something new to try.
	wakeCh chan struct{}

	mu      sync.Mutex
	status  LinkStatus
	session *session
	// role is the one the other instance told when the session opened.
	role string
}

func newLink(m *Mesh, entry *peerEntry) *link {
	return &link{
		mesh:   m,
		entry:  entry,
		logger: m.logger,
		wakeCh: make(chan struct{}, 1),
		status: LinkStatus{NodeID: entry.nodeID, Since: m.now()},
	}
}

func (l *link) wake() {
	select {
	case l.wakeCh <- struct{}{}:
	default:
	}
}

func (l *link) run(ctx context.Context) {
	wait := l.mesh.redialFirst

	for ctx.Err() == nil {
		s, address, err := l.connect(ctx)

		switch {
		case errors.Is(err, errSatellitePeer):
			// The entry is gone, and the address is not dialed again: a satellite was listed as a peer.
			l.logger.Warn().Err(err).Str("address", l.snapshot().Address).Msg("A satellite answers at an address of the peers of the mesh: it is not linked to")
			l.mesh.countPeers()

			return

		case errors.Is(err, errSelf), errors.Is(err, errMerged):
			// The entry is gone: there is nothing left to link to under it.
			l.logger.Debug().Err(err).Msg("The address is not linked to")
			l.mesh.countPeers()

			return

		case err != nil:
			l.setDown(ctx, err)

			// A satellite is reached back for as long as it links to this instance, and not after.
			if l.entry.satellite && !l.mesh.satelliteAlive(l.entry.nodeID) {
				l.mesh.forgetSatellite(l.entry)
				return
			}

			select {
			case <-time.After(wait):
				wait = min(wait*2, l.mesh.redialLongest)
			case <-l.wakeCh:
				wait = l.mesh.redialFirst
			case <-ctx.Done():
			}

			continue
		}

		wait = l.mesh.redialFirst

		l.setUp(s, address)

		// The routes of the instance are followed for as long as the link is up, and dropped with it.
		// A satellite only announces its routers: it imports none.
		followCtx, stopFollowing := context.WithCancel(ctx)
		followed := make(chan struct{})

		go func() {
			defer close(followed)

			if !l.mesh.conf.Satellite {
				l.followRoutes(followCtx, s.peer)
			}
		}()

		err = l.watch(ctx, address)

		stopFollowing()
		<-followed

		l.close()
		l.mesh.setImported(s.peer, nil)
		l.setDown(ctx, err)
	}
}

// followRoutes keeps the table of the routers of the other instance up to date, until the context is done.
func (l *link) followRoutes(ctx context.Context, nodeID string) {
	for ctx.Err() == nil {
		s := l.current()
		if s == nil {
			return
		}

		table, err := l.mesh.askRoutes(ctx, s, l.mesh.importedFrom(nodeID))

		switch {
		case ctx.Err() != nil:
			return

		case err != nil:
			// The pings are what tells a lost link: this only waits for the session that follows, if any.
			l.logger.Debug().Err(err).Str("peer", nodeID).Msg("Unable to ask an instance of the mesh for its routes")

			select {
			case <-time.After(l.mesh.redialFirst):
			case <-ctx.Done():
			}

		case table != nil:
			l.mesh.setImported(nodeID, table)
		}
	}
}

// connect opens a session at the first address of the instance that answers, and introduces this instance.
func (l *link) connect(ctx context.Context) (*session, string, error) {
	addresses := l.mesh.addressesOf(l.entry)
	if len(addresses) == 0 {
		return nil, "", errors.New("no address is known for the instance")
	}

	var failures []string

	for _, address := range addresses {
		l.tried(address)

		s, theirs, err := l.dial(ctx, address)
		if err == nil {
			if err = l.mesh.connected(l.entry, address, theirs); err != nil {
				s.Close()
			}
		}

		if err == nil {
			l.mu.Lock()
			l.role = roleOf(theirs)
			l.mu.Unlock()

			l.mesh.learnPeers(theirs.Peers)
			l.mesh.compareConfiguration(theirs)

			return s, address, nil
		}

		if errors.Is(err, errSelf) || errors.Is(err, errMerged) || errors.Is(err, errSatellitePeer) || ctx.Err() != nil {
			return nil, "", err
		}

		failures = append(failures, address+": "+err.Error())
	}

	return nil, "", errors.New(strings.Join(failures, "; "))
}

// dial opens a session at an address with the transport the configuration asks for.
func (l *link) dial(ctx context.Context, address string) (*session, hello, error) {
	// A satellite is only reached back over QUIC: nothing reaches it over TCP.
	if l.entry.satellite {
		return l.open(ctx, address, TransportQUIC, l.mesh.linkTimeout)
	}

	switch l.mesh.conf.Transport {
	case cluster.MeshTransportQUIC:
		return l.open(ctx, address, TransportQUIC, l.mesh.linkTimeout)

	case cluster.MeshTransportTCP:
		return l.open(ctx, address, TransportTCP, l.mesh.linkTimeout)

	default:
		s, theirs, err := l.open(ctx, address, TransportQUIC, l.mesh.quicDialTimeout)
		if err == nil || ctx.Err() != nil {
			return s, theirs, err
		}

		// QUIC does not get through, e.g. a network that drops UDP: TCP is slower under loss, but it gets there.
		// An instance that refuses the token over QUIC refuses it over TCP too, and that error is the one reported.
		s, theirs, tcpErr := l.open(ctx, address, TransportTCP, l.mesh.linkTimeout)
		if tcpErr != nil {
			return nil, hello{}, fmt.Errorf("%w (QUIC: %w)", tcpErr, err)
		}

		return s, theirs, nil
	}
}

func (l *link) open(ctx context.Context, address, transport string, timeout time.Duration) (*session, hello, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dial := func(ctx context.Context, identity *Identity, address string, linkTimeout time.Duration) (*session, error) {
		return dialQUICFrom(ctx, l.mesh.listener.transport, identity, address, linkTimeout)
	}
	if transport == TransportTCP {
		dial = dialTCP
	}

	s, err := dial(ctx, l.mesh.identity, address, l.mesh.linkTimeout)
	if err != nil {
		return nil, hello{}, err
	}

	theirs, err := l.mesh.sayHello(ctx, s)
	if err != nil {
		s.Close()
		return nil, hello{}, err
	}

	return s, theirs, nil
}

// watch pings the other end until the session is lost or the context is done, and asks it now and then for the instances
// it reaches. On a link over TCP that QUIC could serve, it also tries QUIC again now and then.
func (l *link) watch(ctx context.Context, address string) error {
	pings := time.NewTicker(l.mesh.linkTimeout / 3)
	defer pings.Stop()

	exchange := time.NewTicker(l.mesh.exchangeInterval)
	defer exchange.Stop()

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

			l.setRTT(s.peer, rtt)

		case <-exchange.C:
			peers, err := l.mesh.askPeers(ctx, s)
			if err != nil {
				// The pings are what tells a lost link.
				l.logger.Debug().Err(err).Str("peer", s.peer).Msg("Unable to ask an instance of the mesh for the instances it reaches")
				continue
			}

			l.mesh.learnPeers(peers)

		case <-upgrade:
			if s.transport != TransportTCP {
				continue
			}

			better, theirs, err := l.open(ctx, address, TransportQUIC, l.mesh.quicDialTimeout)
			if err != nil {
				continue
			}

			if theirs.NodeID != s.peer {
				better.Close()
				continue
			}

			l.setUp(better, address)
			s.Close()

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

func (l *link) tried(address string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.status.Up {
		l.status.Address = address
	}
}

func (l *link) setUp(s *session, address string) {
	l.mu.Lock()
	previous := l.status
	l.session = s
	l.status = LinkStatus{NodeID: s.peer, Address: address, Role: l.role, Up: true, Transport: s.transport, Since: l.mesh.now()}
	current := l.status
	l.mu.Unlock()

	l.logger.Info().Str("peer", s.peer).Str("address", address).Str("transport", s.transport).Msg("Linked to an instance of the mesh")

	l.mesh.linkChanged(previous, current)
}

func (l *link) setRTT(nodeID string, rtt time.Duration) {
	l.mu.Lock()
	l.status.RTT = rtt
	l.mu.Unlock()

	l.mesh.contact(nodeID)
	l.mesh.setLinkRTT(nodeID, rtt)
}

func (l *link) setDown(ctx context.Context, cause error) {
	if ctx.Err() != nil {
		// The instance is stopping, or the link is given up.
		return
	}

	l.mu.Lock()
	previous := l.status
	l.status.Up = false
	l.status.Transport = ""
	l.status.RTT = 0
	l.status.Error = cause.Error()
	if previous.Up {
		l.status.Since = l.mesh.now()
	}
	current := l.status
	l.mu.Unlock()

	logger := l.logger.With().Str("peer", current.NodeID).Str("address", current.Address).Logger()

	switch {
	case previous.Up:
		logger.Warn().Err(cause).Msg("Lost the link to an instance of the mesh")
	case previous.Error != current.Error:
		// The attempts repeat for as long as the other instance is away: only a new reason is worth a line.
		logger.Warn().Err(cause).Msg("Unable to link to an instance of the mesh")
	default:
		logger.Debug().Err(cause).Msg("Unable to link to an instance of the mesh")
	}

	l.mesh.linkChanged(previous, current)
}
