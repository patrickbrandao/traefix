package cluster

import (
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"time"

	ptypes "github.com/traefik/paerser/types"
)

const (
	// MeshTransportAuto links the instances with QUIC, and with TCP when QUIC does not get through.
	MeshTransportAuto = "auto"
	// MeshTransportQUIC links the instances with QUIC only.
	MeshTransportQUIC = "quic"
	// MeshTransportTCP links the instances with TCP only.
	MeshTransportTCP = "tcp"

	// MeshExportAll announces every router of the local providers, except the ones that opt out.
	MeshExportAll = "all"
	// MeshExportLabeled announces only the routers that opt in.
	MeshExportLabeled = "labeled"

	// MeshBalanceClientIP sends a client to the same instance, whichever instance it reaches first.
	MeshBalanceClientIP = "clientIP"
	// MeshBalanceRoundRobin spreads the requests evenly.
	MeshBalanceRoundRobin = "roundRobin"
)

// Mesh links this instance to the other instances of the cluster:
// each instance announces its routers, and forwards the requests for the routers of the others (Traefix).
type Mesh struct {
	Address       string          `description:"Address the instance listens on for the other instances, in UDP for QUIC and in TCP." json:"address,omitempty" toml:"address,omitempty" yaml:"address,omitempty" export:"true"`
	Advertise     []string        `description:"Addresses the other instances reach this instance at. Defaults to the address the instance uses to reach the Redis server or its first peer." json:"advertise,omitempty" toml:"advertise,omitempty" yaml:"advertise,omitempty" export:"true"`
	Transport     string          `description:"Transport of the links: auto (QUIC, then TCP), quic or tcp." json:"transport,omitempty" toml:"transport,omitempty" yaml:"transport,omitempty" export:"true"`
	Peers         []string        `description:"Addresses of instances to link to at startup." json:"peers,omitempty" toml:"peers,omitempty" yaml:"peers,omitempty" export:"true"`
	DNS           []string        `description:"DNS names whose SRV or TXT records list the addresses of the instances." json:"dns,omitempty" toml:"dns,omitempty" yaml:"dns,omitempty" export:"true"`
	PeersFile     string          `description:"File that keeps the instances learned from the others, to link to them at the next startup." json:"peersFile,omitempty" toml:"peersFile,omitempty" yaml:"peersFile,omitempty" export:"true"`
	LinkTimeout   ptypes.Duration `description:"Time without an answer after which a link is considered lost." json:"linkTimeout,omitempty" toml:"linkTimeout,omitempty" yaml:"linkTimeout,omitempty" export:"true"`
	PeerRetention ptypes.Duration `description:"Time without a link after which a learned instance is forgotten." json:"peerRetention,omitempty" toml:"peerRetention,omitempty" yaml:"peerRetention,omitempty" export:"true"`
	Export        string          `description:"Routers announced to the other instances: all, or labeled for the ones that opt in." json:"export,omitempty" toml:"export,omitempty" yaml:"export,omitempty" export:"true"`
	Balance       string          `description:"Choice between the instances with the same priority for a router: clientIP or roundRobin." json:"balance,omitempty" toml:"balance,omitempty" yaml:"balance,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (m *Mesh) SetDefaults() {
	m.Address = ":7443"
	m.Transport = MeshTransportAuto
	m.LinkTimeout = ptypes.Duration(10 * time.Second)
	m.PeerRetention = ptypes.Duration(7 * 24 * time.Hour)
	m.Export = MeshExportAll
	m.Balance = MeshBalanceClientIP
}

func (m *Mesh) validate() error {
	if err := checkAddress(m.Address, false); err != nil {
		return fmt.Errorf("cluster mesh address: %w", err)
	}

	for _, address := range m.Advertise {
		if err := checkAddress(address, true); err != nil {
			return fmt.Errorf("cluster mesh advertise: %w", err)
		}
	}

	for _, address := range m.Peers {
		if err := checkAddress(address, true); err != nil {
			return fmt.Errorf("cluster mesh peers: %w", err)
		}
	}

	if slices.Contains(m.DNS, "") {
		return errors.New("cluster mesh dns: a name is empty")
	}

	switch m.Transport {
	case MeshTransportAuto, MeshTransportQUIC, MeshTransportTCP:
	default:
		return fmt.Errorf("cluster mesh transport %q must be auto, quic or tcp", m.Transport)
	}

	switch m.Export {
	case MeshExportAll, MeshExportLabeled:
	default:
		return fmt.Errorf("cluster mesh export %q must be all or labeled", m.Export)
	}

	switch m.Balance {
	case MeshBalanceClientIP, MeshBalanceRoundRobin:
	default:
		return fmt.Errorf("cluster mesh balance %q must be clientIP or roundRobin", m.Balance)
	}

	if time.Duration(m.LinkTimeout) < time.Second {
		return errors.New("cluster mesh linkTimeout must be at least 1 second")
	}

	// An instance that is only slow to come back must not be forgotten.
	if m.PeerRetention < m.LinkTimeout {
		return errors.New("cluster mesh peerRetention cannot be shorter than the linkTimeout")
	}

	return nil
}

// checkAddress checks the host and the port of an address. The host may only be empty for an address to listen on.
func checkAddress(address string, needsHost bool) error {
	host, rawPort, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%q must be a host and a port", address)
	}

	port, err := strconv.Atoi(rawPort)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("%q must have a port between 1 and 65535", address)
	}

	if needsHost && host == "" {
		return fmt.Errorf("%q must have a host", address)
	}

	return nil
}
