// Package cluster joins the Traefix instances that share a Valkey or Redis server:
// each instance announces itself, and they coordinate through locks and events.
package cluster

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v3/pkg/types"
)

// Configuration joins this instance to the other instances of a cluster (Traefix):
// the ones that use the same Redis server and prefix, and the ones it is linked to by the mesh.
type Configuration struct {
	NodeID            string          `description:"Identifier of this instance, unique among the instances of the cluster. Defaults to the host name." json:"nodeID,omitempty" toml:"nodeID,omitempty" yaml:"nodeID,omitempty" export:"true"`
	Priority          int             `description:"Priority of this instance to obtain the ACME certificates: the highest one wins, and 0 never obtains a certificate." json:"priority" toml:"priority" yaml:"priority" export:"true"`
	Prefix            string          `description:"Prefix of the keys and channels of the cluster in the Redis server." json:"prefix,omitempty" toml:"prefix,omitempty" yaml:"prefix,omitempty" export:"true"`
	HeartbeatInterval ptypes.Duration `description:"Interval between two announcements of this instance." json:"heartbeatInterval,omitempty" toml:"heartbeatInterval,omitempty" yaml:"heartbeatInterval,omitempty" export:"true"`
	NodeTimeout       ptypes.Duration `description:"Time without announcement after which the other instances consider this instance gone." json:"nodeTimeout,omitempty" toml:"nodeTimeout,omitempty" yaml:"nodeTimeout,omitempty" export:"true"`
	Redis             *Redis          `description:"Valkey or Redis server shared by the instances of the cluster." json:"redis,omitempty" toml:"redis,omitempty" yaml:"redis,omitempty" export:"true"`

	Token            string   `description:"Secret shared by the instances of the cluster, which proves to an instance that another one belongs to the cluster." json:"token,omitempty" toml:"token,omitempty" yaml:"token,omitempty" loggable:"false"`
	TokenFile        string   `description:"File that holds the token of the cluster." json:"tokenFile,omitempty" toml:"tokenFile,omitempty" yaml:"tokenFile,omitempty" export:"true"`
	AdditionalTokens []string `description:"Tokens accepted from the other instances in addition to the token: the next token before a change, the previous one after it." json:"additionalTokens,omitempty" toml:"additionalTokens,omitempty" yaml:"additionalTokens,omitempty" loggable:"false"`
	Mesh             *Mesh    `description:"Links this instance to the other instances of the cluster, to reach the services they discovered." json:"mesh,omitempty" toml:"mesh,omitempty" yaml:"mesh,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// SetDefaults sets the default values.
func (c *Configuration) SetDefaults() {
	c.Priority = 100
	c.Prefix = "traefix"
	c.HeartbeatInterval = ptypes.Duration(5 * time.Second)
	c.NodeTimeout = ptypes.Duration(15 * time.Second)
}

// Redis is the connection to a Valkey or Redis server: a single instance, a Redis Cluster, or the master elected by Sentinel.
type Redis struct {
	Endpoints        []string         `description:"Address of the server, of the nodes of a Redis Cluster, or of the Sentinels when masterName is set." json:"endpoints,omitempty" toml:"endpoints,omitempty" yaml:"endpoints,omitempty"`
	Username         string           `description:"Username to connect to the server." json:"username,omitempty" toml:"username,omitempty" yaml:"username,omitempty" loggable:"false"`
	Password         string           `description:"Password to connect to the server." json:"password,omitempty" toml:"password,omitempty" yaml:"password,omitempty" loggable:"false"`
	DB               int              `description:"Database selected after connecting to the server." json:"db,omitempty" toml:"db,omitempty" yaml:"db,omitempty"`
	TLS              *types.ClientTLS `description:"TLS configuration of the connection to the server." json:"tls,omitempty" toml:"tls,omitempty" yaml:"tls,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	MasterName       string           `description:"Name of the Sentinel master set. When set, the endpoints are the Sentinels." json:"masterName,omitempty" toml:"masterName,omitempty" yaml:"masterName,omitempty" export:"true"`
	SentinelUsername string           `description:"Username to connect to the Sentinels." json:"sentinelUsername,omitempty" toml:"sentinelUsername,omitempty" yaml:"sentinelUsername,omitempty" loggable:"false"`
	SentinelPassword string           `description:"Password to connect to the Sentinels." json:"sentinelPassword,omitempty" toml:"sentinelPassword,omitempty" yaml:"sentinelPassword,omitempty" loggable:"false"`
	DialTimeout      ptypes.Duration  `description:"Timeout to establish a connection." json:"dialTimeout,omitempty" toml:"dialTimeout,omitempty" yaml:"dialTimeout,omitempty" export:"true"`
	ReadTimeout      ptypes.Duration  `description:"Timeout of a read on a connection." json:"readTimeout,omitempty" toml:"readTimeout,omitempty" yaml:"readTimeout,omitempty" export:"true"`
	WriteTimeout     ptypes.Duration  `description:"Timeout of a write on a connection." json:"writeTimeout,omitempty" toml:"writeTimeout,omitempty" yaml:"writeTimeout,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (r *Redis) SetDefaults() {
	r.Endpoints = []string{"localhost:6379"}
	r.DialTimeout = ptypes.Duration(5 * time.Second)
	r.ReadTimeout = ptypes.Duration(3 * time.Second)
	r.WriteTimeout = ptypes.Duration(3 * time.Second)
}

// nameRegexp keeps the names usable inside a key, a lock value and an event, which separate their parts with ':', '|' and ' '.
var nameRegexp = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Validate checks that the configuration is coherent.
func (c *Configuration) Validate() error {
	if c == nil {
		return nil
	}

	if c.NodeID != "" && !nameRegexp.MatchString(c.NodeID) {
		return fmt.Errorf("cluster nodeID %q must start with a letter or a digit, followed by letters, digits, dots, underscores or hyphens", c.NodeID)
	}

	if !nameRegexp.MatchString(c.Prefix) {
		return fmt.Errorf("cluster prefix %q must start with a letter or a digit, followed by letters, digits, dots, underscores or hyphens", c.Prefix)
	}

	if c.Priority < 0 {
		return errors.New("cluster priority cannot be negative")
	}

	if c.HeartbeatInterval <= 0 {
		return errors.New("cluster heartbeatInterval must be positive")
	}

	// One lost announcement must not make the instance look gone.
	if c.NodeTimeout < 2*c.HeartbeatInterval {
		return errors.New("cluster nodeTimeout must be at least twice the heartbeatInterval")
	}

	if c.Redis == nil && c.Mesh == nil {
		return errors.New("cluster needs the redis option, the mesh option, or both")
	}

	if c.Redis != nil && len(c.Redis.Endpoints) == 0 {
		return errors.New("cluster redis endpoints are required")
	}

	if _, _, err := c.Tokens(); err != nil {
		return err
	}

	if c.Mesh != nil {
		if c.Mesh.Satellite && c.Redis != nil {
			return errors.New("a cluster mesh satellite does not use the Redis server: remove the cluster redis option")
		}

		if c.Token == "" && c.TokenFile == "" {
			return errors.New("cluster mesh needs the cluster token or tokenFile")
		}

		if err := c.Mesh.validate(); err != nil {
			return err
		}
	}

	return nil
}

// minTokenLength keeps the token out of reach of a search:
// the instances derive a signing key from it, and anyone who sees a certificate of the cluster can test candidates offline.
const minTokenLength = 32

// Tokens returns the token of the cluster and the other tokens accepted from the instances.
// The token is empty when none is configured.
func (c *Configuration) Tokens() (string, []string, error) {
	if c.Token != "" && c.TokenFile != "" {
		return "", nil, errors.New("cluster token and tokenFile cannot be set together")
	}

	token := c.Token

	if c.TokenFile != "" {
		data, err := os.ReadFile(c.TokenFile)
		if err != nil {
			return "", nil, fmt.Errorf("reading the cluster tokenFile: %w", err)
		}

		// A file written by hand or by a secret manager often ends with a new line.
		token = strings.TrimSpace(string(data))
		if token == "" {
			return "", nil, fmt.Errorf("the cluster tokenFile %q is empty", c.TokenFile)
		}
	}

	if token == "" {
		if len(c.AdditionalTokens) > 0 {
			return "", nil, errors.New("cluster additionalTokens needs the cluster token or tokenFile")
		}

		return "", nil, nil
	}

	if len(token) < minTokenLength {
		return "", nil, fmt.Errorf("the cluster token must have at least %d characters", minTokenLength)
	}

	var additional []string
	for _, other := range c.AdditionalTokens {
		if len(other) < minTokenLength {
			return "", nil, fmt.Errorf("each of the cluster additionalTokens must have at least %d characters", minTokenLength)
		}

		if other != token && !slices.Contains(additional, other) {
			additional = append(additional, other)
		}
	}

	return token, additional, nil
}

// resolveNodeID returns the configured identifier of the instance, or one made from the host name.
func (c *Configuration) resolveNodeID() (string, error) {
	if c.NodeID != "" {
		return c.NodeID, nil
	}

	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("cluster nodeID is not set and the host name is not available: %w", err)
	}

	if !nameRegexp.MatchString(hostname) {
		return "", fmt.Errorf("cluster nodeID is not set and the host name %q cannot be used as one", hostname)
	}

	return hostname, nil
}

// redisLogger sends the messages of the Redis client to the logs of Traefix.
// The client writes one line for each failed connection: at the debug level, an unreachable server does not flood the logs,
// and the instance reports the loss and the return of the server once.
type redisLogger struct{}

func (redisLogger) Printf(_ context.Context, format string, v ...any) {
	log.Debug().Str("component", "redis").Msgf(format, v...)
}

var setRedisLogger sync.Once

// newClient creates the client of the single instance, of the Redis Cluster or of the Sentinel master.
func (r *Redis) newClient(ctx context.Context) (redis.UniversalClient, error) {
	// The logger of the client is shared by every client of the process.
	setRedisLogger.Do(func() { redis.SetLogger(redisLogger{}) })

	options := &redis.UniversalOptions{
		Addrs:            r.Endpoints,
		Username:         r.Username,
		Password:         r.Password,
		DB:               r.DB,
		MasterName:       r.MasterName,
		SentinelUsername: r.SentinelUsername,
		SentinelPassword: r.SentinelPassword,
		DialTimeout:      time.Duration(r.DialTimeout),
		ReadTimeout:      time.Duration(r.ReadTimeout),
		WriteTimeout:     time.Duration(r.WriteTimeout),
	}

	if r.TLS != nil {
		var err error
		options.TLSConfig, err = r.TLS.CreateTLSConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("creating the TLS configuration of the cluster Redis client: %w", err)
		}
	}

	return redis.NewUniversalClient(options), nil
}
