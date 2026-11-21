package acme

import (
	"context"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v5/challenge"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
)

const (
	// clusterChallengeTTL removes the answer of a challenge whose cleanup was lost, for instance when the instance dies.
	// The CA validates within seconds of the presentation, and the provider gives up after one minute.
	clusterChallengeTTL = 10 * time.Minute

	// clusterChallengeLookupTimeout bounds what a request of the CA, or of a stranger, waits for the server.
	clusterChallengeLookupTimeout = 500 * time.Millisecond

	// clusterChallengeWriteTimeout bounds the presentation and the cleanup, which run inside a certificate request.
	clusterChallengeWriteTimeout = 3 * time.Second

	// clusterChallengeMissTTL is how long an unknown token is answered without asking the server again.
	clusterChallengeMissTTL = 5 * time.Second

	// clusterChallengeMaxMisses bounds the memory that strangers can make the instance use with unknown tokens.
	clusterChallengeMaxMisses = 4096

	// clusterChallengeMaxLookups bounds the connections to the server that the requests of an entry point can take.
	clusterChallengeMaxLookups = 32

	// clusterChallengePause is how long the instance stops asking a server that just failed,
	// so that the requests do not pile up waiting for it.
	clusterChallengePause = time.Second
)

// clusterTokenRegexp is the shape of an ACME token (RFC 8555, section 8.3): base64url, with at least 128 bits of entropy.
// The path is public: anything else is not worth a request to the server.
var clusterTokenRegexp = regexp.MustCompile(`^[A-Za-z0-9_-]{22,128}$`)

// clusterDomainRegexp bounds the Host header that ends in a key: a DNS name or an IP address.
var clusterDomainRegexp = regexp.MustCompile(`^[a-z0-9._:-]{1,253}$`)

// HTTPChallengeSolver presents the HTTP-01 challenges and answers the requests of the CA (Traefix).
type HTTPChallengeSolver interface {
	challenge.Provider
	http.Handler
}

// NewHTTPChallengeSolver returns the HTTP-01 challenge of the instance, shared with the cluster when the instance is in one.
func NewHTTPChallengeSolver(member *cluster.Cluster) HTTPChallengeSolver {
	if member == nil || !member.HasServer() {
		return NewChallengeHTTP()
	}

	return NewClusterChallengeHTTP(member)
}

// ClusterChallengeHTTP shares the HTTP-01 challenges through the Redis server of the cluster:
// the CA reaches any instance behind the name, not only the one that requests the certificate.
type ClusterChallengeHTTP struct {
	local  *ChallengeHTTP
	member *cluster.Cluster

	lookups chan struct{}
	now     func() time.Time

	mu          sync.Mutex
	misses      map[string]time.Time
	pausedUntil time.Time
}

// NewClusterChallengeHTTP creates a ClusterChallengeHTTP.
func NewClusterChallengeHTTP(member *cluster.Cluster) *ClusterChallengeHTTP {
	return &ClusterChallengeHTTP{
		local:   NewChallengeHTTP(),
		member:  member,
		lookups: make(chan struct{}, clusterChallengeMaxLookups),
		now:     time.Now,
		misses:  map[string]time.Time{},
	}
}

// Present keeps the answer of a challenge in this instance and in the server of the cluster.
func (c *ClusterChallengeHTTP) Present(ctx context.Context, domain, token, keyAuth string) error {
	if err := c.local.Present(ctx, domain, token, keyAuth); err != nil {
		return err
	}

	logger := log.Ctx(ctx).With().Str(logs.ProviderName, "acme").Str("domain", domain).Logger()

	key, ok := c.key(token, domain)
	if !ok {
		logger.Warn().Msg("The HTTP-01 challenge cannot be shared with the cluster, only this instance answers it")
		return nil
	}

	writeCtx, cancel := context.WithTimeout(ctx, clusterChallengeWriteTimeout)
	defer cancel()

	// Without the server, the request goes on: the instance still answers the challenge when the CA reaches it,
	// which is always the case for an instance alone behind its names.
	if err := c.member.Client().Set(writeCtx, key, keyAuth, clusterChallengeTTL).Err(); err != nil {
		logger.Warn().Err(err).Msg("Unable to share the HTTP-01 challenge with the cluster, only this instance answers it")
	}

	c.mu.Lock()
	delete(c.misses, key)
	c.mu.Unlock()

	return nil
}

// CleanUp removes the answer of a challenge from this instance and from the server of the cluster.
func (c *ClusterChallengeHTTP) CleanUp(ctx context.Context, domain, token, keyAuth string) error {
	if err := c.local.CleanUp(ctx, domain, token, keyAuth); err != nil {
		return err
	}

	key, ok := c.key(token, domain)
	if !ok {
		return nil
	}

	// The cleanup also runs after a request that was canceled.
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), clusterChallengeWriteTimeout)
	defer cancel()

	if err := c.member.Client().Del(writeCtx, key).Err(); err != nil {
		log.Ctx(ctx).Debug().Err(err).Str(logs.ProviderName, "acme").Str("domain", domain).
			Msg("Unable to remove the HTTP-01 challenge from the cluster, it expires by itself")
	}

	return nil
}

// Timeout calculates the maximum of time allowed to resolved an ACME challenge.
func (c *ClusterChallengeHTTP) Timeout() (timeout, interval time.Duration) {
	return c.local.Timeout()
}

func (c *ClusterChallengeHTTP) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	logger := log.Ctx(req.Context()).With().Str(logs.ProviderName, "acme").Logger()
	ctx := logger.WithContext(req.Context())

	token, err := getPathParam(req.URL)
	if err != nil {
		logger.Debug().Err(err).Msg("Unable to get token")
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	domain, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		domain = req.Host
	}

	result := lookupLocal

	value := c.local.getTokenValue(ctx, token, domain)
	if len(value) == 0 {
		value, result = c.lookup(ctx, token, domain)
	}

	countChallengeLookup(result)

	if len(value) == 0 {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	rw.WriteHeader(http.StatusOK)

	if _, err := rw.Write(value); err != nil {
		logger.Error().Err(err).Msg("Unable to write token")
	}
}

// key returns the key of the answer of a challenge, and false when the token or the domain cannot be one of a challenge.
func (c *ClusterChallengeHTTP) key(token, domain string) (string, bool) {
	// An IPv6 address comes with brackets from a Host header without port.
	domain = strings.ToLower(strings.Trim(domain, "[]"))

	if !clusterTokenRegexp.MatchString(token) || !clusterDomainRegexp.MatchString(domain) {
		return "", false
	}

	return c.member.Key("acme", "http", token, domain), true
}

// lookup asks the server of the cluster for the answer of a challenge presented by another instance.
func (c *ClusterChallengeHTTP) lookup(ctx context.Context, token, domain string) ([]byte, string) {
	key, ok := c.key(token, domain)
	if !ok {
		return nil, lookupInvalid
	}

	if c.skip(key) {
		return nil, lookupMiss
	}

	lookupCtx, cancel := context.WithTimeout(ctx, clusterChallengeLookupTimeout)
	defer cancel()

	select {
	case c.lookups <- struct{}{}:
		defer func() { <-c.lookups }()
	case <-lookupCtx.Done():
		return nil, lookupError
	}

	value, err := c.member.Client().Get(lookupCtx, key).Bytes()

	switch {
	case errors.Is(err, redis.Nil):
		c.missed(key)
		return nil, lookupMiss

	case err != nil:
		// A client that goes away says nothing about the server.
		if ctx.Err() == nil {
			c.pause()
		}

		log.Ctx(ctx).Debug().Err(err).Msgf("Unable to ask the cluster for the ACME challenge for %s", domain)

		return nil, lookupError
	}

	log.Ctx(ctx).Debug().Msgf("Answering the ACME challenge for %s presented by another instance", domain)

	return value, lookupRedis
}

// skip tells whether the server is not asked: it just failed, or it just said that it does not know this challenge.
func (c *ClusterChallengeHTTP) skip(key string) bool {
	now := c.now()

	c.mu.Lock()
	defer c.mu.Unlock()

	if now.Before(c.pausedUntil) {
		return true
	}

	until, ok := c.misses[key]
	if !ok {
		return false
	}

	if now.Before(until) {
		return true
	}

	delete(c.misses, key)

	return false
}

// missed remembers for a short time that the server does not know a challenge.
func (c *ClusterChallengeHTTP) missed(key string) {
	now := c.now()

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.misses) >= clusterChallengeMaxMisses {
		for k, until := range c.misses {
			if !now.Before(until) {
				delete(c.misses, k)
			}
		}
	}

	// When the memory is full of fresh misses, the next ones are not remembered: the limit of lookups still holds.
	if len(c.misses) < clusterChallengeMaxMisses {
		c.misses[key] = now.Add(clusterChallengeMissTTL)
	}
}

// pause stops the lookups for a moment after a failure of the server.
func (c *ClusterChallengeHTTP) pause() {
	now := c.now()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.pausedUntil = now.Add(clusterChallengePause)
}
