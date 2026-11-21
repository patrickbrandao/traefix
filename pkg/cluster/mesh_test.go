package cluster

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptypes "github.com/traefik/paerser/types"
)

const (
	testToken      = "0123456789abcdef0123456789abcdef"
	testOtherToken = "fedcba9876543210fedcba9876543210"
)

func newTestMeshConfiguration() *Configuration {
	conf := &Configuration{Mesh: &Mesh{}}
	conf.SetDefaults()
	conf.Mesh.SetDefaults()
	conf.Token = testToken

	return conf
}

func TestConfiguration_Validate_mesh(t *testing.T) {
	testCases := []struct {
		desc      string
		change    func(conf *Configuration)
		expectErr bool
	}{
		{desc: "defaults, without redis", change: func(*Configuration) {}},
		{desc: "with redis", change: func(conf *Configuration) {
			conf.Redis = &Redis{}
			conf.Redis.SetDefaults()
		}},
		{desc: "every option", change: func(conf *Configuration) {
			conf.AdditionalTokens = []string{testOtherToken}
			conf.Mesh.Address = "192.0.2.1:7443"
			conf.Mesh.Advertise = []string{"203.0.113.10:7443", "srv1.example.com:7443", "[2001:db8::1]:7443"}
			conf.Mesh.Transport = MeshTransportTCP
			conf.Mesh.Peers = []string{"srv2.example.com:7443"}
			conf.Mesh.DNS = []string{"_traefix-mesh._udp.example.com"}
			conf.Mesh.Export = MeshExportLabeled
			conf.Mesh.Balance = MeshBalanceRoundRobin
		}},
		{desc: "neither redis nor mesh", change: func(conf *Configuration) { conf.Mesh = nil }, expectErr: true},
		{desc: "no token", change: func(conf *Configuration) { conf.Token = "" }, expectErr: true},
		{desc: "short token", change: func(conf *Configuration) { conf.Token = "secret" }, expectErr: true},
		{desc: "token and tokenFile", change: func(conf *Configuration) { conf.TokenFile = "/run/secrets/cluster" }, expectErr: true},
		{desc: "short additional token", change: func(conf *Configuration) { conf.AdditionalTokens = []string{"secret"} }, expectErr: true},
		{desc: "address without port", change: func(conf *Configuration) { conf.Mesh.Address = "192.0.2.1" }, expectErr: true},
		{desc: "address with port 0", change: func(conf *Configuration) { conf.Mesh.Address = ":0" }, expectErr: true},
		{desc: "advertise without host", change: func(conf *Configuration) { conf.Mesh.Advertise = []string{":7443"} }, expectErr: true},
		{desc: "peer without port", change: func(conf *Configuration) { conf.Mesh.Peers = []string{"srv2.example.com"} }, expectErr: true},
		{desc: "empty dns name", change: func(conf *Configuration) { conf.Mesh.DNS = []string{""} }, expectErr: true},
		{desc: "unknown transport", change: func(conf *Configuration) { conf.Mesh.Transport = "sctp" }, expectErr: true},
		{desc: "unknown export", change: func(conf *Configuration) { conf.Mesh.Export = "none" }, expectErr: true},
		{desc: "unknown balance", change: func(conf *Configuration) { conf.Mesh.Balance = "random" }, expectErr: true},
		{desc: "short linkTimeout", change: func(conf *Configuration) { conf.Mesh.LinkTimeout = ptypes.Duration(100 * time.Millisecond) }, expectErr: true},
		{desc: "peerRetention under linkTimeout", change: func(conf *Configuration) { conf.Mesh.PeerRetention = ptypes.Duration(time.Second) }, expectErr: true},
		{desc: "satellite with peers", change: func(conf *Configuration) {
			conf.Mesh.Satellite = true
			conf.Mesh.Peers = []string{"edge1.example.com:7443"}
		}},
		{desc: "satellite with dns", change: func(conf *Configuration) {
			conf.Mesh.Satellite = true
			conf.Mesh.DNS = []string{"_traefix-mesh._udp.example.com"}
		}},
		{desc: "satellite without peers nor dns", change: func(conf *Configuration) { conf.Mesh.Satellite = true }, expectErr: true},
		{desc: "satellite with an advertise address", change: func(conf *Configuration) {
			conf.Mesh.Satellite = true
			conf.Mesh.Peers = []string{"edge1.example.com:7443"}
			conf.Mesh.Advertise = []string{"203.0.113.10:7443"}
		}, expectErr: true},
		{desc: "satellite with redis", change: func(conf *Configuration) {
			conf.Mesh.Satellite = true
			conf.Mesh.Peers = []string{"edge1.example.com:7443"}
			conf.Redis = &Redis{}
			conf.Redis.SetDefaults()
		}, expectErr: true},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			conf := newTestMeshConfiguration()
			test.change(conf)

			err := conf.Validate()
			if test.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestConfiguration_Validate_additionalTokensWithoutToken(t *testing.T) {
	conf := &Configuration{Redis: &Redis{}}
	conf.SetDefaults()
	conf.Redis.SetDefaults()

	// The token is optional without the mesh, but tokens to accept make no sense without one.
	require.NoError(t, conf.Validate())

	conf.AdditionalTokens = []string{testOtherToken}
	require.Error(t, conf.Validate())
}

func TestConfiguration_Tokens(t *testing.T) {
	conf := newTestMeshConfiguration()
	conf.AdditionalTokens = []string{testOtherToken, testToken, testOtherToken}

	token, additional, err := conf.Tokens()
	require.NoError(t, err)
	assert.Equal(t, testToken, token)
	assert.Equal(t, []string{testOtherToken}, additional)
}

func TestConfiguration_Tokens_file(t *testing.T) {
	file := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(file, []byte(testToken+"\n"), 0o600))

	conf := newTestMeshConfiguration()
	conf.Token = ""
	conf.TokenFile = file

	token, _, err := conf.Tokens()
	require.NoError(t, err)
	assert.Equal(t, testToken, token)
	require.NoError(t, conf.Validate())

	testCases := []struct {
		desc    string
		content string
	}{
		{desc: "empty file", content: "\n"},
		{desc: "short token", content: "secret\n"},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			require.NoError(t, os.WriteFile(file, []byte(test.content), 0o600))
			require.Error(t, conf.Validate())
		})
	}

	conf.TokenFile = filepath.Join(t.TempDir(), "missing")
	err = conf.Validate()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), testToken)
}

func TestCluster_withoutServer(t *testing.T) {
	conf := newTestMeshConfiguration()
	conf.NodeID = "srv1"

	c, err := New(t.Context(), conf)
	require.NoError(t, err)

	assert.False(t, c.HasServer())
	assert.False(t, c.Up())
	assert.Equal(t, "traefix", c.Prefix())

	c.SetAttribute("mesh.addr", "192.0.2.1:7443")

	// The instance is alone as far as the cluster knows: the mesh brings the others.
	nodes, err := c.Nodes(t.Context())
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, "srv1", nodes[0].ID)
	assert.Equal(t, "192.0.2.1:7443", nodes[0].Attributes["mesh.addr"])

	_, _, err = c.TryLock(t.Context(), "name", time.Second)
	require.ErrorIs(t, err, ErrNoServer)

	_, _, err = c.LockHolder(t.Context(), "name")
	require.ErrorIs(t, err, ErrNoServer)

	require.ErrorIs(t, c.Publish(t.Context(), "kind"), ErrNoServer)

	// Run has nothing to announce, and stops with its context.
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.Run(ctx)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "Run did not stop")
	}
}
