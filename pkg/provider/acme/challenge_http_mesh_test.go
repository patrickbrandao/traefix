package acme

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
)

const meshTestToken = "LoqXcYV8q5ONbJQxbmR7SCTNo3tiAXDfowyjxAjEuX0"

// newMeshTestChallenge returns the challenge of an instance of a cluster that only has the mesh.
func newMeshTestChallenge(t *testing.T) *ClusterChallengeHTTP {
	t.Helper()

	conf := &cluster.Configuration{Mesh: &cluster.Mesh{}}
	conf.SetDefaults()
	conf.Mesh.SetDefaults()
	conf.NodeID = "srv1"
	conf.Token = "0123456789abcdef0123456789abcdef"

	member, err := cluster.New(t.Context(), conf)
	require.NoError(t, err)
	require.False(t, member.HasServer())

	solver, ok := NewHTTPChallengeSolver(member).(*ClusterChallengeHTTP)
	require.True(t, ok)

	return solver
}

func TestClusterChallengeHTTP_mesh(t *testing.T) {
	solver := newMeshTestChallenge(t)

	// Without the mesh yet, the instance only answers what it presented.
	require.NoError(t, solver.Present(t.Context(), "local.example.com", meshTestToken, "local-answer"))

	assert.Equal(t, "local-answer", askChallenge(t, solver, "local.example.com", meshTestToken).Body.String())
	assert.Equal(t, http.StatusNotFound, askChallenge(t, solver, "app.example.com", meshTestToken).Code)
	assert.Equal(t, []byte("local-answer"), solver.LocalAnswer(t.Context(), meshTestToken, "local.example.com"))

	var asked atomic.Int32

	solver.SetMeshLookup(func(_ context.Context, token, domain string) ([]byte, error) {
		asked.Add(1)

		if token == meshTestToken && domain == "app.example.com" {
			return []byte("remote-answer"), nil
		}

		return nil, nil
	})

	// The challenge another instance presented is looked up from the mesh, with the name as the CA wrote it.
	recorder := askChallenge(t, solver, "App.Example.com:80", meshTestToken)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "remote-answer", recorder.Body.String())

	// What no instance knows is remembered for a short time: strangers do not make the instance ask every time.
	before := asked.Load()

	assert.Equal(t, http.StatusNotFound, askChallenge(t, solver, "unknown.example.com", meshTestToken).Code)
	assert.Equal(t, http.StatusNotFound, askChallenge(t, solver, "unknown.example.com", meshTestToken).Code)
	assert.Equal(t, before+1, asked.Load())

	// What cannot be a challenge is not looked up.
	assert.Equal(t, http.StatusNotFound, askChallenge(t, solver, "app.example.com", "short").Code)
	assert.Equal(t, before+1, asked.Load())
}

func TestClusterChallengeHTTP_meshUnreachable(t *testing.T) {
	solver := newMeshTestChallenge(t)

	var asked atomic.Int32

	solver.SetMeshLookup(func(context.Context, string, string) ([]byte, error) {
		asked.Add(1)
		return nil, errors.New("srv2: the other instance does not answer")
	})

	// An instance that does not answer is not a reason to remember the token as unknown, nor to stop asking:
	// the next request of the CA asks again.
	assert.Equal(t, http.StatusNotFound, askChallenge(t, solver, "app.example.com", meshTestToken).Code)
	assert.Equal(t, http.StatusNotFound, askChallenge(t, solver, "app.example.com", meshTestToken).Code)
	assert.Equal(t, int32(2), asked.Load())
}
