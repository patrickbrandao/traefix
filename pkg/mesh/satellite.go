package mesh

import (
	"errors"
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
