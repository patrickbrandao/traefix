package ratelimiter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"golang.org/x/time/rate"
)

const redisPrefix = "rate:"

// sharedRedisClients holds one client per distinct Redis configuration.
// A middleware is built once per router and again on every configuration reload, and since go-redis v9.12
// each client starts a goroutine at construction that keeps it reachable, so a client per instance leaks.
// Sharing changes no semantics, as the buckets live in Redis.
var sharedRedisClients = struct {
	mu      sync.Mutex
	clients map[string]redis.UniversalClient
}{clients: make(map[string]redis.UniversalClient)}

// sharedRedisClient returns the client cached for the Redis configuration, creating it on first use.
func sharedRedisClient(config *dynamic.Redis, options *redis.UniversalOptions) (redis.UniversalClient, error) {
	key, err := redisClientKey(config)
	if err != nil {
		return nil, err
	}

	sharedRedisClients.mu.Lock()
	defer sharedRedisClients.mu.Unlock()

	if client, ok := sharedRedisClients.clients[key]; ok {
		return client, nil
	}

	client := redis.NewUniversalClient(options)
	sharedRedisClients.clients[key] = client

	return client, nil
}

// redisClientKey derives the cache key from the configuration values, and not from redis.UniversalOptions,
// whose TLS config and function pointers differ on every construction.
// The key is a hash so that the cache does not keep another copy of the credentials.
func redisClientKey(config *dynamic.Redis) (string, error) {
	raw, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshaling redis config for client cache key: %w", err)
	}

	h := sha256.New()
	h.Write(raw)

	// A certificate rotated at an unchanged path must give a new client,
	// otherwise the cache would keep using the previous one.
	if config.TLS != nil {
		for _, v := range []string{config.TLS.CA, config.TLS.Cert, config.TLS.Key} {
			h.Write([]byte{0})
			if v == "" {
				continue
			}
			if content, err := os.ReadFile(v); err == nil {
				h.Write(content)
			}
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

type redisLimiter struct {
	rate     rate.Limit // reqs/s
	burst    int64
	maxDelay time.Duration
	period   ptypes.Duration
	logger   *zerolog.Logger
	ttl      int
	client   Rediser
	script   *redis.Script
}

func newRedisLimiter(ctx context.Context, rate rate.Limit, burst int64, maxDelay time.Duration, ttl int, config dynamic.RateLimit, logger *zerolog.Logger) (limiter, error) {
	options := &redis.UniversalOptions{
		Addrs:            config.Redis.Endpoints,
		Username:         config.Redis.Username,
		Password:         config.Redis.Password,
		DB:               config.Redis.DB,
		PoolSize:         config.Redis.PoolSize,
		MinIdleConns:     config.Redis.MinIdleConns,
		MaxActiveConns:   config.Redis.MaxActiveConns,
		MasterName:       config.Redis.MasterName,
		SentinelUsername: config.Redis.SentinelUsername,
		SentinelPassword: config.Redis.SentinelPassword,
	}

	if config.Redis.DialTimeout != nil && *config.Redis.DialTimeout > 0 {
		options.DialTimeout = time.Duration(*config.Redis.DialTimeout)
	}

	if config.Redis.ReadTimeout != nil {
		if *config.Redis.ReadTimeout > 0 {
			options.ReadTimeout = time.Duration(*config.Redis.ReadTimeout)
		} else {
			options.ReadTimeout = -1
		}
	}

	if config.Redis.WriteTimeout != nil {
		if *config.Redis.WriteTimeout > 0 {
			options.WriteTimeout = time.Duration(*config.Redis.WriteTimeout)
		} else {
			options.WriteTimeout = -1
		}
	}

	if config.Redis.TLS != nil {
		var err error
		options.TLSConfig, err = config.Redis.TLS.CreateTLSConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("creating TLS config: %w", err)
		}
	}

	script, err := LoadTokenBucketScript()
	if err != nil {
		return nil, fmt.Errorf("loading bucket script: %w", err)
	}

	client, err := sharedRedisClient(config.Redis, options)
	if err != nil {
		return nil, fmt.Errorf("obtaining redis client: %w", err)
	}

	return &redisLimiter{
		rate:     rate,
		burst:    burst,
		period:   config.Period,
		maxDelay: maxDelay,
		logger:   logger,
		ttl:      ttl,
		client:   client,
		script:   script,
	}, nil
}

func (r *redisLimiter) Allow(ctx context.Context, source string) (*time.Duration, error) {
	ok, delay, err := r.evaluateScript(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("evaluating script: %w", err)
	}
	if !ok {
		return nil, nil
	}
	return delay, nil
}

func (r *redisLimiter) evaluateScript(ctx context.Context, key string) (bool, *time.Duration, error) {
	if r.rate == rate.Inf {
		// A nil delay would reject the request, while an infinite rate must allow all of them.
		var noDelay time.Duration
		return true, &noDelay, nil
	}

	params := []any{
		float64(r.rate / 1000000),
		r.burst,
		r.ttl,
		time.Now().UnixMicro(),
		r.maxDelay.Microseconds(),
	}
	v, err := r.script.Run(ctx, r.client, []string{redisPrefix + key}, params...).Result()
	if err != nil {
		return false, nil, fmt.Errorf("running script: %w", err)
	}

	values := v.([]any)
	ok, err := strconv.ParseBool(values[0].(string))
	if err != nil {
		return false, nil, fmt.Errorf("parsing ok value from redis rate lua script: %w", err)
	}
	delay, err := strconv.ParseFloat(values[1].(string), 64)
	if err != nil {
		return false, nil, fmt.Errorf("parsing delay value from redis rate lua script: %w", err)
	}

	microDelay := time.Duration(delay * float64(time.Microsecond))
	return ok, &microDelay, nil
}
