package acme

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-acme/lego/v5/acme"
	"github.com/go-acme/lego/v5/certificate"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
)

func (i *testInstance) certificates() []*CertAndStore {
	i.provider.certificatesMu.RLock()
	defer i.provider.certificatesMu.RUnlock()

	return append([]*CertAndStore(nil), i.provider.certificates...)
}

func (i *testInstance) held(id string) bool {
	i.t.Helper()

	_, held, err := i.member.LockHolder(i.t.Context(), i.coord.lockName(id))
	require.NoError(i.t, err)

	return held
}

func TestCoordinator_announceRenewals(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 100, true)
	second := newTestInstance(t, server, "srv2", 50, true)

	var mu sync.Mutex
	var events []cluster.Event

	first.member.Subscribe(coordinatorEventRenew, func(event cluster.Event) {
		mu.Lock()
		defer mu.Unlock()

		events = append(events, event)
	})

	received := func() []cluster.Event {
		mu.Lock()
		defer mu.Unlock()

		return append([]cluster.Event(nil), events...)
	}

	// The instance with the highest priority has no one to tell.
	first.coord.announceRenewals(t.Context(), []string{"a:1"})

	// The subscription may not be in place for the first events.
	require.Eventually(t, func() bool {
		second.coord.announceRenewals(t.Context(), []string{"a:1", "b:2"})

		return len(received()) > 0
	}, 5*time.Second, 20*time.Millisecond)

	event := received()[0]
	assert.Equal(t, "srv2", event.Sender)
	assert.Equal(t, []string{"le", strconv.Itoa(50), "a:1", "b:2"}, event.Args)
}

func TestProvider_electRenewals_takesAnnouncedRenewals(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 100, true)
	second := newTestInstance(t, server, "srv2", 50, true)

	first.resolve("example.com")
	first.resolve("example.org")
	require.True(t, second.covers("example.com"))
	require.True(t, second.covers("example.org"))

	server.FastForward(testElectionLockTTL + time.Second)

	com := certID([]string{"example.com"})
	org := certID([]string{"example.org"})

	announced := renewItem(com, first.provider.certificateSum(com))

	// The instance takes the renewal another one announced, for the certificate both hold.
	// What it does not hold, or holds in another version, is left to the instance that announced it.
	first.coord.reserve(t.Context(), first.provider, []string{
		announced,
		renewItem(org, "0000000000000000"),
		renewItem(certID([]string{"example.net"}), "0000000000000000"),
		"malformed",
	})

	assert.True(t, first.held(com))
	assert.False(t, first.held(org))

	// The instance that announced it finds the renewal taken, and takes the other one.
	elected, done := second.provider.electRenewals(t.Context(), second.certificates())
	require.Len(t, elected, 1)
	assert.Equal(t, "example.org", elected[0].Domain.Main)
	done()

	// The check of the first instance finds nothing due by itself: it renews what it took.
	elected, done = first.provider.electRenewals(t.Context(), nil)
	require.Len(t, elected, 1)
	assert.Equal(t, "example.com", elected[0].Domain.Main)

	renewed := newStoreTestCertificate(t, in90Days().Add(24*time.Hour), "example.com")
	require.NoError(t, first.provider.addCertificateForDomain(elected[0].Domain,
		&certificate.Resource{Certificate: renewed.Certificate.Certificate, PrivateKey: renewed.Key}, "default"))
	done()

	assert.False(t, first.held(com))
	assert.False(t, server.Exists("traefix:acme:{le}:fail:"+com))

	// An announcement that comes late names the certificate that was replaced: it is ignored.
	first.coord.reserve(t.Context(), first.provider, []string{announced})
	assert.False(t, first.held(com))

	elected, done = first.provider.electRenewals(t.Context(), nil)
	done()
	assert.Empty(t, elected)
}

func TestCoordinator_release(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 100, true)

	first.resolve("example.com")
	server.FastForward(testElectionLockTTL + time.Second)

	id := certID([]string{"example.com"})

	first.coord.reserve(t.Context(), first.provider, []string{renewItem(id, first.provider.certificateSum(id))})
	require.True(t, first.held(id))

	// A reserved renewal that no renewal took does not keep the other instances away.
	first.coord.release(t.Context())
	assert.False(t, first.held(id))
}

func TestCoordinator_acquireAll_waitsOnce(t *testing.T) {
	server := miniredis.RunT(t)

	newTestInstance(t, server, "srv1", 100, true)
	second := newTestInstance(t, server, "srv2", 50, true)

	var ids []string
	for i := range 10 {
		ids = append(ids, certID([]string{fmt.Sprintf("app%d.example.com", i)}))
	}

	start := time.Now()
	locks := second.coord.acquireAll(t.Context(), ids)
	elapsed := time.Since(start)

	assert.Len(t, locks, len(ids))
	assert.GreaterOrEqual(t, elapsed, testElectionStep)
	assert.Less(t, elapsed, 5*testElectionStep)
}

func TestCoordinator_acquireAll_afterItsOwnFailure(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 50, true)

	failed := certID([]string{"failed.example.com"})
	other := certID([]string{"other.example.com"})

	first.coord.failed(t.Context(), failed, errors.New("boom"), false)
	first.advance(coordinatorBackoff[0] + time.Second)

	// The instance that failed lets the other instances try first: with one level, two steps.
	start := time.Now()
	locks := first.coord.acquireAll(t.Context(), []string{failed, other})

	assert.Len(t, locks, 2)
	assert.GreaterOrEqual(t, time.Since(start), 2*testElectionStep)
}

func TestCoordinator_serverDown(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 50, true)
	second := newTestInstance(t, server, "srv2", 50, true)

	first.resolve("example.org")
	require.True(t, second.covers("example.org"))

	server.Close()
	first.store.refresh("le")
	require.Error(t, first.store.Ready("le"))

	// Nothing is asked to the ACME server, and nothing waits for the Redis server.
	start := time.Now()
	first.resolve("example.com")
	first.coord.poll(t.Context())

	elected, done := first.provider.electRenewals(t.Context(), first.certificates())
	done()

	assert.Empty(t, elected)
	assert.Equal(t, 1, first.requests())
	assert.Less(t, time.Since(start), time.Second)

	// The request is taken up when the server is back, and the other instance learns about it.
	require.NoError(t, server.Restart())
	first.store.refresh("le")
	require.NoError(t, first.store.Ready("le"))

	first.coord.poll(t.Context())

	require.Eventually(t, func() bool { return second.covers("example.com") }, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, 2, first.requests())
	assert.Equal(t, 0, second.requests())
}

func TestCoordinator_warnOrphan(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 0, true)

	var logs bytes.Buffer
	first.coord.logger = zerolog.New(&logs)

	first.resolve("example.com")
	first.coord.poll(t.Context())

	assert.Contains(t, logs.String(), "No instance of the cluster requests the certificates of this resolver")

	// The warning is not repeated, and comes back after an instance that requests certificates has come and gone.
	logs.Reset()
	first.advance(coordinatorWantsInterval)
	first.coord.poll(t.Context())
	assert.Empty(t, logs.String())

	second := newTestInstance(t, server, "srv2", 50, true)

	first.advance(coordinatorWantsInterval)
	first.coord.poll(t.Context())
	assert.Empty(t, logs.String())

	server.Del("traefix:node:" + second.nodeID)

	first.advance(coordinatorWantsInterval)
	first.coord.poll(t.Context())
	assert.Contains(t, logs.String(), "No instance of the cluster requests the certificates of this resolver")
}

func TestCoordinator_clearsARequestLeftBehind(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestInstance(t, server, "srv1", 50, true)
	second := newTestInstance(t, server, "srv2", 50, true)

	first.resolve("example.com")
	require.True(t, second.covers("example.com"))

	// The request stays when the instance that asked for the certificate stops before the certificate arrives.
	id := certID([]string{"example.com"})

	want, err := json.Marshal(wantRecord{Domains: []string{"example.com"}, Store: "default", Node: "srv3"})
	require.NoError(t, err)
	require.NoError(t, server.Set("traefix:acme:{le}:want:"+id, string(want)))
	_, err = server.SAdd("traefix:acme:{le}:wants", id)
	require.NoError(t, err)

	second.coord.poll(t.Context())

	assert.False(t, server.Exists("traefix:acme:{le}:want:"+id))
	assert.Equal(t, 0, second.requests())
}

func Test_renewCertificate(t *testing.T) {
	refused := &acme.ProblemDetails{Type: acme.UnauthorizedErrorType, HTTPStatus: 403, Detail: "requester account did not request the certificate being replaced by this order"}

	testCases := []struct {
		desc string
		// errs are the errors of the successive requests.
		errs          []error
		useARI        bool
		expectedCalls []bool
		expectedErr   bool
	}{
		{
			desc:          "renewed at once",
			errs:          []error{nil},
			useARI:        true,
			expectedCalls: []bool{true},
		},
		{
			desc:          "order refused, renewed without the link to the replaced certificate",
			errs:          []error{refused, nil},
			useARI:        true,
			expectedCalls: []bool{true, false},
		},
		{
			desc:          "order refused twice",
			errs:          []error{refused, refused},
			useARI:        true,
			expectedCalls: []bool{true, false},
			expectedErr:   true,
		},
		{
			desc:          "order refused without the link",
			errs:          []error{refused},
			expectedCalls: []bool{false},
			expectedErr:   true,
		},
		{
			desc:          "validation failed, not tried again",
			errs:          []error{fmt.Errorf("example.com: %w", refused)},
			useARI:        true,
			expectedCalls: []bool{true},
			expectedErr:   true,
		},
		{
			desc:          "ACME server unreachable, not tried again",
			errs:          []error{errors.New("connection refused")},
			useARI:        true,
			expectedCalls: []bool{true},
			expectedErr:   true,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			var calls []bool

			renew := func(_ context.Context, res certificate.Resource, opts *certificate.RenewOptions) (*certificate.Resource, error) {
				err := test.errs[len(calls)]
				calls = append(calls, opts.UseARICertID)

				if err != nil {
					return nil, err
				}

				return &res, nil
			}

			opts := &certificate.RenewOptions{Bundle: true, UseARICertID: test.useARI}

			renewed, err := renewCertificate(t.Context(), renew, certificate.Resource{Domains: []string{"example.com"}}, opts)

			assert.Equal(t, test.expectedCalls, calls)
			assert.Equal(t, test.useARI, opts.UseARICertID)

			if test.expectedErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, renewed)
		})
	}
}
