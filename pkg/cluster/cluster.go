package cluster

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/version"
)

// leaveTimeout bounds the farewell of an instance that stops, which must not delay the shutdown.
const leaveTimeout = time.Second

// Node is what an instance announces to the others.
type Node struct {
	ID       string `json:"id"`
	Priority int    `json:"priority"`
	Version  string `json:"version"`
	// Instance changes every time the process starts, which tells a restart from a second instance with the same ID.
	Instance  string    `json:"instance"`
	StartedAt time.Time `json:"startedAt"`
	// Attributes are published by the features that need the other instances to know something about this one.
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Cluster is the membership of this instance in a cluster.
type Cluster struct {
	conf   *Configuration
	client redis.UniversalClient
	logger zerolog.Logger

	mu        sync.RWMutex
	self      Node
	up        bool
	announced bool
	handlers  map[string][]func(Event)
	onConnect []func()
}

// New creates the membership of this instance. Nothing is sent to the server before Run.
func New(ctx context.Context, conf *Configuration) (*Cluster, error) {
	nodeID, err := conf.resolveNodeID()
	if err != nil {
		return nil, err
	}

	client, err := conf.Redis.newClient(ctx)
	if err != nil {
		return nil, err
	}

	instance := make([]byte, 8)
	if _, err := rand.Read(instance); err != nil {
		return nil, err
	}

	return &Cluster{
		conf:   conf,
		client: client,
		logger: log.With().Str("cluster", conf.Prefix).Str("nodeID", nodeID).Logger(),
		self: Node{
			ID:        nodeID,
			Priority:  conf.Priority,
			Version:   version.Version,
			Instance:  hex.EncodeToString(instance),
			StartedAt: time.Now().UTC().Truncate(time.Second),
		},
		handlers: map[string][]func(Event){},
	}, nil
}

// NodeID returns the identifier of this instance.
func (c *Cluster) NodeID() string {
	return c.self.ID
}

// Priority returns the priority of this instance.
func (c *Cluster) Priority() int {
	return c.self.Priority
}

// Client returns the client of the Redis server of the cluster.
func (c *Cluster) Client() redis.UniversalClient {
	return c.client
}

// Key builds a key or a channel name of the cluster from its parts.
func (c *Cluster) Key(parts ...string) string {
	return c.conf.Prefix + ":" + strings.Join(parts, ":")
}

// Up tells whether the last announcement reached the server.
func (c *Cluster) Up() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.up
}

// SetAttribute publishes a value with the next announcements of this instance.
func (c *Cluster) SetAttribute(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.self.Attributes == nil {
		c.self.Attributes = map[string]string{}
	}
	c.self.Attributes[key] = value
}

// Run announces this instance and listens to the events until the context is done.
func (c *Cluster) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Go(func() { c.listen(ctx) })

	c.heartbeat(ctx)

	ticker := time.NewTicker(time.Duration(c.conf.HeartbeatInterval))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.heartbeat(ctx)
		case <-ctx.Done():
			wg.Wait()
			c.leave(ctx)

			return
		}
	}
}

// Nodes returns the instances that announced themselves within the node timeout, this one included.
func (c *Cluster) Nodes(ctx context.Context) ([]Node, error) {
	ids, err := c.client.SMembers(ctx, c.Key("nodes")).Result()
	if err != nil {
		return nil, err
	}

	nodes := make([]Node, 0, len(ids))
	for _, id := range ids {
		data, err := c.client.Get(ctx, c.Key("node", id)).Bytes()
		if errors.Is(err, redis.Nil) {
			// The announcement expired: the instance is gone. One that is back adds itself again.
			c.client.SRem(ctx, c.Key("nodes"), id)
			continue
		}
		if err != nil {
			return nil, err
		}

		var node Node
		if err := json.Unmarshal(data, &node); err != nil {
			c.logger.Warn().Err(err).Str("node", id).Msg("Unable to read the announcement of a cluster instance")
			continue
		}

		nodes = append(nodes, node)
	}

	slices.SortFunc(nodes, func(a, b Node) int { return strings.Compare(a.ID, b.ID) })

	return nodes, nil
}

func (c *Cluster) heartbeat(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.conf.HeartbeatInterval))
	defer cancel()

	err := c.announce(ctx)

	c.mu.Lock()
	wasUp := c.up
	c.up = err == nil
	c.mu.Unlock()

	switch {
	case err != nil && ctx.Err() != nil && errors.Is(context.Cause(ctx), context.Canceled):
		// The instance is stopping.
	case err != nil && wasUp:
		c.logger.Warn().Err(err).Msg("Lost the cluster Redis server")
	case err != nil:
		c.logger.Debug().Err(err).Msg("Unable to reach the cluster Redis server")
	case !wasUp:
		c.logger.Info().Int("priority", c.self.Priority).Msg("Joined the cluster")
	}
}

func (c *Cluster) announce(ctx context.Context) error {
	c.mu.RLock()
	self := c.self
	self.Attributes = maps.Clone(c.self.Attributes)
	announced := c.announced
	c.mu.RUnlock()

	data, err := json.Marshal(self)
	if err != nil {
		return err
	}

	// The time to live is counted by the server, so the clocks of the instances do not have to agree.
	previous, err := c.client.SetArgs(ctx, c.Key("node", self.ID), data, redis.SetArgs{
		Get: true,
		TTL: time.Duration(c.conf.NodeTimeout),
	}).Bytes()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	// The first announcement may replace the one of this instance before it restarted.
	if err == nil && announced {
		var node Node
		if json.Unmarshal(previous, &node) == nil && node.Instance != self.Instance {
			c.logger.Error().Msg("Another instance announces itself with the same cluster nodeID: give each instance its own nodeID")
		}
	}

	if err := c.client.SAdd(ctx, c.Key("nodes"), self.ID).Err(); err != nil {
		return err
	}

	c.mu.Lock()
	c.announced = true
	c.mu.Unlock()

	return nil
}

// leave removes the announcement, so that the other instances do not wait for it to expire.
func (c *Cluster) leave(ctx context.Context) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), leaveTimeout)
	defer cancel()

	// Another instance with the same nodeID keeps its own announcement.
	if err := compareAndDeleteInstance.Run(ctx, c.client, []string{c.Key("node", c.self.ID)}, c.self.Instance).Err(); err != nil {
		c.logger.Debug().Err(err).Msg("Unable to leave the cluster")
	}

	if err := c.client.Close(); err != nil {
		c.logger.Debug().Err(err).Msg("Unable to close the cluster Redis client")
	}
}

// compareAndDeleteInstance deletes an announcement only when this process wrote it.
var compareAndDeleteInstance = redis.NewScript(`
local value = redis.call("get", KEYS[1])
if value and cjson.decode(value).instance == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`)
