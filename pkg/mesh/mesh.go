package mesh

import (
	"context"
	"errors"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/version"
)

const (
	// attributeAddresses is where an instance tells the cluster the addresses the other instances reach it at.
	attributeAddresses = "mesh.addr"
	// attributeAuthority is the fingerprint of the token of an instance: the instances with another one cannot link to it.
	attributeAuthority = "mesh.authority"
)

// Mesh is the part an instance takes in the mesh of its cluster.
type Mesh struct {
	conf      *cluster.Mesh
	member    *cluster.Cluster
	identity  *Identity
	advertise []string
	version   string
	logger    zerolog.Logger

	linkTimeout       time.Duration
	quicDialTimeout   time.Duration
	quicRetryInterval time.Duration
	redialFirst       time.Duration
	redialLongest     time.Duration

	listener *listener
	links    []*link

	mu sync.Mutex
	// inbound are the instances that opened a link to this one, by identifier.
	inbound map[string]InboundStatus
}

// InboundStatus is an instance that opened a link to this one.
type InboundStatus struct {
	NodeID    string
	Version   string
	Remote    string
	Advertise []string
	LastSeen  time.Time
}

// Status is what an instance knows of its links.
type Status struct {
	NodeID    string
	Advertise []string
	// Links are the links this instance opens, in the order of their addresses.
	Links []LinkStatus
	// Inbound are the instances that opened a link to this one, in the order of their identifiers.
	Inbound []InboundStatus
}

// New prepares the mesh of an instance: its identity, the addresses it tells the other instances,
// and the address it listens on, which is bound at once so that a busy port stops the startup.
func New(conf *cluster.Configuration, member *cluster.Cluster) (*Mesh, error) {
	if conf == nil || conf.Mesh == nil {
		return nil, errors.New("the mesh needs the cluster mesh option")
	}

	token, additional, err := conf.Tokens()
	if err != nil {
		return nil, err
	}

	identity, err := NewIdentity(member.Prefix(), member.NodeID(), token, additional)
	if err != nil {
		return nil, err
	}

	logger := log.With().Str("cluster", member.Prefix()).Str("nodeID", member.NodeID()).Logger()

	advertise, guessed, err := advertisedAddresses(conf, outboundIP)
	if err != nil {
		return nil, err
	}

	if guessed {
		logger.Info().Strs("advertise", advertise).
			Msg("The cluster mesh advertise option is not set: the other instances are told to reach this instance at these addresses")
	}

	if !member.HasServer() && len(conf.Mesh.Peers) == 0 && len(conf.Mesh.DNS) == 0 {
		logger.Warn().Msg("The mesh has no Redis server, peers or dns to find the other instances: this instance waits for them to link to it")
	}

	member.SetAttribute(attributeAddresses, strings.Join(advertise, ","))
	member.SetAttribute(attributeAuthority, identity.Fingerprint())

	// The fingerprint lets an operator check that two instances hold the same token, without showing the token.
	logger.Info().Str("authority", identity.Fingerprint()).Int("additionalTokens", len(additional)).
		Msg("The mesh identity of this instance is ready")

	linkTimeout := time.Duration(conf.Mesh.LinkTimeout)

	l, err := newListener(conf.Mesh.Address, conf.Mesh.Transport, identity, linkTimeout)
	if err != nil {
		return nil, err
	}

	m := &Mesh{
		conf:              conf.Mesh,
		member:            member,
		identity:          identity,
		advertise:         advertise,
		version:           version.Version,
		logger:            logger,
		linkTimeout:       linkTimeout,
		quicDialTimeout:   min(defaultQUICDialTimeout, linkTimeout),
		quicRetryInterval: defaultQUICRetryInterval,
		redialFirst:       defaultRedialFirst,
		redialLongest:     defaultRedialLongest,
		listener:          l,
		inbound:           map[string]InboundStatus{},
	}

	var seen []string
	for _, address := range conf.Mesh.Peers {
		if !slices.Contains(seen, address) {
			seen = append(seen, address)
			m.links = append(m.links, newLink(m, address))
		}
	}

	return m, nil
}

// Run accepts the links of the other instances and keeps the links to them open, until the context is done.
func (m *Mesh) Run(ctx context.Context) {
	ctx = m.logger.WithContext(ctx)

	var wg sync.WaitGroup

	wg.Go(func() { m.listener.serve(ctx, m) })

	for _, l := range m.links {
		wg.Go(func() { l.run(ctx) })
	}

	wg.Wait()

	for _, l := range m.links {
		l.close()
	}
}

// Status returns the links of this instance.
func (m *Mesh) Status() Status {
	status := Status{NodeID: m.member.NodeID(), Advertise: m.advertise}

	for _, l := range m.links {
		status.Links = append(status.Links, l.snapshot())
	}

	m.mu.Lock()
	for _, inbound := range m.inbound {
		status.Inbound = append(status.Inbound, inbound)
	}
	m.mu.Unlock()

	slices.SortFunc(status.Inbound, func(a, b InboundStatus) int { return strings.Compare(a.NodeID, b.NodeID) })

	return status
}

// Identity returns what this instance shows to the others and checks of them.
func (m *Mesh) Identity() *Identity {
	return m.identity
}

// Advertise returns the addresses the other instances reach this instance at.
func (m *Mesh) Advertise() []string {
	return m.advertise
}

// greeted records the instance that introduced itself through a link it opened.
func (m *Mesh) greeted(theirs hello, remote string) {
	m.mu.Lock()
	_, known := m.inbound[theirs.NodeID]
	m.inbound[theirs.NodeID] = InboundStatus{
		NodeID:    theirs.NodeID,
		Version:   theirs.Version,
		Remote:    remote,
		Advertise: theirs.Advertise,
		LastSeen:  time.Now(),
	}
	m.mu.Unlock()

	if !known {
		m.logger.Info().Str("peer", theirs.NodeID).Str("remote", remote).Msg("An instance of the mesh linked to this one")
	}
}

// seen records a sign of life of an instance that opened a link to this one.
func (m *Mesh) seen(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if inbound, ok := m.inbound[nodeID]; ok {
		inbound.LastSeen = time.Now()
		m.inbound[nodeID] = inbound
	}
}

// linkChanged follows a link in the metrics.
func (m *Mesh) linkChanged(previous, current LinkStatus) {
	// A link has no name in the metrics before the instance behind its address is known:
	// until then it only counts among the links that are down.
	if current.NodeID != "" && !current.Self {
		for _, transport := range []string{TransportQUIC, TransportTCP} {
			setLinkUp(current.NodeID, transport, current.Up && current.Transport == transport)
		}

		setLinkRTT(current.NodeID, current.RTT)
	}

	// Another instance answers at the address by now.
	if previous.NodeID != "" && previous.NodeID != current.NodeID && !previous.Self {
		for _, transport := range []string{TransportQUIC, TransportTCP} {
			setLinkUp(previous.NodeID, transport, false)
		}

		setLinkRTT(previous.NodeID, 0)
	}

	up, down := 0, 0
	for _, l := range m.links {
		switch status := l.snapshot(); {
		case status.Self:
		case status.Up:
			up++
		default:
			down++
		}
	}

	setPeers(up, down)
}

func (m *Mesh) linkMeasured(status LinkStatus) {
	setLinkRTT(status.NodeID, status.RTT)
}

// advertisedAddresses returns the addresses to tell the other instances, and whether they had to be guessed.
// The outbound function returns the local address the instance uses to reach a target.
func advertisedAddresses(conf *cluster.Configuration, outbound func(target string) (net.IP, error)) ([]string, bool, error) {
	if len(conf.Mesh.Advertise) > 0 {
		return conf.Mesh.Advertise, false, nil
	}

	host, port, err := net.SplitHostPort(conf.Mesh.Address)
	if err != nil {
		return nil, false, err
	}

	// An address to listen on that names one interface is also where the instance is reached.
	if ip := net.ParseIP(host); host != "" && (ip == nil || !ip.IsUnspecified()) {
		return []string{conf.Mesh.Address}, false, nil
	}

	// Otherwise, the address the instance uses to reach the cluster is the best guess: it is on the right network.
	var targets []string
	if conf.Redis != nil {
		targets = append(targets, conf.Redis.Endpoints...)
	}
	targets = append(targets, conf.Mesh.Peers...)

	for _, target := range targets {
		ip, err := outbound(target)
		if err != nil || ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
			continue
		}

		return []string{net.JoinHostPort(ip.String(), port)}, true, nil
	}

	if ip := firstInterfaceIP(); ip != nil {
		return []string{net.JoinHostPort(ip.String(), port)}, true, nil
	}

	return nil, false, errors.New("unable to find the address of this instance for the mesh: set the cluster mesh advertise option")
}

// outboundIP returns the local address the instance would use to reach a target. Nothing is sent: UDP needs no handshake.
func outboundIP(target string) (net.IP, error) {
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		return nil, err
	}

	conn, err := net.Dial("udp", net.JoinHostPort(host, strconv.Itoa(9)))
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, errors.New("unexpected local address type")
	}

	return addr.IP, nil
}

// firstInterfaceIP returns an address of this host that other hosts can reach, if any.
func firstInterfaceIP() net.IP {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || !ipNet.IP.IsGlobalUnicast() {
			continue
		}

		return ipNet.IP
	}

	return nil
}
