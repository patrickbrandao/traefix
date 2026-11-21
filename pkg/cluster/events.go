package cluster

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// eventVersion lets a later format be told apart: the instances of a cluster are not all upgraded at once.
	eventVersion = "v1"

	listenRetryFirst   = time.Second
	listenRetryLongest = 30 * time.Second
)

// Event is a message an instance sent to all the instances of the cluster, itself included.
// Events can be lost while the server is unreachable: they speed things up,
// and what depends on them must also check the server from time to time.
type Event struct {
	// Sender is the nodeID of the instance that sent the event.
	Sender string
	Kind   string
	Args   []string
}

// Subscribe registers a handler for the events of a kind.
// The handlers of an instance run one at a time: a slow one delays the next events.
func (c *Cluster) Subscribe(kind string, handler func(Event)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers[kind] = append(c.handlers[kind], handler)
}

// OnConnect registers a function called each time the instance starts receiving events,
// at startup and after a reconnection: the time to catch up on what was missed.
// A function registered while the instance receives events is first called at the next reconnection.
func (c *Cluster) OnConnect(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.onConnect = append(c.onConnect, fn)
}

// Publish sends an event to all the instances.
func (c *Cluster) Publish(ctx context.Context, kind string, args ...string) error {
	if c.client == nil {
		return ErrNoServer
	}

	fields := make([]string, 0, len(args)+3)
	fields = append(fields, eventVersion, c.self.ID, kind)
	for _, arg := range args {
		fields = append(fields, url.QueryEscape(arg))
	}

	return c.client.Publish(ctx, c.Key("events"), strings.Join(fields, " ")).Err()
}

func parseEvent(payload string) (Event, bool) {
	fields := strings.Fields(payload)
	if len(fields) < 3 || fields[0] != eventVersion {
		return Event{}, false
	}

	event := Event{Sender: fields[1], Kind: fields[2]}
	for _, field := range fields[3:] {
		arg, err := url.QueryUnescape(field)
		if err != nil {
			return Event{}, false
		}

		event.Args = append(event.Args, arg)
	}

	return event, true
}

// listen receives the events until the context is done, and subscribes again when the connection drops.
func (c *Cluster) listen(ctx context.Context) {
	retry := listenRetryFirst

	for ctx.Err() == nil {
		connected, err := c.receive(ctx)
		if ctx.Err() != nil {
			return
		}

		if connected {
			retry = listenRetryFirst
		}

		c.logger.Debug().Err(err).Msg("Lost the subscription to the cluster events")

		select {
		case <-time.After(retry):
			retry = min(retry*2, listenRetryLongest)
		case <-ctx.Done():
			return
		}
	}
}

// receive handles the events of one subscription, and reports whether the subscription was established.
func (c *Cluster) receive(ctx context.Context) (bool, error) {
	pubsub := c.client.Subscribe(ctx, c.Key("events"))
	defer func() { _ = pubsub.Close() }()

	// A read waiting for the next event does not see the context end: closing the subscription wakes it up.
	stop := context.AfterFunc(ctx, func() { _ = pubsub.Close() })
	defer stop()

	connected := false

	for {
		message, err := pubsub.Receive(ctx)
		if err != nil {
			return connected, err
		}

		switch message := message.(type) {
		case *redis.Subscription:
			// The client subscribes again by itself after a reconnection, and says so here.
			if message.Kind != "subscribe" {
				continue
			}

			connected = true
			c.connected()
		case *redis.Message:
			c.dispatch(message.Payload)
		case *redis.Pong:
		default:
			return connected, errors.New("unexpected message on the cluster events subscription")
		}
	}
}

func (c *Cluster) connected() {
	c.mu.RLock()
	onConnect := c.onConnect
	c.mu.RUnlock()

	for _, fn := range onConnect {
		fn()
	}
}

func (c *Cluster) dispatch(payload string) {
	event, ok := parseEvent(payload)
	if !ok {
		c.logger.Debug().Str("payload", payload).Msg("Ignored a cluster event of an unknown format")
		return
	}

	c.mu.RLock()
	handlers := c.handlers[event.Kind]
	c.mu.RUnlock()

	for _, handler := range handlers {
		handler(event)
	}
}
