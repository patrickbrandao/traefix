package cluster

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// lockValueSeparator splits the node from the random part of a lock value. A nodeID cannot contain it.
const lockValueSeparator = "|"

// Lock is a named lock of the cluster, held by one instance at a time.
// It expires by itself when its holder stops extending it, so a crashed holder never blocks the others for long.
type Lock struct {
	cluster *Cluster
	key     string
	value   string
	ttl     time.Duration

	lostOnce sync.Once
	lost     chan struct{}
}

// TryLock takes the lock when no instance holds it, and reports whether it did.
// The holder must call Keep to hold the lock longer than its time to live, and Release when done.
func (c *Cluster) TryLock(ctx context.Context, name string, ttl time.Duration) (*Lock, bool, error) {
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return nil, false, err
	}

	lock := &Lock{
		cluster: c,
		key:     c.Key("lock", name),
		value:   c.self.ID + lockValueSeparator + hex.EncodeToString(token),
		ttl:     ttl,
		lost:    make(chan struct{}),
	}

	err := c.client.SetArgs(ctx, lock.key, lock.value, redis.SetArgs{Mode: "NX", TTL: ttl}).Err()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	return lock, true, nil
}

// LockHolder returns the identifier of the instance that holds a lock, if any.
func (c *Cluster) LockHolder(ctx context.Context, name string) (string, bool, error) {
	value, err := c.client.Get(ctx, c.Key("lock", name)).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	nodeID, _, _ := strings.Cut(value, lockValueSeparator)

	return nodeID, true, nil
}

// Keep extends the lock at a third of its time to live until the context is done or the lock is lost.
// It must run in its own goroutine.
func (l *Lock) Keep(ctx context.Context) {
	ticker := time.NewTicker(max(l.ttl/3, time.Millisecond))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			held, err := l.Extend(ctx)
			if err != nil {
				// The server cannot be reached, which does not tell that another instance took the lock:
				// the work goes on, and its result is only kept if no other instance produced one.
				l.cluster.logger.Debug().Err(err).Str("lock", l.key).Msg("Unable to extend a cluster lock")
				continue
			}

			if !held {
				return
			}
		case <-ctx.Done():
			return
		case <-l.lost:
			return
		}
	}
}

// Extend restarts the time to live of the lock, and reports whether this instance still holds it.
func (l *Lock) Extend(ctx context.Context) (bool, error) {
	extended, err := compareAndExpire.Run(ctx, l.cluster.client, []string{l.key}, l.value, l.ttl.Milliseconds()).Int()
	if err != nil {
		return false, err
	}

	if extended == 0 {
		l.lostOnce.Do(func() { close(l.lost) })
		return false, nil
	}

	return true, nil
}

// Lost is closed when the server says that this instance does not hold the lock anymore.
func (l *Lock) Lost() <-chan struct{} {
	return l.lost
}

// Release frees the lock, unless another instance holds it by now.
func (l *Lock) Release(ctx context.Context) error {
	l.lostOnce.Do(func() { close(l.lost) })

	return compareAndDelete.Run(ctx, l.cluster.client, []string{l.key}, l.value).Err()
}

// The value is compared on the server: a holder that was too slow must not extend or free the lock of its successor.
var (
	compareAndExpire = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("pexpire", KEYS[1], ARGV[2])
end
return 0
`)

	compareAndDelete = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`)
)
