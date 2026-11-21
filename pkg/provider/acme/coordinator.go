package acme

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-acme/lego/v5/certificate"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	"github.com/traefik/traefik/v3/pkg/safe"
	"github.com/traefik/traefik/v3/pkg/types"
)

const (
	// coordinatorEventWant tells the instances that a certificate is needed: the one that sends it may not be the one to obtain it.
	coordinatorEventWant = "want"
	// coordinatorEventRenew makes the instances check their renewals now, so that the one with the highest priority takes them.
	coordinatorEventRenew = "renew"

	coordinatorPollInterval  = 5 * time.Second
	coordinatorWantsInterval = 30 * time.Second
	coordinatorWantRefresh   = time.Hour

	// coordinatorRecordTTL is longer than the longest wait between two attempts: a failure is not forgotten in between.
	coordinatorRecordTTL = 24 * time.Hour
	// coordinatorGiveUpAfter stops the attempts for a certificate that keeps failing, e.g. for a domain that left the configuration.
	// A configuration change that still asks for it starts them again.
	coordinatorGiveUpAfter = 14 * 24 * time.Hour

	coordinatorMaxJitter   = 500 * time.Millisecond
	coordinatorAccountWait = 500 * time.Millisecond

	coordinatorMaxError = 500
)

// coordinatorBackoff spaces the attempts after a failure. With the first try, it allows at most four attempts in an hour,
// under the limit of five failed validations per hour of Let's Encrypt.
var coordinatorBackoff = []time.Duration{
	2 * time.Minute, 15 * time.Minute, 30 * time.Minute, time.Hour, 2 * time.Hour, 6 * time.Hour,
}

// coordinatorAttemptKey marks the context of an attempt the coordinator started itself,
// as opposed to a request of the configuration of this instance.
type coordinatorAttemptKey struct{}

// coordinator makes the instances that share a certificate store agree on the one that asks the ACME server:
// one instance for each certificate, the one with the highest priority that is able to.
type coordinator struct {
	member       *cluster.Cluster
	store        *RedisStore
	resolverName string
	step         time.Duration
	lockTTL      time.Duration
	// settle is the time an instance that just started waits before its first elections:
	// the instances that start together need it to see each other.
	settle time.Duration
	logger zerolog.Logger

	now    func() time.Time
	jitter func() time.Duration
	// obtain replaces the request to the ACME server, in the tests.
	obtain func(ctx context.Context, domain types.Domain, tlsStore string) (types.Domain, *certificate.Resource, error)

	mu          sync.Mutex
	provider    *Provider
	digest      string
	settleUntil time.Time
	pending     map[string]*pendingRequest
	warned      map[string]struct{}
	lastWants   time.Time

	signals  chan coordinatorSignal
	renewing atomic.Bool
}

// pendingRequest is a certificate that some instance needs and that no instance stored yet.
type pendingRequest struct {
	domains  []string
	tlsStore string
	// local is true when the configuration of this instance asks for the certificate.
	local bool
	// notBefore is the time before which this instance does not try again.
	notBefore   time.Time
	wantWritten time.Time
}

type coordinatorSignal struct {
	kind string
	id   string
}

type wantRecord struct {
	Domains []string `json:"domains"`
	Store   string   `json:"store"`
	Node    string   `json:"node"`
	At      string   `json:"at"`
}

type failRecord struct {
	Error       string `json:"error"`
	Node        string `json:"node"`
	Attempts    int    `json:"attempts"`
	FirstAt     int64  `json:"firstAt"`
	NextRetryAt int64  `json:"nextRetryAt"`
}

func newCoordinator(member *cluster.Cluster, store *RedisStore, resolverName string, step, lockTTL time.Duration) *coordinator {
	return &coordinator{
		member:       member,
		store:        store,
		resolverName: resolverName,
		step:         step,
		lockTTL:      lockTTL,
		settle:       member.HeartbeatInterval(),
		logger:       log.With().Str(logs.ProviderName, resolverName+resolverSuffix).Logger(),
		now:          time.Now,
		jitter:       func() time.Duration { return rand.N(coordinatorMaxJitter) },
		pending:      map[string]*pendingRequest{},
		warned:       map[string]struct{}{},
		signals:      make(chan coordinatorSignal, 64),
	}
}

// attach binds the resolver and announces it to the other instances, and reports whether it was already bound.
func (c *coordinator) attach(provider *Provider) bool {
	c.mu.Lock()
	attached := c.provider != nil
	c.provider = provider
	c.digest = resolverDigest(provider.Configuration)
	if !attached {
		c.settleUntil = c.now().Add(c.settle)
	}
	c.mu.Unlock()

	// The instances with the resolver are the ones that count in the election.
	c.member.SetAttribute(c.attribute(), c.digest)

	return attached
}

// start attaches the resolver and follows the requests of the other instances.
func (c *coordinator) start(provider *Provider, pool *safe.Pool) {
	if c.attach(provider) {
		return
	}

	c.member.Subscribe(coordinatorEventWant, func(event cluster.Event) {
		if len(event.Args) == 2 && event.Args[0] == c.resolverName && event.Sender != c.member.NodeID() {
			c.signal(coordinatorSignal{kind: coordinatorEventWant, id: event.Args[1]})
		}
	})

	c.member.Subscribe(coordinatorEventRenew, func(event cluster.Event) {
		if len(event.Args) == 1 && event.Args[0] == c.resolverName && event.Sender != c.member.NodeID() {
			c.signal(coordinatorSignal{kind: coordinatorEventRenew})
		}
	})

	pool.GoCtx(c.run)
}

func (c *coordinator) signal(sig coordinatorSignal) {
	// A signal that does not fit is caught by the next poll.
	select {
	case c.signals <- sig:
	default:
	}
}

func (c *coordinator) run(ctx context.Context) {
	ctx = c.logger.WithContext(ctx)

	ticker := time.NewTicker(coordinatorPollInterval)
	defer ticker.Stop()

	for {
		select {
		case sig := <-c.signals:
			switch sig.kind {
			case coordinatorEventWant:
				c.learn(ctx, sig.id)
				c.poll(ctx)
			case coordinatorEventRenew:
				c.renew(ctx)
			}
		case <-ticker.C:
			c.poll(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// poll follows the pending requests: it drops the ones that are served, and tries the ones that are due.
func (c *coordinator) poll(ctx context.Context) {
	c.mu.Lock()
	provider := c.provider
	listWants := c.now().Sub(c.lastWants) >= coordinatorWantsInterval
	if listWants {
		c.lastWants = c.now()
	}
	c.mu.Unlock()

	if provider == nil {
		return
	}

	if listWants {
		c.listWants(ctx)
	}

	c.mu.Lock()
	pending := make(map[string]pendingRequest, len(c.pending))
	for id, request := range c.pending {
		pending[id] = *request
	}
	c.mu.Unlock()

	now := c.now()

	for id, request := range pending {
		if len(provider.uncoveredDomains(request.domains, request.tlsStore)) == 0 {
			c.forget(ctx, id)
			continue
		}

		if request.local && now.Sub(request.wantWritten) > coordinatorWantRefresh {
			c.writeWant(ctx, id, request.domains, request.tlsStore)
		}

		if c.member.Priority() == 0 || now.Before(request.notBefore) {
			continue
		}

		// The attempt reports when to try next. Until then, this keeps the next polls from starting it again.
		c.postpone(id, now.Add(2*c.step+coordinatorMaxJitter))

		provider.resolveDomains(context.WithValue(ctx, coordinatorAttemptKey{}, true), request.domains, request.tlsStore)
	}
}

// listWants learns the requests of the other instances that this one missed, and drops the ones that are gone.
func (c *coordinator) listWants(ctx context.Context) {
	ids, err := c.member.Client().SMembers(ctx, c.key("wants")).Result()
	if err != nil {
		return
	}

	listed := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		listed[id] = struct{}{}
		c.learn(ctx, id)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for id, request := range c.pending {
		if _, ok := listed[id]; !ok && !request.local {
			delete(c.pending, id)
		}
	}
}

// learn records the request of another instance.
func (c *coordinator) learn(ctx context.Context, id string) {
	c.mu.Lock()
	_, known := c.pending[id]
	c.mu.Unlock()

	if known {
		return
	}

	data, err := c.member.Client().Get(ctx, c.key("want", id)).Bytes()
	if errors.Is(err, redis.Nil) {
		// The request expired: its entry in the index goes with it.
		c.member.Client().SRem(ctx, c.key("wants"), id)
		return
	}
	if err != nil {
		return
	}

	var want wantRecord
	if err := json.Unmarshal(data, &want); err != nil || len(want.Domains) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, known := c.pending[id]; !known {
		c.pending[id] = &pendingRequest{domains: want.Domains, tlsStore: want.Store}
	}
}

// request records that the configuration of this instance needs a certificate, and tells the other instances.
func (c *coordinator) request(ctx context.Context, id string, domains []string, tlsStore string) {
	c.mu.Lock()
	request, known := c.pending[id]
	if !known {
		request = &pendingRequest{domains: domains, tlsStore: tlsStore}
		c.pending[id] = request
	}
	announce := !request.local
	request.local = true
	c.mu.Unlock()

	if !announce {
		return
	}

	c.writeWant(ctx, id, domains, tlsStore)

	if err := c.member.Publish(ctx, coordinatorEventWant, c.resolverName, id); err != nil {
		// The other instances find the request when they list the requests.
		c.logger.Debug().Err(err).Msg("Unable to announce a certificate request to the cluster")
	}
}

func (c *coordinator) writeWant(ctx context.Context, id string, domains []string, tlsStore string) {
	data, err := json.Marshal(wantRecord{
		Domains: domains,
		Store:   tlsStore,
		Node:    c.member.NodeID(),
		At:      c.now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return
	}

	client := c.member.Client()

	if err := client.Set(ctx, c.key("want", id), data, coordinatorRecordTTL).Err(); err != nil {
		c.logger.Debug().Err(err).Msg("Unable to store a certificate request in the cluster")
		return
	}

	client.SAdd(ctx, c.key("wants"), id)

	c.mu.Lock()
	defer c.mu.Unlock()

	if request, ok := c.pending[id]; ok {
		request.wantWritten = c.now()
	}
}

// forget drops a request that is served.
func (c *coordinator) forget(ctx context.Context, id string) {
	c.mu.Lock()
	request, known := c.pending[id]
	delete(c.pending, id)
	c.mu.Unlock()

	if known && request.local {
		c.clear(ctx, id)
	}
}

// clear removes what the cluster keeps about a request.
func (c *coordinator) clear(ctx context.Context, id string) {
	client := c.member.Client()

	client.Del(ctx, c.key("want", id), c.key("fail", id))
	client.SRem(ctx, c.key("wants"), id)
}

// succeeded records that this instance obtained the certificate.
func (c *coordinator) succeeded(ctx context.Context, id string) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()

	c.clear(ctx, id)
}

// failed records the failure of an attempt, which tells every instance when the next one is allowed.
// With giveUp, a certificate that has failed for too long is abandoned.
func (c *coordinator) failed(ctx context.Context, id string, cause error, giveUp bool) {
	now := c.now()

	record := failRecord{FirstAt: now.Unix()}
	if previous, err := c.readFail(ctx, id); err == nil && previous != nil {
		record = *previous
	}

	record.Attempts++
	record.Node = c.member.NodeID()
	record.Error = cause.Error()
	if len(record.Error) > coordinatorMaxError {
		record.Error = record.Error[:coordinatorMaxError]
	}

	if giveUp && now.Sub(time.Unix(record.FirstAt, 0)) > coordinatorGiveUpAfter {
		c.logger.Warn().Strs("domains", c.domainsOf(id)).Int("attempts", record.Attempts).
			Msg("Gave up on a certificate that keeps failing, until the configuration asks for it again")

		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()

		c.clear(ctx, id)

		return
	}

	next := now.Add(coordinatorBackoff[min(record.Attempts, len(coordinatorBackoff))-1])
	record.NextRetryAt = next.Unix()

	data, err := json.Marshal(record)
	if err != nil {
		return
	}

	if err := c.member.Client().Set(ctx, c.key("fail", id), data, coordinatorRecordTTL).Err(); err != nil {
		c.logger.Debug().Err(err).Msg("Unable to store a certificate failure in the cluster")
	}

	c.postpone(id, next)
}

func (c *coordinator) readFail(ctx context.Context, id string) (*failRecord, error) {
	data, err := c.member.Client().Get(ctx, c.key("fail", id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	record := &failRecord{}
	if err := json.Unmarshal(data, record); err != nil {
		//nolint:nilerr // A record that cannot be read does not hold the certificate back.
		return nil, nil
	}

	return record, nil
}

// acquire decides whether this instance asks the ACME server for a certificate now, and returns the lock when it does.
// The instances with the highest priority try first, and the others after one step for each level above their own:
// they only get the lock when the ones above did not take it.
func (c *coordinator) acquire(ctx context.Context, id string) (*cluster.Lock, error) {
	if c.member.Priority() == 0 {
		return nil, nil
	}

	failure, err := c.readFail(ctx, id)
	if err != nil {
		return nil, err
	}

	if failure != nil {
		if next := time.Unix(failure.NextRetryAt, 0); c.now().Before(next) {
			c.postpone(id, next)
			return nil, nil
		}
	}

	if _, held, err := c.member.LockHolder(ctx, c.lockName(id)); err != nil || held {
		c.postpone(id, c.now().Add(c.step))
		return nil, err
	}

	// The instances are counted once the ones that started together had the time to announce themselves.
	c.mu.Lock()
	settling := c.settleUntil.Sub(c.now())
	c.mu.Unlock()

	if err := sleep(ctx, settling); err != nil {
		return nil, err
	}

	delay, err := c.electionDelay(ctx, failure)
	if err != nil {
		return nil, err
	}

	if err := sleep(ctx, delay); err != nil {
		return nil, err
	}

	lock, ok, err := c.member.TryLock(ctx, c.lockName(id), c.lockTTL)
	if err != nil || !ok {
		c.postpone(id, c.now().Add(c.step))
		return nil, err
	}

	// Another instance may have tried and failed while this one waited for its turn.
	if failure, err := c.readFail(ctx, id); err == nil && failure != nil {
		if next := time.Unix(failure.NextRetryAt, 0); c.now().Before(next) {
			c.postpone(id, next)
			_ = lock.Release(ctx)

			return nil, nil
		}
	}

	return lock, nil
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}

	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// hold extends a lock while the work goes on, and cancels the returned context when the lock is lost:
// the server says that another instance may be doing the same work.
func (c *coordinator) hold(ctx context.Context, lock *cluster.Lock) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	go lock.Keep(ctx)

	go func() {
		select {
		case <-lock.Lost():
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

// electionDelay returns how long this instance waits before it tries to take a request.
func (c *coordinator) electionDelay(ctx context.Context, failure *failRecord) (time.Duration, error) {
	above, levels, err := c.levels(ctx)
	if err != nil {
		return 0, err
	}

	delay := time.Duration(above)*c.step + c.jitter()

	// The instance that just failed lets every other instance try before it does again.
	if failure != nil && failure.Node == c.member.NodeID() {
		delay += time.Duration(levels+1) * c.step
	}

	return delay, nil
}

// levels counts the distinct priorities above the one of this instance, and the distinct priorities in all,
// among the instances that have the resolver and may obtain certificates.
func (c *coordinator) levels(ctx context.Context) (int, int, error) {
	nodes, err := c.member.Nodes(ctx)
	if err != nil {
		return 0, 0, err
	}

	c.mu.Lock()
	digest := c.digest
	c.mu.Unlock()

	all := map[int]struct{}{c.member.Priority(): {}}
	above := map[int]struct{}{}

	for _, node := range nodes {
		nodeDigest, ok := node.Attributes[c.attribute()]
		if !ok || node.Priority == 0 {
			continue
		}

		if nodeDigest != digest {
			c.warnMismatch(node.ID)
		}

		all[node.Priority] = struct{}{}
		if node.Priority > c.member.Priority() {
			above[node.Priority] = struct{}{}
		}
	}

	return len(above), len(all), nil
}

// warnMismatch tells, once for each instance, that it does not configure the resolver the same way.
func (c *coordinator) warnMismatch(nodeID string) {
	c.mu.Lock()
	_, warned := c.warned[nodeID]
	c.warned[nodeID] = struct{}{}
	c.mu.Unlock()

	if !warned {
		c.logger.Warn().Str("node", nodeID).
			Msg("Another instance of the cluster configures this certificate resolver differently (CA server, email, key type, challenge, profile or preferred chain): the certificates depend on the instance that obtains them")
	}
}

// lockAccount returns the ACME account the instances share when one is usable,
// or the lock under which this instance registers it. The other instances wait for it instead of registering their own.
func (c *coordinator) lockAccount(ctx context.Context, usable func(*Account) bool) (*Account, *cluster.Lock, error) {
	deadline := c.now().Add(2 * c.lockTTL)

	for {
		account, err := c.store.GetAccount(c.resolverName)
		if err != nil {
			return nil, nil, err
		}

		if usable(account) {
			return account, nil, nil
		}

		lock, ok, err := c.member.TryLock(ctx, c.lockName("account"), c.lockTTL)
		if err != nil {
			return nil, nil, err
		}

		if ok {
			// The account may have been stored between the read and the lock.
			account, err := c.store.GetAccount(c.resolverName)
			if err != nil || usable(account) {
				_ = lock.Release(ctx)
				return account, nil, err
			}

			return nil, lock, nil
		}

		if c.now().After(deadline) {
			return nil, nil, errors.New("another instance of the cluster is registering the ACME account for too long")
		}

		select {
		case <-time.After(coordinatorAccountWait):
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}
}

// renew checks the renewals of this instance now, unless it is already doing so.
func (c *coordinator) renew(ctx context.Context) {
	c.mu.Lock()
	provider := c.provider
	c.mu.Unlock()

	if provider == nil || !c.renewing.CompareAndSwap(false, true) {
		return
	}

	safe.Go(func() {
		defer c.renewing.Store(false)

		renewPeriod, renewInterval := getCertificateRenewDurations(provider.CertificatesDuration)
		provider.renewCertificates(ctx, renewPeriod, renewInterval)
	})
}

func (c *coordinator) postpone(id string, until time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if request, ok := c.pending[id]; ok {
		request.notBefore = until
	}
}

func (c *coordinator) domainsOf(id string) []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if request, ok := c.pending[id]; ok {
		return request.domains
	}

	return nil
}

func (c *coordinator) key(parts ...string) string {
	return c.store.key(c.resolverName, parts...)
}

func (c *coordinator) lockName(id string) string {
	return "acme:" + c.resolverName + ":" + id
}

func (c *coordinator) attribute() string {
	return "acme." + c.resolverName
}

// resolverDigest identifies the settings that the instances must share to obtain the same certificates.
func resolverDigest(conf *Configuration) string {
	challenge := "none"
	switch {
	case conf.DNSChallenge != nil:
		challenge = "dns:" + conf.DNSChallenge.Provider
	case conf.HTTPChallenge != nil:
		challenge = "http"
	case conf.TLSChallenge != nil:
		challenge = "tls"
	}

	sum := sha256.Sum256([]byte(strings.Join([]string{
		conf.CAServer, conf.Email, conf.KeyType, challenge, conf.Profile, conf.PreferredChain,
	}, "\x00")))

	return hex.EncodeToString(sum[:8])
}
