package mesh

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// testNAT stands between a satellite and one other instance, like the NAT of a home router: the satellite sends to the
// NAT in place of the other instance, the other instance sees the satellite at an address of the NAT, and only the
// packets that come back from that instance pass. The mapping is per destination, as in a symmetric NAT.
type testNAT struct {
	inside *net.UDPConn
	target *net.UDPAddr

	mu       sync.Mutex
	mappings map[string]*natMapping
	closed   bool

	// dropped counts the packets that reached a mapping from another address than the target.
	dropped atomic.Int64
}

type natMapping struct {
	outside   *net.UDPConn
	satellite *net.UDPAddr
}

func newTestNAT(t *testing.T, target string) *testNAT {
	t.Helper()

	targetAddr, err := net.ResolveUDPAddr("udp", target)
	require.NoError(t, err)

	inside, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err)

	nat := &testNAT{inside: inside, target: targetAddr, mappings: map[string]*natMapping{}}

	go nat.readInside()

	t.Cleanup(nat.close)

	return nat
}

// address is where the satellite reaches the other instance, through the NAT.
func (n *testNAT) address() string {
	return n.inside.LocalAddr().String()
}

// mapped returns the addresses the other instance sees the satellite at.
func (n *testNAT) mapped() []string {
	n.mu.Lock()
	defer n.mu.Unlock()

	var addresses []string
	for _, mapping := range n.mappings {
		addresses = append(addresses, mapping.outside.LocalAddr().String())
	}

	return addresses
}

// remap forgets the mappings, as a NAT that flushes its table: the next packets of the satellite get a new port.
func (n *testNAT) remap() {
	n.mu.Lock()
	defer n.mu.Unlock()

	for key, mapping := range n.mappings {
		_ = mapping.outside.Close()
		delete(n.mappings, key)
	}
}

func (n *testNAT) readInside() {
	buf := make([]byte, maxPacketSize)

	for {
		size, from, err := n.inside.ReadFromUDP(buf)
		if err != nil {
			return
		}

		mapping := n.mappingFor(from)
		if mapping == nil {
			return
		}

		_, _ = mapping.outside.WriteToUDP(buf[:size], n.target)
	}
}

func (n *testNAT) mappingFor(satellite *net.UDPAddr) *natMapping {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.closed {
		return nil
	}

	if mapping, ok := n.mappings[satellite.String()]; ok {
		return mapping
	}

	outside, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		return nil
	}

	mapping := &natMapping{outside: outside, satellite: satellite}
	n.mappings[satellite.String()] = mapping

	go n.readOutside(mapping)

	return mapping
}

func (n *testNAT) readOutside(mapping *natMapping) {
	buf := make([]byte, maxPacketSize)

	for {
		size, from, err := mapping.outside.ReadFromUDP(buf)
		if err != nil {
			return
		}

		if !from.IP.Equal(n.target.IP) || from.Port != n.target.Port {
			n.dropped.Add(1)
			continue
		}

		_, _ = n.inside.WriteToUDP(buf[:size], mapping.satellite)
	}
}

func (n *testNAT) close() {
	n.mu.Lock()
	n.closed = true
	n.mu.Unlock()

	n.remap()
	_ = n.inside.Close()
}
