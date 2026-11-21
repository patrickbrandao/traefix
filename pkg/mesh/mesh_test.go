package mesh

import (
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
)

func newTestConfiguration() *cluster.Configuration {
	conf := &cluster.Configuration{Mesh: &cluster.Mesh{}}
	conf.SetDefaults()
	conf.Mesh.SetDefaults()
	conf.NodeID = "srv1"
	conf.Token = testToken

	return conf
}

func Test_advertisedAddresses(t *testing.T) {
	outbound := func(answers map[string]string) func(string) (net.IP, error) {
		return func(target string) (net.IP, error) {
			answer, ok := answers[target]
			if !ok {
				return nil, errors.New("unreachable")
			}

			return net.ParseIP(answer), nil
		}
	}

	testCases := []struct {
		desc            string
		change          func(conf *cluster.Configuration)
		outbound        map[string]string
		expected        []string
		expectedGuessed bool
	}{
		{
			desc:     "advertise option",
			change:   func(conf *cluster.Configuration) { conf.Mesh.Advertise = []string{"203.0.113.10:8443"} },
			expected: []string{"203.0.113.10:8443"},
		},
		{
			desc:     "address of one interface",
			change:   func(conf *cluster.Configuration) { conf.Mesh.Address = "192.0.2.1:7443" },
			expected: []string{"192.0.2.1:7443"},
		},
		{
			desc: "address used to reach the Redis server",
			change: func(conf *cluster.Configuration) {
				conf.Redis = &cluster.Redis{Endpoints: []string{"valkey:6379"}}
				conf.Mesh.Peers = []string{"srv2.example.com:7443"}
			},
			outbound:        map[string]string{"valkey:6379": "10.0.0.5", "srv2.example.com:7443": "198.51.100.7"},
			expected:        []string{"10.0.0.5:7443"},
			expectedGuessed: true,
		},
		{
			desc: "address used to reach the first peer",
			change: func(conf *cluster.Configuration) {
				conf.Mesh.Address = "[::]:9443"
				conf.Mesh.Peers = []string{"down.example.com:7443", "srv2.example.com:7443"}
			},
			outbound:        map[string]string{"srv2.example.com:7443": "2001:db8::5"},
			expected:        []string{"[2001:db8::5]:9443"},
			expectedGuessed: true,
		},
		{
			// A Redis server on the same host tells nothing about the address the other hosts reach.
			desc: "loopback address is not advertised",
			change: func(conf *cluster.Configuration) {
				conf.Redis = &cluster.Redis{Endpoints: []string{"localhost:6379"}}
				conf.Mesh.Peers = []string{"srv2.example.com:7443"}
			},
			outbound:        map[string]string{"localhost:6379": "127.0.0.1", "srv2.example.com:7443": "198.51.100.7"},
			expected:        []string{"198.51.100.7:7443"},
			expectedGuessed: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			conf := newTestConfiguration()
			test.change(conf)

			addresses, guessed, err := advertisedAddresses(conf, outbound(test.outbound))
			require.NoError(t, err)

			assert.Equal(t, test.expected, addresses)
			assert.Equal(t, test.expectedGuessed, guessed)
		})
	}
}

func TestNew(t *testing.T) {
	conf := newTestConfiguration()
	conf.Mesh.Address = freeAddress(t)
	conf.Mesh.Advertise = []string{"203.0.113.10:7443", "srv1.example.com:7443"}
	conf.AdditionalTokens = []string{testOtherToken}

	member, err := cluster.New(t.Context(), conf)
	require.NoError(t, err)

	mesh, err := New(conf, member)
	require.NoError(t, err)
	t.Cleanup(mesh.listener.close)

	assert.Equal(t, conf.Mesh.Advertise, mesh.Advertise())
	assert.Equal(t, "srv1", mesh.Identity().NodeID())

	// The other instances find the addresses and the token fingerprint in the announcement of this one.
	nodes, err := member.Nodes(t.Context())
	require.NoError(t, err)
	require.Len(t, nodes, 1)

	assert.Equal(t, "203.0.113.10:7443,srv1.example.com:7443", nodes[0].Attributes[attributeAddresses])
	assert.Equal(t, mesh.Identity().Fingerprint(), nodes[0].Attributes[attributeAuthority])
	assert.NotContains(t, nodes[0].Attributes[attributeAuthority], testToken)
}

func TestNew_needsTheMeshOption(t *testing.T) {
	conf := newTestConfiguration()

	member, err := cluster.New(t.Context(), conf)
	require.NoError(t, err)

	conf.Mesh = nil

	_, err = New(conf, member)
	require.Error(t, err)
}

func TestNew_addressInUse(t *testing.T) {
	conf := newTestConfiguration()
	conf.Mesh.Address = freeAddress(t)

	member, err := cluster.New(t.Context(), conf)
	require.NoError(t, err)

	first, err := New(conf, member)
	require.NoError(t, err)
	t.Cleanup(first.listener.close)

	// The port is bound at once: a busy one stops the startup instead of leaving an instance that nobody can link to.
	_, err = New(conf, member)
	require.Error(t, err)
}
