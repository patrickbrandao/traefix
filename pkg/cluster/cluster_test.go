package cluster

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptypes "github.com/traefik/paerser/types"
)

func newTestConfiguration(server *miniredis.Miniredis, nodeID string, priority int) *Configuration {
	conf := &Configuration{Redis: &Redis{}}
	conf.SetDefaults()
	conf.Redis.SetDefaults()

	conf.NodeID = nodeID
	conf.Priority = priority
	conf.Redis.Endpoints = []string{server.Addr()}

	return conf
}

func newTestCluster(t *testing.T, server *miniredis.Miniredis, nodeID string, priority int) *Cluster {
	t.Helper()

	c, err := New(t.Context(), newTestConfiguration(server, nodeID, priority))
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.client.Close() })

	return c
}

// runTestCluster runs the instance and waits for it to receive the events.
func runTestCluster(t *testing.T, c *Cluster) context.CancelFunc {
	t.Helper()

	connected := make(chan struct{})
	var once sync.Once
	c.OnConnect(func() { once.Do(func() { close(connected) }) })

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.Run(ctx)
	}()

	stop := func() {
		cancel()
		<-done
	}
	t.Cleanup(stop)

	select {
	case <-connected:
	case <-time.After(5 * time.Second):
		require.Fail(t, "the instance did not connect")
	}

	return stop
}

func TestConfiguration_Validate(t *testing.T) {
	testCases := []struct {
		desc      string
		change    func(conf *Configuration)
		expectErr bool
	}{
		{desc: "defaults", change: func(*Configuration) {}},
		{desc: "priority 0", change: func(conf *Configuration) { conf.Priority = 0 }},
		{desc: "nodeID", change: func(conf *Configuration) { conf.NodeID = "srv-1.example_com" }},
		{desc: "negative priority", change: func(conf *Configuration) { conf.Priority = -1 }, expectErr: true},
		{desc: "nodeID with a separator", change: func(conf *Configuration) { conf.NodeID = "srv|1" }, expectErr: true},
		{desc: "nodeID with a space", change: func(conf *Configuration) { conf.NodeID = "srv 1" }, expectErr: true},
		{desc: "empty prefix", change: func(conf *Configuration) { conf.Prefix = "" }, expectErr: true},
		{desc: "prefix with a colon", change: func(conf *Configuration) { conf.Prefix = "a:b" }, expectErr: true},
		{desc: "no heartbeat", change: func(conf *Configuration) { conf.HeartbeatInterval = 0 }, expectErr: true},
		{desc: "short nodeTimeout", change: func(conf *Configuration) { conf.NodeTimeout = conf.HeartbeatInterval }, expectErr: true},
		{desc: "neither redis nor mesh", change: func(conf *Configuration) { conf.Redis = nil }, expectErr: true},
		{desc: "no endpoints", change: func(conf *Configuration) { conf.Redis.Endpoints = nil }, expectErr: true},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			conf := &Configuration{Redis: &Redis{}}
			conf.SetDefaults()
			conf.Redis.SetDefaults()
			test.change(conf)

			err := conf.Validate()
			if test.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}

	var conf *Configuration
	require.NoError(t, conf.Validate())
}

func TestCluster_Nodes(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestCluster(t, server, "srv1", 100)
	second := newTestCluster(t, server, "srv2", 50)
	second.SetAttribute("acme.le", "digest")

	first.heartbeat(t.Context())
	second.heartbeat(t.Context())

	assert.True(t, first.Up())

	nodes, err := first.Nodes(t.Context())
	require.NoError(t, err)
	require.Len(t, nodes, 2)

	assert.Equal(t, first.self, nodes[0])
	assert.Equal(t, second.self, nodes[1])
	assert.Equal(t, map[string]string{"acme.le": "digest"}, nodes[1].Attributes)

	// An instance that stops announcing itself is gone after the node timeout, counted by the server.
	server.FastForward(10 * time.Second)
	first.heartbeat(t.Context())
	server.FastForward(10 * time.Second)

	nodes, err = second.Nodes(t.Context())
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, "srv1", nodes[0].ID)

	members, err := server.SMembers("traefix:nodes")
	require.NoError(t, err)
	assert.Equal(t, []string{"srv1"}, members)

	// An instance that is back is seen again.
	second.heartbeat(t.Context())

	nodes, err = first.Nodes(t.Context())
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
}

func TestCluster_Up(t *testing.T) {
	server := miniredis.RunT(t)

	c := newTestCluster(t, server, "srv1", 100)
	assert.False(t, c.Up())

	c.heartbeat(t.Context())
	assert.True(t, c.Up())

	server.Close()

	c.heartbeat(t.Context())
	assert.False(t, c.Up())
}

func TestCluster_duplicateNodeID(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestCluster(t, server, "srv1", 100)
	second := newTestCluster(t, server, "srv1", 100)

	var logs bytes.Buffer
	second.logger = zerolog.New(&logs)

	// The first announcement may replace the one of the same instance before a restart.
	first.heartbeat(t.Context())
	second.heartbeat(t.Context())
	assert.NotContains(t, logs.String(), "same cluster nodeID")

	first.heartbeat(t.Context())
	second.heartbeat(t.Context())
	assert.Contains(t, logs.String(), "same cluster nodeID")
}

func TestCluster_Run_leaves(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestCluster(t, server, "srv1", 100)
	stop := runTestCluster(t, first)

	require.Eventually(t, func() bool { return server.Exists("traefix:node:srv1") }, 5*time.Second, 10*time.Millisecond)

	stop()
	assert.False(t, server.Exists("traefix:node:srv1"))
}

func TestCluster_Run_leaveKeepsTheAnnouncementOfAnotherInstance(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestCluster(t, server, "srv1", 100)
	stop := runTestCluster(t, first)

	second := newTestCluster(t, server, "srv1", 100)
	second.heartbeat(t.Context())

	stop()
	assert.True(t, server.Exists("traefix:node:srv1"))
}

func TestCluster_Key(t *testing.T) {
	server := miniredis.RunT(t)

	conf := newTestConfiguration(server, "srv1", 100)
	conf.Prefix = "prod"

	c, err := New(t.Context(), conf)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.client.Close() })

	assert.Equal(t, "prod:acme:le:account", c.Key("acme", "le", "account"))
}

func TestNew_nodeIDFromHostname(t *testing.T) {
	server := miniredis.RunT(t)

	c, err := New(t.Context(), newTestConfiguration(server, "", 100))
	if err != nil {
		// The host name of the machine running the tests may not be usable as an identifier.
		t.Skip(err)
	}
	t.Cleanup(func() { _ = c.client.Close() })

	assert.NotEmpty(t, c.NodeID())
}

func TestCluster_heartbeatInterval(t *testing.T) {
	server := miniredis.RunT(t)

	conf := newTestConfiguration(server, "srv1", 100)
	conf.HeartbeatInterval = ptypes.Duration(20 * time.Millisecond)
	conf.NodeTimeout = ptypes.Duration(time.Minute)

	c, err := New(t.Context(), conf)
	require.NoError(t, err)

	runTestCluster(t, c)

	// The announcement is written again at each interval, with its full time to live.
	server.FastForward(30 * time.Second)
	require.Eventually(t, func() bool { return server.TTL("traefix:node:srv1") == time.Minute }, 5*time.Second, 10*time.Millisecond)
}
