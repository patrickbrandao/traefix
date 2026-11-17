package ratelimiter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/types"
	"golang.org/x/time/rate"
)

func buildRedisLimiter(t *testing.T, config *dynamic.Redis) *redisLimiter {
	t.Helper()

	logger := zerolog.Nop()
	l, err := newRedisLimiter(t.Context(), rate.Limit(10), 5, time.Second, 60, dynamic.RateLimit{Redis: config}, &logger)
	require.NoError(t, err)

	rl, ok := l.(*redisLimiter)
	require.True(t, ok)

	return rl
}

func TestRedisLimiter_sharedClientPerConfig(t *testing.T) {
	first := buildRedisLimiter(t, &dynamic.Redis{Endpoints: []string{"127.0.0.1:6379"}, DB: 3})

	for range 20 {
		l := buildRedisLimiter(t, &dynamic.Redis{Endpoints: []string{"127.0.0.1:6379"}, DB: 3})
		assert.Same(t, first.client, l.client)
	}

	other := buildRedisLimiter(t, &dynamic.Redis{Endpoints: []string{"127.0.0.1:6379"}, DB: 4})
	assert.NotSame(t, first.client, other.client)
}

func TestRedisClientKey(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(caPath, []byte("generation-1"), 0o600))

	config := &dynamic.Redis{TLS: &types.ClientTLS{CA: caPath}}
	before, err := redisClientKey(config)
	require.NoError(t, err)

	same, err := redisClientKey(&dynamic.Redis{TLS: &types.ClientTLS{CA: caPath}})
	require.NoError(t, err)
	assert.Equal(t, before, same)

	require.NoError(t, os.WriteFile(caPath, []byte("generation-2"), 0o600))
	after, err := redisClientKey(config)
	require.NoError(t, err)
	assert.NotEqual(t, before, after)

	insecure, err := redisClientKey(&dynamic.Redis{TLS: &types.ClientTLS{CA: caPath, InsecureSkipVerify: true}})
	require.NoError(t, err)
	assert.NotEqual(t, after, insecure)

	password, err := redisClientKey(&dynamic.Redis{Password: "secret"})
	require.NoError(t, err)
	assert.NotContains(t, password, "secret")
}

func TestRedisLimiter_clientClosedWhenUnused(t *testing.T) {
	config := &dynamic.Redis{Endpoints: []string{"127.0.0.1:0"}, DB: 5}

	kept := buildRedisLimiter(t, config)
	client := func() redis.UniversalClient {
		return buildRedisLimiter(t, config).client.(redis.UniversalClient)
	}()

	key, err := redisClientKey(config)
	require.NoError(t, err)

	// The limiter still in use keeps the shared client open.
	require.Eventually(t, func() bool {
		runtime.GC()

		sharedRedisClients.mu.Lock()
		defer sharedRedisClients.mu.Unlock()

		return sharedRedisClients.clients[key].refs == 1
	}, 5*time.Second, 10*time.Millisecond)
	assert.NotErrorIs(t, client.Ping(context.Background()).Err(), redis.ErrClosed)

	runtime.KeepAlive(kept)

	require.Eventually(t, func() bool {
		runtime.GC()
		return errors.Is(client.Ping(context.Background()).Err(), redis.ErrClosed)
	}, 5*time.Second, 10*time.Millisecond)

	sharedRedisClients.mu.Lock()
	defer sharedRedisClients.mu.Unlock()

	assert.NotContains(t, sharedRedisClients.clients, key)
}
