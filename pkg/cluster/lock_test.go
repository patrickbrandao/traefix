package cluster

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertLost(t *testing.T, lock *Lock, lost bool) {
	t.Helper()

	select {
	case <-lock.Lost():
		assert.True(t, lost)
	default:
		assert.False(t, lost)
	}
}

func TestCluster_TryLock(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestCluster(t, server, "srv1", 100)
	second := newTestCluster(t, server, "srv2", 100)

	_, held, err := first.LockHolder(t.Context(), "acme:le:abc")
	require.NoError(t, err)
	assert.False(t, held)

	lock, ok, err := first.TryLock(t.Context(), "acme:le:abc", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)

	_, ok, err = second.TryLock(t.Context(), "acme:le:abc", time.Minute)
	require.NoError(t, err)
	assert.False(t, ok)

	// A lock on another name is free.
	other, ok, err := second.TryLock(t.Context(), "acme:le:def", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, other.Release(t.Context()))

	holder, held, err := second.LockHolder(t.Context(), "acme:le:abc")
	require.NoError(t, err)
	assert.True(t, held)
	assert.Equal(t, "srv1", holder)

	require.NoError(t, lock.Release(t.Context()))

	_, ok, err = second.TryLock(t.Context(), "acme:le:abc", time.Minute)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestLock_expires(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestCluster(t, server, "srv1", 100)
	second := newTestCluster(t, server, "srv2", 100)

	lock, ok, err := first.TryLock(t.Context(), "name", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)

	server.FastForward(40 * time.Second)

	held, err := lock.Extend(t.Context())
	require.NoError(t, err)
	assert.True(t, held)
	assertLost(t, lock, false)

	// The holder stops extending the lock, as after a crash.
	server.FastForward(61 * time.Second)

	successor, ok, err := second.TryLock(t.Context(), "name", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)

	held, err = lock.Extend(t.Context())
	require.NoError(t, err)
	assert.False(t, held)
	assertLost(t, lock, true)

	// The late holder frees nothing: the lock belongs to its successor.
	require.NoError(t, lock.Release(t.Context()))

	holder, held, err := first.LockHolder(t.Context(), "name")
	require.NoError(t, err)
	assert.True(t, held)
	assert.Equal(t, "srv2", holder)

	held, err = successor.Extend(t.Context())
	require.NoError(t, err)
	assert.True(t, held)
}

func TestLock_Keep(t *testing.T) {
	server := miniredis.RunT(t)

	c := newTestCluster(t, server, "srv1", 100)

	lock, ok, err := c.TryLock(t.Context(), "name", 150*time.Millisecond)
	require.NoError(t, err)
	require.True(t, ok)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		lock.Keep(ctx)
	}()

	server.FastForward(100 * time.Millisecond)
	require.Eventually(t, func() bool { return server.TTL("traefix:lock:name") == 150*time.Millisecond }, 5*time.Second, 5*time.Millisecond)

	cancel()
	<-done
	assertLost(t, lock, false)
}

func TestLock_Keep_stopsWhenTheLockIsLost(t *testing.T) {
	server := miniredis.RunT(t)

	c := newTestCluster(t, server, "srv1", 100)

	lock, ok, err := c.TryLock(t.Context(), "name", 60*time.Millisecond)
	require.NoError(t, err)
	require.True(t, ok)

	done := make(chan struct{})
	go func() {
		defer close(done)
		lock.Keep(t.Context())
	}()

	server.Del("traefix:lock:name")

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "Keep did not stop")
	}

	assertLost(t, lock, true)
}

// An unreachable server does not tell that another instance took the lock.
func TestLock_Keep_goesOnWhenTheServerIsUnreachable(t *testing.T) {
	server := miniredis.RunT(t)

	c := newTestCluster(t, server, "srv1", 100)

	lock, ok, err := c.TryLock(t.Context(), "name", 60*time.Millisecond)
	require.NoError(t, err)
	require.True(t, ok)

	server.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	lock.Keep(ctx)

	assertLost(t, lock, false)

	_, err = lock.Extend(t.Context())
	require.Error(t, err)
}
