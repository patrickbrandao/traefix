package mesh

import (
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/cluster"
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
}

// New prepares the mesh of an instance: its identity, and the addresses it tells the other instances.
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

	return &Mesh{conf: conf.Mesh, member: member, identity: identity, advertise: advertise}, nil
}

// Identity returns what this instance shows to the others and checks of them.
func (m *Mesh) Identity() *Identity {
	return m.identity
}

// Advertise returns the addresses the other instances reach this instance at.
func (m *Mesh) Advertise() []string {
	return m.advertise
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
