package mesh

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testToken      = "0123456789abcdef0123456789abcdef"
	testOtherToken = "fedcba9876543210fedcba9876543210"
)

func init() {
	// The real count makes each derivation take seconds under the race detector.
	authorityIterations = 1000
}

func newTestIdentity(t *testing.T, nodeID, token string, additional ...string) *Identity {
	t.Helper()

	identity, err := NewIdentity("traefix", nodeID, token, additional)
	require.NoError(t, err)

	return identity
}

// link opens a TLS connection between two instances, and returns the instance each side found at the other end.
func link(t *testing.T, server, client *Identity) (string, string, error) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	type result struct {
		peer string
		err  error
	}

	accepted := make(chan result, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			accepted <- result{err: err}
			return
		}
		defer func() { _ = conn.Close() }()

		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

		serverTLS := tls.Server(conn, server.ServerTLS("traefix-mesh/1"))
		if err := serverTLS.Handshake(); err != nil {
			accepted <- result{err: err}
			return
		}

		peer, err := server.PeerNode(serverTLS.ConnectionState())
		if err != nil {
			accepted <- result{err: err}
			return
		}

		// The side that opens the link only learns that it was accepted when it reads.
		_, err = serverTLS.Write([]byte{1})
		accepted <- result{peer: peer, err: err}
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	clientTLS := tls.Client(conn, client.ClientTLS("traefix-mesh/1"))

	clientErr := clientTLS.Handshake()
	if clientErr == nil {
		_, clientErr = clientTLS.Read(make([]byte, 1))
	}

	if clientErr != nil {
		_ = conn.Close()
	}

	fromServer := <-accepted

	if clientErr != nil {
		return "", "", clientErr
	}

	if fromServer.err != nil {
		return "", "", fromServer.err
	}

	seenByClient, err := client.PeerNode(clientTLS.ConnectionState())

	return fromServer.peer, seenByClient, err
}

func TestDeriveAuthority(t *testing.T) {
	first, err := deriveAuthority(testToken, "traefix")
	require.NoError(t, err)

	second, err := deriveAuthority(testToken, "traefix")
	require.NoError(t, err)

	// Two instances with the same token build the same authority, down to the last byte of its certificate.
	assert.Equal(t, first.cert.Raw, second.cert.Raw)
	assert.Equal(t, first.fingerprint(), second.fingerprint())
	assert.True(t, first.cert.IsCA)

	otherToken, err := deriveAuthority(testOtherToken, "traefix")
	require.NoError(t, err)
	assert.NotEqual(t, first.fingerprint(), otherToken.fingerprint())

	// Two clusters that share a token by mistake are still apart.
	otherPrefix, err := deriveAuthority(testToken, "staging")
	require.NoError(t, err)
	assert.NotEqual(t, first.fingerprint(), otherPrefix.fingerprint())
}

func TestDeriveAuthority_productionCost(t *testing.T) {
	assert.Equal(t, 600_000, defaultAuthorityIterations)

	authorityIterations = defaultAuthorityIterations
	t.Cleanup(func() { authorityIterations = 1000 })

	// The authority of this token and prefix must never change: the instances of different versions could not link.
	authority, err := deriveAuthority(testToken, "traefix")
	require.NoError(t, err)

	assert.Equal(t, "74994b0d2c79d819", authority.fingerprint())
}

func TestIdentity_link(t *testing.T) {
	first := newTestIdentity(t, "srv1", testToken)
	second := newTestIdentity(t, "srv-2.example_com", testToken)

	assert.Equal(t, first.Fingerprint(), second.Fingerprint())
	assert.Equal(t, "srv1", first.NodeID())

	seenByServer, seenByClient, err := link(t, first, second)
	require.NoError(t, err)

	assert.Equal(t, "srv-2.example_com", seenByServer)
	assert.Equal(t, "srv1", seenByClient)
}

func TestIdentity_link_refused(t *testing.T) {
	testCases := []struct {
		desc   string
		server *Identity
		client *Identity
	}{
		{
			desc:   "another token",
			server: newTestIdentity(t, "srv1", testToken),
			client: newTestIdentity(t, "srv2", testOtherToken),
		},
		{
			desc: "same token, another cluster",
			server: func() *Identity {
				identity, err := NewIdentity("staging", "srv1", testToken, nil)
				require.NoError(t, err)

				return identity
			}(),
			client: newTestIdentity(t, "srv2", testToken),
		},
		{
			// Each side must accept the other: one that learned the next token is not enough.
			desc:   "token accepted by one side only",
			server: newTestIdentity(t, "srv1", testToken, testOtherToken),
			client: newTestIdentity(t, "srv2", testOtherToken),
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			_, _, err := link(t, test.server, test.client)
			require.Error(t, err)

			_, _, err = link(t, test.client, test.server)
			require.Error(t, err)
		})
	}
}

func TestIdentity_link_tokenChange(t *testing.T) {
	// The steps of a token change, one instance at a time: every pair of instances can link at every step.
	steps := []struct {
		desc   string
		first  *Identity
		second *Identity
	}{
		{
			desc:   "one instance accepts the next token",
			first:  newTestIdentity(t, "srv1", testToken, testOtherToken),
			second: newTestIdentity(t, "srv2", testToken),
		},
		{
			desc:   "both accept the next token",
			first:  newTestIdentity(t, "srv1", testToken, testOtherToken),
			second: newTestIdentity(t, "srv2", testToken, testOtherToken),
		},
		{
			desc:   "one instance uses the next token",
			first:  newTestIdentity(t, "srv1", testOtherToken, testToken),
			second: newTestIdentity(t, "srv2", testToken, testOtherToken),
		},
		{
			desc:   "both use the next token",
			first:  newTestIdentity(t, "srv1", testOtherToken, testToken),
			second: newTestIdentity(t, "srv2", testOtherToken, testToken),
		},
		{
			desc:   "one instance forgets the previous token",
			first:  newTestIdentity(t, "srv1", testOtherToken),
			second: newTestIdentity(t, "srv2", testOtherToken, testToken),
		},
	}

	for _, step := range steps {
		t.Run(step.desc, func(t *testing.T) {
			_, _, err := link(t, step.first, step.second)
			require.NoError(t, err)

			_, _, err = link(t, step.second, step.first)
			require.NoError(t, err)

			assert.True(t, step.first.Accepts(step.second.Fingerprint()))
			assert.True(t, step.second.Accepts(step.first.Fingerprint()))
		})
	}
}

func TestIdentity_PeerNode_withoutCertificate(t *testing.T) {
	identity := newTestIdentity(t, "srv1", testToken)

	_, err := identity.PeerNode(tls.ConnectionState{})
	require.Error(t, err)
}

func TestNewIdentity_needsAToken(t *testing.T) {
	_, err := NewIdentity("traefix", "srv1", "", nil)
	require.Error(t, err)
}
