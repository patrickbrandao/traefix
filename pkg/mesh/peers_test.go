package mesh

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v3/pkg/cluster"
)

func Test_cleanAddresses(t *testing.T) {
	addresses := cleanAddresses([]string{
		" 203.0.113.10:7443 ", "srv1.example.com:7443", "[2001:db8::1]:7443", "203.0.113.10:7443",
		"", "srv1.example.com", ":7443", "srv1.example.com:0", "srv1.example.com:70000", "srv1.example.com:port",
	})

	assert.Equal(t, []string{"203.0.113.10:7443", "srv1.example.com:7443", "[2001:db8::1]:7443"}, addresses)

	var many []string
	for i := range 20 {
		many = append(many, net.JoinHostPort("192.0.2.1", strconv.Itoa(7000+i)))
	}

	assert.Len(t, cleanAddresses(many), maxPeerAddresses)
}

func Test_mergeAddresses(t *testing.T) {
	assert.Equal(t, []string{"b:1", "a:1", "c:1"}, mergeAddresses([]string{"b:1", "a:1"}, []string{"a:1", "b:1", "c:1"}))
	assert.Equal(t, []string{"a:1"}, mergeAddresses(nil, []string{"a:1"}))
	assert.Equal(t, []string{"a:1", "b:1"}, mergeAddresses([]string{"a:1", "a:1", "b:1"}, []string{"b:1"}))
}

// A new instance that knows one instance of the mesh finds the others, and every instance links to every other one.
func TestMesh_discover_exchange(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv2", testToken, nil, first.address)
	third := newTestNode(t, "srv3", testToken, nil, first.address)

	first.waitLinked("srv2", "srv3")
	second.waitLinked("srv1", "srv3")
	third.waitLinked("srv1", "srv2")

	link, _ := second.linkTo("srv3")
	assert.Equal(t, []string{sourceExchange}, link.Sources)
	assert.Equal(t, []string{third.address}, link.Addresses)

	link, _ = second.linkTo("srv1")
	assert.Contains(t, link.Sources, sourcePeers)

	// An instance that joins later is found the same way, by the ones that were not told about it.
	fourth := newTestNode(t, "srv4", testToken, nil, third.address)

	fourth.waitLinked("srv1", "srv2", "srv3")
	first.waitLinked("srv2", "srv3", "srv4")
}

func withRedis(server *miniredis.Miniredis) func(conf *cluster.Configuration) {
	return func(conf *cluster.Configuration) {
		conf.Redis = &cluster.Redis{}
		conf.Redis.SetDefaults()
		conf.Redis.Endpoints = []string{server.Addr()}
		conf.HeartbeatInterval = ptypes.Duration(100 * time.Millisecond)
		conf.NodeTimeout = ptypes.Duration(time.Second)
	}
}

// The instances that share a Redis server find each other there, without any list.
func TestMesh_discover_redis(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestNode(t, "srv1", testToken, withRedis(server))
	second := newTestNode(t, "srv2", testToken, withRedis(server))
	third := newTestNode(t, "srv3", testToken, withRedis(server))

	// An instance of the cluster with another token is left alone, instead of being dialed for ever.
	stranger := newTestNode(t, "srv9", testOtherToken, withRedis(server))

	first.waitLinked("srv2", "srv3")
	second.waitLinked("srv1", "srv3")
	third.waitLinked("srv1", "srv2")

	// An instance can hear of another one from that instance first: the Redis server is recorded as a source when it is read.
	require.Eventually(t, func() bool {
		link, _ := first.linkTo("srv2")
		return slices.Contains(link.Sources, sourceRedis)
	}, 5*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		first.mesh.mu.Lock()
		defer first.mesh.mu.Unlock()

		_, refused := first.mesh.refused["srv9"]

		return refused
	}, 5*time.Second, 10*time.Millisecond)

	assert.Empty(t, stranger.mesh.Status().Inbound)

	for _, link := range stranger.mesh.Status().Links {
		assert.False(t, link.Up)
	}
}

func TestMesh_discover_dns(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv2", testToken, nil)

	firstHost, firstPort, err := net.SplitHostPort(first.address)
	require.NoError(t, err)

	port, err := strconv.Atoi(firstPort)
	require.NoError(t, err)

	third := newIdleTestNode(t, "srv3", testToken, func(conf *cluster.Configuration) {
		conf.Mesh.DNS = []string{"_traefix-mesh._udp.example.com", "mesh.example.com", "missing.example.com"}
	})

	third.lookupSRV = func(_ context.Context, name string) ([]*net.SRV, error) {
		if name != "_traefix-mesh._udp.example.com" {
			return nil, errors.New("no such host")
		}

		// A resolver writes the target with its final dot.
		return []*net.SRV{{Target: firstHost + ".", Port: uint16(port)}}, nil //nolint:gosec // A port.
	}

	third.lookupTXT = func(_ context.Context, name string) ([]string, error) {
		if name != "mesh.example.com" {
			return nil, errors.New("no such host")
		}

		return []string{"v=spf1 -all", "traefix-mesh=" + second.address + ", 127.0.0.1:1"}, nil
	}

	third.start()

	require.Eventually(t, func() bool {
		srv1, ok1 := third.linkTo("srv1")
		srv2, ok2 := third.linkTo("srv2")

		return ok1 && ok2 && srv1.Up && srv2.Up
	}, 15*time.Second, 10*time.Millisecond)

	link, _ := third.linkTo("srv1")
	assert.Contains(t, link.Sources, sourceDNS)

	// The address of the record that nobody answers at stays a link that is down.
	var down int
	for _, link := range third.mesh.Status().Links {
		if !link.Up {
			down++
			assert.Equal(t, "127.0.0.1:1", link.Address)
			assert.Equal(t, []string{sourceDNS}, link.Sources)
		}
	}

	assert.Equal(t, 1, down)
}

func TestMesh_peersFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "mesh-peers.json")

	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv2", testToken, nil, first.address)

	third := newTestNode(t, "srv3", testToken, func(conf *cluster.Configuration) { conf.Mesh.PeersFile = file }, first.address)

	third.waitLinked("srv1", "srv2")

	// The file is written while the instance runs: a crash does not lose it.
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(file)
		if err != nil {
			return false
		}

		var saved peersFile

		return json.Unmarshal(data, &saved) == nil && len(saved.Peers) == 2
	}, 10*time.Second, 10*time.Millisecond)

	third.stop()

	data, err := os.ReadFile(file)
	require.NoError(t, err)

	var saved peersFile
	require.NoError(t, json.Unmarshal(data, &saved))

	require.Len(t, saved.Peers, 2)
	assert.Equal(t, peersFileVersion, saved.Version)
	assert.Equal(t, "srv1", saved.Peers[0].NodeID)
	assert.Equal(t, []string{first.address}, saved.Peers[0].Addresses)
	assert.Equal(t, "srv2", saved.Peers[1].NodeID)
	assert.Equal(t, []string{second.address}, saved.Peers[1].Addresses)
	require.NotNil(t, saved.Peers[1].LastContact)

	// The instance comes back with no list and nobody to ask: the file is enough to link to everyone again.
	third.conf.Mesh.Peers = nil
	third.start()

	third.waitLinked("srv1", "srv2")

	link, _ := third.linkTo("srv2")
	assert.Contains(t, link.Sources, sourceFile)
}

func TestMesh_peersFile_tooOldOrBroken(t *testing.T) {
	file := filepath.Join(t.TempDir(), "mesh-peers.json")

	recent := time.Now().Add(-time.Hour).UTC()
	old := time.Now().Add(-30 * 24 * time.Hour).UTC()

	content, err := json.Marshal(peersFile{Version: peersFileVersion, Peers: []peerInfo{
		{NodeID: "recent", Addresses: []string{"127.0.0.1:1"}, LastContact: &recent},
		{NodeID: "old", Addresses: []string{"127.0.0.1:2"}, LastContact: &old},
		{NodeID: "undated", Addresses: []string{"127.0.0.1:3"}},
		{NodeID: "not a node", Addresses: []string{"127.0.0.1:4"}, LastContact: &recent},
		{NodeID: "unreachable", Addresses: []string{"nowhere"}, LastContact: &recent},
	}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(file, content, 0o600))

	node := newTestNode(t, "srv1", testToken, func(conf *cluster.Configuration) { conf.Mesh.PeersFile = file })

	// Only the instance that was in contact within the retention time is tried again.
	links := node.mesh.Status().Links
	require.Len(t, links, 1)
	assert.Equal(t, "recent", links[0].NodeID)
	assert.Equal(t, []string{sourceFile}, links[0].Sources)

	node.stop()

	// A file that cannot be read does not stop the instance.
	require.NoError(t, os.WriteFile(file, []byte("{"), 0o600))

	node.start()
	assert.Empty(t, node.mesh.Status().Links)
}

func TestMesh_forgetLost(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv2", testToken, nil, first.address)
	third := newTestNode(t, "srv3", testToken, nil, first.address)

	second.waitLinked("srv1", "srv3")

	first.stop()
	third.stop()

	require.Eventually(t, func() bool {
		srv1, _ := second.linkTo("srv1")
		srv3, _ := second.linkTo("srv3")

		return !srv1.Up && !srv3.Up
	}, 10*time.Second, 10*time.Millisecond)

	// Within the retention time, both are still dialed.
	second.advance(time.Duration(second.conf.Mesh.PeerRetention) - time.Hour)
	time.Sleep(200 * time.Millisecond)

	_, known := second.linkTo("srv3")
	require.True(t, known)

	// After it, the instance that was learned is forgotten. The one of the configuration never is.
	second.advance(2 * time.Hour)

	require.Eventually(t, func() bool {
		_, known := second.linkTo("srv3")
		return !known
	}, 10*time.Second, 10*time.Millisecond)

	_, known = second.linkTo("srv1")
	assert.True(t, known)
}

// The address of the configuration and the address an instance announces are one instance, and one link.
func TestMesh_oneLinkForOneInstance(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)

	_, port, err := net.SplitHostPort(first.address)
	require.NoError(t, err)

	// The same instance, under another name than the one it announces.
	alias := net.JoinHostPort("localhost", port)

	second := newTestNode(t, "srv2", testToken, nil, alias)

	second.waitLinked("srv1")

	// Told again about the instance, by its name and by its address, the instance keeps its one link.
	second.mesh.learn(sourceExchange, "srv1", []string{first.address})
	second.mesh.learn(sourceDNS, "", []string{first.address, alias})

	time.Sleep(300 * time.Millisecond)

	second.waitLinked("srv1")

	link, _ := second.linkTo("srv1")
	assert.ElementsMatch(t, []string{alias, first.address}, link.Addresses)
	assert.Equal(t, []string{sourceDNS, sourceExchange, sourcePeers}, link.Sources)
}

func TestMesh_anotherInstanceAtTheAddress(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv2", testToken, nil)

	// The address was told for an instance, and another one answers there: its certificate says so.
	second.mesh.learn(sourceExchange, "srv7", []string{first.address})

	require.Eventually(t, func() bool {
		link, ok := second.linkTo("srv7")
		return ok && link.Error != ""
	}, 10*time.Second, 10*time.Millisecond)

	link, _ := second.linkTo("srv7")
	assert.False(t, link.Up)
	assert.Contains(t, link.Error, `"srv1" answers at the address`)
}

func TestMesh_knownPeers(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv2", testToken, nil, first.address)

	first.waitLinked("srv2")
	second.waitLinked("srv1")

	peers := first.mesh.knownPeers()
	require.Len(t, peers, 1)
	assert.Equal(t, "srv2", peers[0].NodeID)
	assert.Equal(t, []string{second.address}, peers[0].Addresses)

	// An instance that is gone is not told about any more, so that it does not travel from list to list.
	second.stop()
	first.advance(time.Minute)

	assert.Empty(t, first.mesh.knownPeers())
}
