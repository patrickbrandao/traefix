package mesh

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// answering makes the instance know the answer of one challenge.
func answering(token, domain, value string) ChallengeAnswer {
	return func(_ context.Context, t, d string) []byte {
		if t == token && d == domain {
			return []byte(value)
		}

		return nil
	}
}

func TestMesh_LookupChallenge(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv2", testToken, nil, first.address)
	third := newTestNode(t, "srv3", testToken, nil, first.address)

	first.waitLinked("srv2", "srv3")

	// Only the third instance presented the challenge.
	second.mesh.SetChallengeAnswer(answering("token-b", "b.example.com", "nope"))
	third.mesh.SetChallengeAnswer(answering("token-a", "app.example.com", "answer-a"))

	value, err := first.mesh.LookupChallenge(t.Context(), "token-a", "app.example.com")
	require.NoError(t, err)
	assert.Equal(t, "answer-a", string(value))

	// No instance knows it: nothing, and no error.
	value, err = first.mesh.LookupChallenge(t.Context(), "token-x", "app.example.com")
	require.NoError(t, err)
	assert.Empty(t, value)

	// An instance asked does not ask further: the third one does not know what the second one presented.
	value, err = third.mesh.LookupChallenge(t.Context(), "token-a", "app.example.com")
	require.NoError(t, err)
	assert.Empty(t, value)

	// An instance that does not answer in time is an error, while the others say they do not know.
	ctx, cancel := context.WithTimeout(t.Context(), time.Nanosecond)
	defer cancel()

	_, err = first.mesh.LookupChallenge(ctx, "token-x", "app.example.com")
	require.Error(t, err)
}

func TestMesh_LookupChallenge_alone(t *testing.T) {
	node := newTestNode(t, "srv1", testToken, nil)

	value, err := node.mesh.LookupChallenge(t.Context(), "token-a", "app.example.com")
	require.NoError(t, err)
	assert.Empty(t, value)
}
