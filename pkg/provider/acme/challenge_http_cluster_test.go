package acme

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testChallengeToken   = "evaGxfADs6pSRb2LAv9IZf17Dt3juxGJ-PCt92wr-oA"
	testChallengeKeyAuth = testChallengeToken + ".9jg46WB3rR_AHD-EBXdN7cBkH1WOu0tA3M9fm21mqTI"
)

func newTestClusterChallenge(t *testing.T, server *miniredis.Miniredis, nodeID string) *ClusterChallengeHTTP {
	t.Helper()

	return NewClusterChallengeHTTP(newStoreTestCluster(t, server, nodeID))
}

func askChallenge(t *testing.T, handler http.Handler, host, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/.well-known/acme-challenge/"+token, nil)
	req.Host = host

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	return recorder
}

func TestNewHTTPChallengeSolver(t *testing.T) {
	assert.IsType(t, &ChallengeHTTP{}, NewHTTPChallengeSolver(nil))

	member := newStoreTestCluster(t, miniredis.RunT(t), "srv1")
	assert.IsType(t, &ClusterChallengeHTTP{}, NewHTTPChallengeSolver(member))
}

func TestClusterChallengeHTTP_anotherInstanceAnswers(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestClusterChallenge(t, server, "srv1")
	second := newTestClusterChallenge(t, server, "srv2")

	require.NoError(t, first.Present(t.Context(), "app.example.com", testChallengeToken, testChallengeKeyAuth))

	key := "traefix:acme:http:" + testChallengeToken + ":app.example.com"
	require.True(t, server.Exists(key))
	assert.Equal(t, clusterChallengeTTL, server.TTL(key))

	testCases := []struct {
		desc     string
		handler  http.Handler
		host     string
		expected int
	}{
		{desc: "the instance that presented it", handler: first, host: "app.example.com", expected: http.StatusOK},
		{desc: "another instance", handler: second, host: "app.example.com", expected: http.StatusOK},
		{desc: "host with a port", handler: second, host: "app.example.com:80", expected: http.StatusOK},
		{desc: "host in another case", handler: second, host: "APP.Example.com", expected: http.StatusOK},
		{desc: "another domain", handler: second, host: "other.example.com", expected: http.StatusNotFound},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			recorder := askChallenge(t, test.handler, test.host, testChallengeToken)

			assert.Equal(t, test.expected, recorder.Code)

			if test.expected == http.StatusOK {
				assert.Equal(t, testChallengeKeyAuth, recorder.Body.String())
			}
		})
	}

	require.NoError(t, first.CleanUp(t.Context(), "app.example.com", testChallengeToken, testChallengeKeyAuth))

	assert.False(t, server.Exists(key))
	assert.Equal(t, http.StatusNotFound, askChallenge(t, first, "app.example.com", testChallengeToken).Code)
	assert.Equal(t, http.StatusNotFound, askChallenge(t, second, "app.example.com", testChallengeToken).Code)
}

func TestClusterChallengeHTTP_ipv6(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestClusterChallenge(t, server, "srv1")
	second := newTestClusterChallenge(t, server, "srv2")

	require.NoError(t, first.Present(t.Context(), "2001:db8::1", testChallengeToken, testChallengeKeyAuth))

	assert.Equal(t, http.StatusOK, askChallenge(t, second, "[2001:db8::1]:80", testChallengeToken).Code)
	assert.Equal(t, http.StatusOK, askChallenge(t, second, "[2001:db8::1]", testChallengeToken).Code)
}

func TestClusterChallengeHTTP_doesNotAskForWhatCannotBeAChallenge(t *testing.T) {
	testCases := []struct {
		desc  string
		token string
		host  string
	}{
		{desc: "short token", token: "short", host: "app.example.com"},
		{desc: "token with a dot", token: "xmrlpc.php-0123456789-0123456789", host: "app.example.com"},
		{desc: "host with a space", token: testChallengeToken, host: "app example.com"},
		{desc: "host with a slash", token: testChallengeToken, host: "app/example.com"},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			server := miniredis.RunT(t)
			challenge := newTestClusterChallenge(t, server, "srv1")

			// The answer is there: only the refusal to ask explains the 404.
			require.NoError(t, server.Set("traefix:acme:http:"+test.token+":"+test.host, "found"))

			req := httptest.NewRequest(http.MethodGet, "/.well-known/acme-challenge/"+test.token, nil)
			req.Host = test.host

			recorder := httptest.NewRecorder()
			challenge.ServeHTTP(recorder, req)

			assert.Equal(t, http.StatusNotFound, recorder.Code)
		})
	}
}

func TestClusterChallengeHTTP_remembersAMissForAShortTime(t *testing.T) {
	server := miniredis.RunT(t)
	challenge := newTestClusterChallenge(t, server, "srv1")

	now := time.Now()
	challenge.now = func() time.Time { return now }

	assert.Equal(t, http.StatusNotFound, askChallenge(t, challenge, "app.example.com", testChallengeToken).Code)

	require.NoError(t, server.Set("traefix:acme:http:"+testChallengeToken+":app.example.com", testChallengeKeyAuth))

	assert.Equal(t, http.StatusNotFound, askChallenge(t, challenge, "app.example.com", testChallengeToken).Code)

	now = now.Add(clusterChallengeMissTTL)

	assert.Equal(t, http.StatusOK, askChallenge(t, challenge, "app.example.com", testChallengeToken).Code)
}

func TestClusterChallengeHTTP_presentForgetsAMiss(t *testing.T) {
	server := miniredis.RunT(t)
	challenge := newTestClusterChallenge(t, server, "srv1")

	assert.Equal(t, http.StatusNotFound, askChallenge(t, challenge, "app.example.com", testChallengeToken).Code)

	require.NoError(t, challenge.Present(t.Context(), "app.example.com", testChallengeToken, testChallengeKeyAuth))

	challenge.mu.Lock()
	defer challenge.mu.Unlock()

	assert.Empty(t, challenge.misses)
}

func TestClusterChallengeHTTP_boundsTheMisses(t *testing.T) {
	server := miniredis.RunT(t)
	challenge := newTestClusterChallenge(t, server, "srv1")

	now := time.Now()
	challenge.now = func() time.Time { return now }

	for i := range clusterChallengeMaxMisses + 10 {
		challenge.missed(fmt.Sprintf("key-%d", i))
	}

	assert.Len(t, challenge.misses, clusterChallengeMaxMisses)

	now = now.Add(clusterChallengeMissTTL)
	challenge.missed("fresh")

	assert.Len(t, challenge.misses, 1)
}

func TestClusterChallengeHTTP_serverDown(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestClusterChallenge(t, server, "srv1")
	second := newTestClusterChallenge(t, server, "srv2")

	now := time.Now()
	second.now = func() time.Time { return now }

	server.Close()

	require.NoError(t, first.Present(t.Context(), "app.example.com", testChallengeToken, testChallengeKeyAuth))

	recorder := askChallenge(t, first, "app.example.com", testChallengeToken)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, testChallengeKeyAuth, recorder.Body.String())

	start := time.Now()
	assert.Equal(t, http.StatusNotFound, askChallenge(t, second, "app.example.com", testChallengeToken).Code)
	assert.Less(t, time.Since(start), 2*clusterChallengeLookupTimeout)

	second.mu.Lock()
	pausedUntil := second.pausedUntil
	second.mu.Unlock()

	assert.Equal(t, now.Add(clusterChallengePause), pausedUntil)

	require.NoError(t, first.CleanUp(t.Context(), "app.example.com", testChallengeToken, testChallengeKeyAuth))
	assert.Equal(t, http.StatusNotFound, askChallenge(t, first, "app.example.com", testChallengeToken).Code)
}

func TestClusterChallengeHTTP_serverBack(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestClusterChallenge(t, server, "srv1")
	second := newTestClusterChallenge(t, server, "srv2")

	now := time.Now()
	second.now = func() time.Time { return now }

	require.NoError(t, first.Present(t.Context(), "app.example.com", testChallengeToken, testChallengeKeyAuth))

	second.pause()

	assert.Equal(t, http.StatusNotFound, askChallenge(t, second, "app.example.com", testChallengeToken).Code)

	now = now.Add(clusterChallengePause)

	assert.Equal(t, http.StatusOK, askChallenge(t, second, "app.example.com", testChallengeToken).Code)
}
