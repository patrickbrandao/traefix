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
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
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
	// satelliteAddresses are the addresses where a satellite answered: no link is opened to them.
	satelliteAddresses map[string]struct{}
	// refused are the instances of the cluster with another token, which were warned about.
	refused map[string]struct{}
	// dirty tells that the file of learned instances is out of date.
	dirty bool

	// The entry points of this instance: the ones a router can listen on (all but the UDP ones), the UDP ones,
	// and the ones of a router that names none.
	entryPoints        []string
	udpEntryPoints     []string
	defaultEntryPoints []string

	routesPollTimeout time.Duration
	publishDelay      time.Duration

	// exported are the routers this instance announces. Its generation changes with it, and exportChanged is closed then.
	exported         []Route
	exportGeneration uint64
	exportChanged    chan struct{}
	// localRouters are the names of all the routers of the local providers.
	localRouters map[string]struct{}
	// imported are the tables of the other instances, by identifier.
	imported map[string]*importedTable
	// importReport is what became of the routes of the other instances.
	importReport []ImportedRoute
	warned       map[string]struct{}

	configurationChan chan<- dynamic.Message
	publishCh         chan struct{}

	// localHTTP and localTCP are the routers this instance announces, as their providers wrote them.
	localHTTP map[string]*dynamic.Router
	localTCP  map[string]*dynamic.TCPRouter
	localUDP  map[string]*dynamic.UDPRouter
	// localUDPEntryPoints are the UDP entry points that have a router of the local providers, announced or not:
	// an entry point has one UDP router only.
	localUDPEntryPoints map[string]struct{}

	// entryPointSummaries are what this instance tells the others of its entry points.
	entryPointSummaries map[string]EntryPointSummary
	// mismatches are the differences of configuration with the other instances, by identifier.
	mismatches map[string][]string

	// quicSeen is when each instance last sent a request over QUIC: on a satellite, that it reaches the satellite back.
	quicSeen     map[string]time.Time
	reverseGrace time.Duration

	// certificateSource gives the certificates of the resolvers of this instance, for the satellites.
	certificateSource CertificateSource
	// challengeAnswer gives the answers of the HTTP-01 challenges this instance presents, for the other instances.
	challengeAnswer ChallengeAnswer
	// received are the certificates the instances gave this satellite, by identifier.
	received             map[string][]Certificate
	certificatesInterval time.Duration
	// routing serves these routers by name, for what the other instances forward.
	routing atomic.Pointer[routing]
	// tables are the routes as the requests need them.
	tables atomic.Pointer[forwardTables]
	// turn counts the requests, for the round robin.
	turn atomic.Uint64
	// quarantines are the instances that failed for a route, by route and instance.
	quarantines map[string]*quarantine

	healthInterval time.Duration
	// localBase are the routers to announce, as the providers give them. The announced table adds their state.
	localBase []Route
	// localDown are the announced routers that have no server to answer.
	localDown atomic.Pointer[map[string]struct{}]

	runCtx  context.Context //nolint:containedctx // The links learned while the mesh runs are started under it.
	stopped bool
	linksWG sync.WaitGroup
}

// InboundStatus is an instance that opened a link to this one.
type InboundStatus struct {
	NodeID    string
	Role      string
	Version   string
	Remote    string
	Advertise []string
	LastSeen  time.Time
}

// Status is what an instance knows of its links.
type Status struct {
	NodeID    string
	Role      string
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

	var (
		advertise []string
		guessed   bool
	)

	// A satellite is never reached at an address: it has none to tell.
	if !conf.Mesh.Satellite {
		advertise, guessed, err = advertisedAddresses(conf, outboundIP)
		if err != nil {
			return nil, err
		}
	}

	switch {
	case conf.Mesh.Satellite:
		logger.Info().Strs("peers", conf.Mesh.Peers).Strs("dns", conf.Mesh.DNS).
			Msg("This instance is a satellite of the mesh: it dials the other instances, and gets its traffic through the links it opens")

	case guessed:
		logger.Info().Strs("advertise", advertise).
			Msg("The cluster mesh advertise option is not set: the other instances are told to reach this instance at these addresses")
	}

	if !member.HasServer() && len(conf.Mesh.Peers) == 0 && len(conf.Mesh.DNS) == 0 {
		logger.Warn().Msg("The mesh has no Redis server, peers or dns to find the other instances: this instance waits for them to link to it")
	}

	if len(advertise) > 0 {
		member.SetAttribute(attributeAddresses, strings.Join(advertise, ","))
	}
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
		conf:                 conf.Mesh,
		member:               member,
		identity:             identity,
		advertise:            advertise,
		version:              version.Version,
		logger:               logger,
		linkTimeout:          linkTimeout,
		quicDialTimeout:      min(defaultQUICDialTimeout, linkTimeout),
		quicRetryInterval:    defaultQUICRetryInterval,
		redialFirst:          defaultRedialFirst,
		redialLongest:        defaultRedialLongest,
		discoverInterval:     defaultDiscoverInterval,
		dnsInterval:          defaultDNSInterval,
		exchangeInterval:     defaultExchangeInterval,
		now:                  time.Now,
		listener:             l,
		inbound:              map[string]InboundStatus{},
		entries:              map[*peerEntry]struct{}{},
		byNode:               map[string]*peerEntry{},
		byAddress:            map[string]*peerEntry{},
		selfAddresses:        map[string]struct{}{},
		satelliteAddresses:   map[string]struct{}{},
		refused:              map[string]struct{}{},
		routesPollTimeout:    routesPollTimeout,
		publishDelay:         publishDelay,
		exported:             []Route{},
		exportChanged:        make(chan struct{}),
		localRouters:         map[string]struct{}{},
		imported:             map[string]*importedTable{},
		warned:               map[string]struct{}{},
		publishCh:            make(chan struct{}, 1),
		localHTTP:            map[string]*dynamic.Router{},
		localTCP:             map[string]*dynamic.TCPRouter{},
		localUDP:             map[string]*dynamic.UDPRouter{},
		mismatches:           map[string][]string{},
		quicSeen:             map[string]time.Time{},
		reverseGrace:         defaultReverseGrace,
		received:             map[string][]Certificate{},
		certificatesInterval: defaultCertificatesInterval,
		quarantines:          map[string]*quarantine{},
		healthInterval:       defaultHealthInterval,
	}

	m.tables.Store(newForwardTables())

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

	wg.Go(func() { m.listener.serve(ctx, m, m.acceptReverse) })
	wg.Go(func() { m.discover(ctx) })
	wg.Go(func() { m.publishLoop(ctx) })
	wg.Go(func() { m.healthLoop(ctx) })

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
	status := Status{NodeID: m.member.NodeID(), Role: m.role(), Advertise: m.advertise}

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

// SetEntryPoints tells the mesh the entry points of this instance: the ones a router can listen on, the UDP ones,
// and the ones of a router that names none. It must be called before Run.
func (m *Mesh) SetEntryPoints(entryPoints, udpEntryPoints, defaultEntryPoints []string) {
	m.entryPoints = entryPoints
	m.udpEntryPoints = udpEntryPoints
	m.defaultEntryPoints = defaultEntryPoints
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
		Role:      roleOf(theirs),
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
func (m *Mesh) linkChanged(previous, current LinkStatus, satellite bool) {
	// A link has no name in the metrics before the instance behind its address is known:
	// until then it only counts among the links that are down.
	if current.NodeID != "" {
		for _, transport := range linkTransports(satellite) {
			m.setLinkUp(current.NodeID, transport, current.Up && current.Transport == transport)
		}

		m.setLinkRTT(current.NodeID, current.RTT)
	}

	// Another instance answers at the address by now.
	if previous.NodeID != "" && previous.NodeID != current.NodeID {
		for _, transport := range linkTransports(satellite) {
			m.setLinkUp(previous.NodeID, transport, false)
		}

		m.setLinkRTT(previous.NodeID, 0)
	}

	m.countPeers()
}

// linkTransports returns the transports a link can be up with: only the link back to a satellite goes over a
// connection the other end opened.
func linkTransports(satellite bool) []string {
	if satellite {
		return []string{TransportQUIC, TransportTCP, TransportTCPReverse}
	}

	return []string{TransportQUIC, TransportTCP}
}

// countPeers follows the count of links in the metrics. The links back to the satellites count apart: a central can
// have hundreds of them, and they are not instances it looks for.
func (m *Mesh) countPeers() {
	var peers, satellites [2]int

	m.mu.Lock()
	for entry := range m.entries {
		counts := &peers
		if entry.satellite {
			counts = &satellites
		}

		if entry.link.snapshot().Up {
			counts[0]++
		} else {
			counts[1]++
		}
	}
	m.mu.Unlock()

	m.setPeers(peers[0], peers[1])

	if !m.conf.Satellite {
		m.setSatellites(satellites[0], satellites[1])
	}
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
