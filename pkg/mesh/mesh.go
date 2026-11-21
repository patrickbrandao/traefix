package mesh

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/observability/metrics"
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

	discoverInterval time.Duration
	dnsInterval      time.Duration
	exchangeInterval time.Duration

	// metrics replaces the registry of the process, in the tests with several instances.
	metrics metrics.Registry

	now       func() time.Time
	lookupSRV func(ctx context.Context, name string) ([]*net.SRV, error)
	lookupTXT func(ctx context.Context, name string) ([]string, error)

	listener *listener

	mu sync.Mutex
	// inbound are the instances that opened a link to this one, by identifier.
	inbound map[string]InboundStatus

	// The instances this one links to: all of them, the ones whose identifier is known, and the ones only known by an address.
	entries   map[*peerEntry]struct{}
	byNode    map[string]*peerEntry
	byAddress map[string]*peerEntry
	// selfAddresses are the addresses that turned out to be this instance.
	selfAddresses map[string]struct{}
	// refused are the instances of the cluster with another token, which were warned about.
	refused map[string]struct{}
	// dirty tells that the file of learned instances is out of date.
	dirty bool

	runCtx  context.Context //nolint:containedctx // The links learned while the mesh runs are started under it.
	stopped bool
	linksWG sync.WaitGroup
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
	// Links are the links this instance opens, in the order of the identifiers of the instances, then of their addresses.
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
		discoverInterval:  defaultDiscoverInterval,
		dnsInterval:       defaultDNSInterval,
		exchangeInterval:  defaultExchangeInterval,
		now:               time.Now,
		listener:          l,
		inbound:           map[string]InboundStatus{},
		entries:           map[*peerEntry]struct{}{},
		byNode:            map[string]*peerEntry{},
		byAddress:         map[string]*peerEntry{},
		selfAddresses:     map[string]struct{}{},
		refused:           map[string]struct{}{},
	}

	m.lookupSRV = func(ctx context.Context, name string) ([]*net.SRV, error) {
		// Without a service and a protocol, the name is looked up as it is written.
		_, records, err := net.DefaultResolver.LookupSRV(ctx, "", "", name)
		return records, err
	}
	m.lookupTXT = net.DefaultResolver.LookupTXT

	// The links only start with Run.
	m.loadPeers()
	m.learn(sourcePeers, "", conf.Mesh.Peers)

	return m, nil
}

// Run accepts the links of the other instances, finds them and keeps the links to them open, until the context is done.
func (m *Mesh) Run(ctx context.Context) {
	ctx = m.logger.WithContext(ctx)

	m.mu.Lock()
	m.runCtx = ctx
	for entry := range m.entries {
		m.startLocked(entry)
	}
	m.mu.Unlock()

	// The counts exist from the start, at zero for an instance that knows nobody yet.
	m.countPeers()

	var wg sync.WaitGroup

	wg.Go(func() { m.listener.serve(ctx, m) })
	wg.Go(func() { m.discover(ctx) })

	wg.Wait()

	// No link starts from here on, so that the wait below sees them all.
	m.mu.Lock()
	m.stopped = true
	m.mu.Unlock()

	m.linksWG.Wait()

	m.savePeers(true)
}

// Status returns the links of this instance.
func (m *Mesh) Status() Status {
	status := Status{NodeID: m.member.NodeID(), Advertise: m.advertise}

	m.mu.Lock()
	for entry := range m.entries {
		link := entry.link.snapshot()
		link.NodeID = entry.nodeID
		link.Addresses = slices.Clone(entry.addresses)
		link.Sources = slices.Sorted(maps.Keys(entry.sources))
		link.LastContact = entry.lastContact

		if link.Address == "" && len(entry.addresses) > 0 {
			link.Address = entry.addresses[0]
		}

		status.Links = append(status.Links, link)
	}

	for _, inbound := range m.inbound {
		status.Inbound = append(status.Inbound, inbound)
	}
	m.mu.Unlock()

	slices.SortFunc(status.Links, func(a, b LinkStatus) int {
		return cmp.Or(strings.Compare(a.NodeID, b.NodeID), strings.Compare(a.Address, b.Address))
	})
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
		LastSeen:  m.now(),
	}
	m.mu.Unlock()

	if !known {
		m.logger.Info().Str("peer", theirs.NodeID).Str("remote", remote).Msg("An instance of the mesh linked to this one")
	}
}

// seen records a sign of life of an instance that opened a link to this one.
func (m *Mesh) seen(nodeID string) {
	m.mu.Lock()
	if inbound, ok := m.inbound[nodeID]; ok {
		inbound.LastSeen = m.now()
		m.inbound[nodeID] = inbound
	}
	m.mu.Unlock()

	m.contact(nodeID)
}

// linkChanged follows a link in the metrics.
func (m *Mesh) linkChanged(previous, current LinkStatus) {
	// A link has no name in the metrics before the instance behind its address is known:
	// until then it only counts among the links that are down.
	if current.NodeID != "" {
		for _, transport := range []string{TransportQUIC, TransportTCP} {
			m.setLinkUp(current.NodeID, transport, current.Up && current.Transport == transport)
		}

		m.setLinkRTT(current.NodeID, current.RTT)
	}

	// Another instance answers at the address by now.
	if previous.NodeID != "" && previous.NodeID != current.NodeID {
		for _, transport := range []string{TransportQUIC, TransportTCP} {
			m.setLinkUp(previous.NodeID, transport, false)
		}

		m.setLinkRTT(previous.NodeID, 0)
	}

	m.countPeers()
}

// countPeers follows the count of links in the metrics.
func (m *Mesh) countPeers() {
	up, down := 0, 0

	m.mu.Lock()
	for entry := range m.entries {
		if entry.link.snapshot().Up {
			up++
		} else {
			down++
		}
	}
	m.mu.Unlock()

	m.setPeers(up, down)
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
