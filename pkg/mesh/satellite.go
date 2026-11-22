package mesh

import (
	"errors"
	"net/http"
)

const (
	// RoleMember is an instance that the other instances reach at its advertised addresses.
	RoleMember = "member"
	// RoleSatellite is an instance that accepts no link: it dials the other instances, and gets its traffic
	// through the links it opened.
	RoleSatellite = "satellite"
)

// errSatellitePeer stops a link to an address where a satellite answers: a satellite is never reached first,
// and a satellite does not link to another one.
var errSatellitePeer = errors.New("a satellite answers at the address: a satellite is only reached through the links it opens")

// role returns the role of this instance in the mesh.
func (m *Mesh) role() string {
	if m.conf.Satellite {
		return RoleSatellite
	}

	return RoleMember
}

// roleOf returns the role another instance told, which is a member for the versions that tell none.
func roleOf(theirs hello) string {
	if theirs.Role == "" {
		return RoleMember
	}

	return theirs.Role
}

// refuseSatelliteLocked keeps a link from being opened to an address where a satellite answers,
// whichever source tells the address next.
func (m *Mesh) refuseSatelliteLocked(entry *peerEntry, address string) {
	m.satelliteAddresses[address] = struct{}{}
	m.removeEntryLocked(entry)
}

// learnSatellite records a satellite that linked to this instance, at the address its link came from: the address
// its NAT maps to it, for the packets that come back from the port of the mesh of this instance.
// The instance dials it back there, from that port, and that link is the one the requests for its routers take.
func (m *Mesh) learnSatellite(nodeID, address string) {
	if nodeID == m.member.NodeID() || !nodeIDRegexp.MatchString(nodeID) || address == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	entry, known := m.byNode[nodeID]

	if known && !entry.satellite {
		// The instance was known as a member, e.g. from the file of the peers, and is a satellite now.
		m.removeEntryLocked(entry)

		if entry.cancel != nil {
			entry.cancel()
		}

		known = false
	}

	if !known {
		entry = &peerEntry{
			nodeID:    nodeID,
			addresses: []string{address},
			sources:   map[string]struct{}{sourceSatellite: {}},
			learnedAt: m.now(),
			satellite: true,
		}

		entry.link = newLink(m, entry)
		m.entries[entry] = struct{}{}
		m.byNode[nodeID] = entry

		m.startLocked(entry)

		m.logger.Info().Str("peer", nodeID).Str("observed", address).
			Msg("A satellite of the mesh linked to this instance: it is reached back at the address its link comes from")

		return
	}

	m.observedLocked(entry, address)
}

// observeSatellite follows the address the link of a satellite comes from. A NAT that maps the satellite to another
// port moves its link, which QUIC carries over, while the link back still goes to the address of before.
func (m *Mesh) observeSatellite(req *http.Request) {
	if req.ProtoMajor != 3 {
		return
	}

	nodeID := peerFrom(req.Context())

	m.mu.Lock()
	defer m.mu.Unlock()

	if entry, known := m.byNode[nodeID]; known && entry.satellite {
		m.observedLocked(entry, req.RemoteAddr)
	}
}

func (m *Mesh) observedLocked(entry *peerEntry, address string) {
	if len(entry.addresses) == 1 && entry.addresses[0] == address {
		return
	}

	previous := entry.addresses
	entry.addresses = []string{address}

	m.logger.Info().Str("peer", entry.nodeID).Strs("previous", previous).Str("observed", address).
		Msg("A satellite of the mesh comes from another address: it is reached back there")

	// The link back goes to a mapping the NAT of the satellite no longer has: it is opened again at the new one.
	if s := entry.link.current(); s != nil {
		go s.Close()
	}

	entry.link.wake()
}

// satelliteAlive tells whether the link of a satellite to this instance gave a sign of life lately:
// this instance reaches a satellite back only for as long as the satellite links to it.
func (m *Mesh) satelliteAlive(nodeID string) bool {
	now := m.now()

	m.mu.Lock()
	defer m.mu.Unlock()

	inbound, ok := m.inbound[nodeID]

	return ok && inbound.Role == RoleSatellite && now.Sub(inbound.LastSeen) <= m.linkTimeout
}

// forgetSatellite drops a satellite whose link to this instance is gone.
func (m *Mesh) forgetSatellite(entry *peerEntry) {
	m.mu.Lock()
	_, current := m.entries[entry]
	if current {
		m.removeEntryLocked(entry)
	}
	m.mu.Unlock()

	if !current {
		return
	}

	if entry.cancel != nil {
		entry.cancel()
	}

	m.logger.Info().Str("peer", entry.nodeID).Msg("A satellite of the mesh no longer links to this instance: it is not reached back any more")

	m.countPeers()
}
