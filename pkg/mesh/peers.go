package mesh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// The sources an instance learns the other instances from.
const (
	// sourcePeers is the list of the configuration. What comes from it is never forgotten.
	sourcePeers = "peers"
	// sourceRedis is the announcement of the instances in the Redis server of the cluster.
	sourceRedis = "redis"
	// sourceDNS are the DNS records of the configuration.
	sourceDNS = "dns"
	// sourceExchange is what the other instances tell: who they are when they open a link, and who they reach.
	sourceExchange = "exchange"
	// sourceFile is the file of the instances learned before the last stop.
	sourceFile = "file"
)

const (
	// maxPeerAddresses bounds the addresses kept for an instance, and so the time a dial takes to try them all.
	maxPeerAddresses = 8
	// maxExchangedPeers bounds what an instance takes from the list of another one.
	maxExchangedPeers = 256

	defaultDiscoverInterval = 5 * time.Second
	defaultDNSInterval      = time.Minute
	defaultExchangeInterval = 30 * time.Second
	// peersFileInterval is how often the file is written again when nothing but the dates of the last contacts changed.
	peersFileInterval = 5 * time.Minute

	dnsTimeout = 10 * time.Second
	// dnsTXTPrefix starts the TXT record that lists instances, where SRV records are not an option.
	dnsTXTPrefix = "traefix-mesh="

	peersFileVersion = 1
)

var (
	// errSelf stops the link to an address that is this instance: a list of peers shared by all the instances names each of them.
	errSelf = errors.New("the address is the one of this instance")
	// errMerged stops the link to an address of an instance this one already knows by another address.
	errMerged = errors.New("the instance at the address is already known")

	errSameNodeID = errors.New("another instance uses the nodeID of this one: give each instance its own cluster nodeID")

	// nodeIDRegexp is the one of the cluster configuration: what another instance says is checked like what the operator writes.
	nodeIDRegexp = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,252}$`)
)

// peerInfo is an instance as the instances tell each other, and as the file of learned instances keeps it.
type peerInfo struct {
	NodeID    string   `json:"nodeID"`
	Addresses []string `json:"addresses"`
	// LastContact is only written to the file.
	LastContact *time.Time `json:"lastContact,omitempty"`
}

// peerEntry is an instance this one links to: known by its identifier, or only by an address until it answers there.
type peerEntry struct {
	nodeID string
	// addresses are tried in order: the one that worked last comes first.
	addresses []string
	sources   map[string]struct{}

	learnedAt   time.Time
	lastContact time.Time

	link   *link
	cancel context.CancelFunc
}

type peersFile struct {
	Version int        `json:"version"`
	Peers   []peerInfo `json:"peers"`
}

// learn records an instance that a source tells about, and starts a link to it when it is new.
// Without an identifier, each address is an instance to find out.
func (m *Mesh) learn(source, nodeID string, addresses []string) {
	m.learnAt(source, nodeID, addresses, m.now(), time.Time{})
}

func (m *Mesh) learnAt(source, nodeID string, addresses []string, learnedAt, lastContact time.Time) {
	addresses = cleanAddresses(addresses)

	if len(addresses) == 0 || nodeID == m.member.NodeID() || (nodeID != "" && !nodeIDRegexp.MatchString(nodeID)) {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if nodeID == "" {
		for _, address := range addresses {
			m.learnAddressLocked(source, address, learnedAt)
		}

		return
	}

	entry, known := m.byNode[nodeID]

	if !known {
		// An address that is being dialed without knowing who answers there gets its name: one link is enough.
		for _, address := range addresses {
			if pending, ok := m.byAddress[address]; ok {
				for _, a := range pending.addresses {
					delete(m.byAddress, a)
				}

				pending.nodeID = nodeID
				m.byNode[nodeID] = pending

				entry, known = pending, true

				break
			}
		}
	}

	if !known {
		entry = m.addEntryLocked(nodeID, addresses, source, learnedAt)
		entry.lastContact = lastContact
		m.byNode[nodeID] = entry
		m.dirty = true

		return
	}

	entry.sources[source] = struct{}{}

	// What is told now comes before what was known: an instance that moved is found at its new address first.
	if merged := mergeAddresses(addresses, entry.addresses); !slices.Equal(merged, entry.addresses) {
		entry.addresses = merged
		m.dirty = true

		entry.link.wake()
	}
}

func (m *Mesh) learnAddressLocked(source, address string, learnedAt time.Time) {
	if _, self := m.selfAddresses[address]; self {
		return
	}

	if entry, known := m.byAddress[address]; known {
		entry.sources[source] = struct{}{}
		return
	}

	// The address of an instance that is already known is not another instance.
	for entry := range m.entries {
		if entry.nodeID != "" && slices.Contains(entry.addresses, address) {
			entry.sources[source] = struct{}{}
			return
		}
	}

	m.byAddress[address] = m.addEntryLocked("", []string{address}, source, learnedAt)
}

func (m *Mesh) addEntryLocked(nodeID string, addresses []string, source string, learnedAt time.Time) *peerEntry {
	entry := &peerEntry{
		nodeID:    nodeID,
		addresses: addresses,
		sources:   map[string]struct{}{source: {}},
		learnedAt: learnedAt,
	}

	entry.link = newLink(m, entry)
	m.entries[entry] = struct{}{}

	m.startLocked(entry)

	return entry
}

// startLocked runs the link of an instance, once the mesh runs.
func (m *Mesh) startLocked(entry *peerEntry) {
	if m.runCtx == nil || m.stopped || entry.cancel != nil {
		return
	}

	ctx, cancel := context.WithCancel(m.runCtx)
	entry.cancel = cancel

	m.linksWG.Go(func() { entry.link.run(ctx) })
}

func (m *Mesh) removeEntryLocked(entry *peerEntry) {
	delete(m.entries, entry)

	if m.byNode[entry.nodeID] == entry {
		delete(m.byNode, entry.nodeID)
	}

	for _, address := range entry.addresses {
		if m.byAddress[address] == entry {
			delete(m.byAddress, address)
		}
	}

	m.dirty = true
}

// connected records that an instance answered at an address of an entry, and tells the link whether to keep the session.
func (m *Mesh) connected(entry *peerEntry, address string, theirs hello) error {
	now := m.now()

	m.mu.Lock()
	defer m.mu.Unlock()

	if theirs.NodeID == m.member.NodeID() {
		if theirs.Instance != m.member.Instance() {
			return errSameNodeID
		}

		// The address is not tried again, whichever source tells it next.
		m.selfAddresses[address] = struct{}{}

		if entry.nodeID == "" {
			m.removeEntryLocked(entry)
			return errSelf
		}

		entry.addresses = slices.DeleteFunc(entry.addresses, func(a string) bool { return a == address })

		return errors.New("this instance answers at the address")
	}

	switch {
	case entry.nodeID == theirs.NodeID:

	case entry.nodeID != "":
		// The address was told for one instance, and another one answers there.
		return fmt.Errorf("the instance %q answers at the address", theirs.NodeID)

	default:
		if existing, known := m.byNode[theirs.NodeID]; known {
			// Two addresses of one instance, e.g. one of the configuration and one the instance announces: one link is enough.
			for source := range entry.sources {
				existing.sources[source] = struct{}{}
			}
			existing.addresses = mergeAddresses(existing.addresses, entry.addresses)

			m.removeEntryLocked(entry)
			existing.link.wake()

			return errMerged
		}

		for _, a := range entry.addresses {
			delete(m.byAddress, a)
		}

		entry.nodeID = theirs.NodeID
		m.byNode[entry.nodeID] = entry
	}

	// The address that worked comes first, then the ones the instance says it is reached at.
	entry.addresses = mergeAddresses(append([]string{address}, cleanAddresses(theirs.Advertise)...), entry.addresses)
	entry.lastContact = now
	m.dirty = true

	return nil
}

// contact records a sign of life of an instance, through a link in either direction.
func (m *Mesh) contact(nodeID string) {
	now := m.now()

	m.mu.Lock()
	defer m.mu.Unlock()

	if entry, known := m.byNode[nodeID]; known {
		entry.lastContact = now
	}
}

func (m *Mesh) addressesOf(entry *peerEntry) []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(entry.addresses)
}

// knownPeers returns the instances to tell the others about: the ones this instance is in contact with right now.
// An instance that is gone must not travel from list to list for ever.
func (m *Mesh) knownPeers() []peerInfo {
	now := m.now()

	m.mu.Lock()
	defer m.mu.Unlock()

	var peers []peerInfo
	for entry := range m.entries {
		if entry.nodeID != "" && now.Sub(entry.lastContact) <= 3*m.linkTimeout {
			peers = append(peers, peerInfo{NodeID: entry.nodeID, Addresses: slices.Clone(entry.addresses)})
		}
	}

	slices.SortFunc(peers, func(a, b peerInfo) int { return strings.Compare(a.NodeID, b.NodeID) })

	return peers[:min(len(peers), maxExchangedPeers)]
}

// learnPeers records the instances another instance tells about.
func (m *Mesh) learnPeers(peers []peerInfo) {
	for _, peer := range peers[:min(len(peers), maxExchangedPeers)] {
		if peer.NodeID != "" {
			m.learn(sourceExchange, peer.NodeID, peer.Addresses)
		}
	}
}

// discover follows the sources of instances, forgets the ones that are gone, and keeps the file of learned instances.
func (m *Mesh) discover(ctx context.Context) {
	m.discoverRedis(ctx)
	m.discoverDNS(ctx)

	refresh := time.NewTicker(m.discoverInterval)
	defer refresh.Stop()

	dns := time.NewTicker(m.dnsInterval)
	defer dns.Stop()

	file := time.NewTicker(peersFileInterval)
	defer file.Stop()

	for {
		select {
		case <-refresh.C:
			m.discoverRedis(ctx)
			m.forgetLost()
			m.savePeers(false)

		case <-dns.C:
			m.discoverDNS(ctx)

		case <-file.C:
			m.savePeers(true)

		case <-ctx.Done():
			return
		}
	}
}

// discoverRedis learns the instances that announce a mesh address in the Redis server of the cluster.
func (m *Mesh) discoverRedis(ctx context.Context) {
	if !m.member.HasServer() {
		return
	}

	nodes, err := m.member.Nodes(ctx)
	if err != nil {
		m.logger.Debug().Err(err).Msg("Unable to list the instances of the cluster for the mesh")
		return
	}

	for _, node := range nodes {
		addresses := node.Attributes[attributeAddresses]
		if node.ID == m.member.NodeID() || addresses == "" {
			continue
		}

		if !m.identity.Accepts(node.Attributes[attributeAuthority]) {
			m.mu.Lock()
			_, warned := m.refused[node.ID]
			m.refused[node.ID] = struct{}{}
			m.mu.Unlock()

			if !warned {
				m.logger.Warn().Str("peer", node.ID).
					Msg("An instance of the cluster holds another cluster token: this instance does not link to it")
			}

			continue
		}

		m.learn(sourceRedis, node.ID, strings.Split(addresses, ","))
	}
}

// discoverDNS learns the addresses that the DNS names of the configuration list.
func (m *Mesh) discoverDNS(ctx context.Context) {
	for _, name := range m.conf.DNS {
		var addresses []string

		lookupCtx, cancel := context.WithTimeout(ctx, dnsTimeout)

		records, srvErr := m.lookupSRV(lookupCtx, name)
		for _, record := range records {
			addresses = append(addresses, net.JoinHostPort(strings.TrimSuffix(record.Target, "."), strconv.Itoa(int(record.Port))))
		}

		texts, txtErr := m.lookupTXT(lookupCtx, name)
		for _, text := range texts {
			if list, ok := strings.CutPrefix(strings.TrimSpace(text), dnsTXTPrefix); ok {
				addresses = append(addresses, strings.Split(list, ",")...)
			}
		}

		cancel()

		if len(addresses) == 0 {
			m.logger.Debug().Errs("errors", []error{srvErr, txtErr}).Str("name", name).Msg("No instance of the mesh found in the DNS")
			continue
		}

		m.learn(sourceDNS, "", addresses)
	}
}

// forgetLost drops the learned instances that have not been in contact for the retention time.
func (m *Mesh) forgetLost() {
	now := m.now()
	retention := time.Duration(m.conf.PeerRetention)

	var forgotten []*peerEntry

	m.mu.Lock()
	for entry := range m.entries {
		if _, configured := entry.sources[sourcePeers]; configured {
			continue
		}

		last := entry.lastContact
		if entry.learnedAt.After(last) {
			last = entry.learnedAt
		}

		if now.Sub(last) <= retention || entry.link.snapshot().Up {
			continue
		}

		m.removeEntryLocked(entry)
		forgotten = append(forgotten, entry)
	}
	m.mu.Unlock()

	for _, entry := range forgotten {
		if entry.cancel != nil {
			entry.cancel()
		}

		m.logger.Info().Str("peer", entry.nodeID).Strs("addresses", entry.addresses).
			Msg("Forgot an instance of the mesh that has been away for the peerRetention time")
	}

	if len(forgotten) > 0 {
		m.countPeers()
	}
}

// loadPeers learns the instances of the file, unless they have been away for the retention time already.
func (m *Mesh) loadPeers() {
	if m.conf.PeersFile == "" {
		return
	}

	data, err := os.ReadFile(m.conf.PeersFile)
	if errors.Is(err, os.ErrNotExist) {
		return
	}

	var file peersFile
	if err == nil {
		err = json.Unmarshal(data, &file)
	}

	if err != nil {
		// The file only speeds the startup up: the other sources still find the instances.
		m.logger.Warn().Err(err).Str("file", m.conf.PeersFile).Msg("Unable to read the file of the instances of the mesh")
		return
	}

	now := m.now()

	for _, peer := range file.Peers {
		if peer.NodeID == "" || peer.LastContact == nil || now.Sub(*peer.LastContact) > time.Duration(m.conf.PeerRetention) {
			continue
		}

		// The time away goes on counting from the last contact, across the restarts.
		m.learnAt(sourceFile, peer.NodeID, peer.Addresses, *peer.LastContact, *peer.LastContact)
	}

	m.mu.Lock()
	m.dirty = false
	m.mu.Unlock()
}

// savePeers writes the file of the learned instances when something changed, or when asked to anyway.
func (m *Mesh) savePeers(force bool) {
	if m.conf.PeersFile == "" {
		return
	}

	m.mu.Lock()
	if !m.dirty && !force {
		m.mu.Unlock()
		return
	}
	m.dirty = false

	file := peersFile{Version: peersFileVersion, Peers: []peerInfo{}}
	for entry := range m.entries {
		// An instance that never answered has nothing worth dialing at the next startup.
		if entry.nodeID == "" || entry.lastContact.IsZero() {
			continue
		}

		lastContact := entry.lastContact.UTC().Truncate(time.Second)
		file.Peers = append(file.Peers, peerInfo{NodeID: entry.nodeID, Addresses: slices.Clone(entry.addresses), LastContact: &lastContact})
	}
	m.mu.Unlock()

	slices.SortFunc(file.Peers, func(a, b peerInfo) int { return strings.Compare(a.NodeID, b.NodeID) })

	data, err := json.MarshalIndent(file, "", "  ")
	if err == nil {
		err = writeFileAtomic(m.conf.PeersFile, append(data, '\n'))
	}

	if err != nil {
		m.logger.Warn().Err(err).Str("file", m.conf.PeersFile).Msg("Unable to write the file of the instances of the mesh")
	}
}

// writeFileAtomic replaces a file in one step: a reader, or a crash, never meets half of it.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), path)
}

// cleanAddresses keeps the addresses with a host and a port, once each, in order, up to the limit.
func cleanAddresses(addresses []string) []string {
	var clean []string

	for _, address := range addresses {
		address = strings.TrimSpace(address)

		host, rawPort, err := net.SplitHostPort(address)
		if err != nil || host == "" || len(address) > 261 {
			continue
		}

		if port, err := strconv.Atoi(rawPort); err != nil || port < 1 || port > 65535 {
			continue
		}

		if !slices.Contains(clean, address) {
			clean = append(clean, address)
		}

		if len(clean) == maxPeerAddresses {
			break
		}
	}

	return clean
}

// mergeAddresses returns the first addresses followed by the others, each one once, up to the limit.
func mergeAddresses(first, others []string) []string {
	merged := make([]string, 0, len(first)+len(others))

	for _, address := range slices.Concat(first, others) {
		if !slices.Contains(merged, address) {
			merged = append(merged, address)
		}
	}

	return merged[:min(len(merged), maxPeerAddresses)]
}
