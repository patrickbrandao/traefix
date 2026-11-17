package docker

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	eventtypes "github.com/moby/moby/api/types/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/safe"
)

func TestShouldHandleEvent(t *testing.T) {
	event := func(id string, action eventtypes.Action) eventtypes.Message {
		return eventtypes.Message{Action: action, Actor: eventtypes.Actor{ID: id}}
	}

	testCases := []struct {
		desc   string
		events []eventtypes.Message
		want   []bool
	}{
		{
			desc:   "start and die events",
			events: []eventtypes.Message{event("a", eventtypes.ActionStart), event("a", eventtypes.ActionDie)},
			want:   []bool{true, true},
		},
		{
			desc:   "unrelated event",
			events: []eventtypes.Message{event("a", eventtypes.ActionStop)},
			want:   []bool{false},
		},
		{
			desc: "health transitions",
			events: []eventtypes.Message{
				event("a", eventtypes.ActionStart),
				event("a", eventtypes.ActionHealthStatusHealthy),
				event("a", eventtypes.ActionHealthStatusUnhealthy),
				event("a", eventtypes.ActionHealthStatusHealthy),
			},
			want: []bool{true, true, true, true},
		},
		{
			desc: "repeated healthy events",
			events: []eventtypes.Message{
				event("a", eventtypes.ActionHealthStatusHealthy),
				event("a", eventtypes.ActionHealthStatusHealthy),
				event("a", eventtypes.ActionHealthStatusHealthy),
			},
			want: []bool{true, false, false},
		},
		{
			desc: "repeated unhealthy events",
			events: []eventtypes.Message{
				event("a", eventtypes.ActionHealthStatusUnhealthy),
				event("a", eventtypes.ActionHealthStatusUnhealthy),
			},
			want: []bool{true, false},
		},
		{
			desc: "health status is tracked per container",
			events: []eventtypes.Message{
				event("a", eventtypes.ActionHealthStatusHealthy),
				event("b", eventtypes.ActionHealthStatusHealthy),
				event("b", eventtypes.ActionHealthStatusHealthy),
			},
			want: []bool{true, true, false},
		},
		{
			desc: "restart resets the health status",
			events: []eventtypes.Message{
				event("a", eventtypes.ActionHealthStatusHealthy),
				event("a", eventtypes.ActionDie),
				event("a", eventtypes.ActionStart),
				event("a", eventtypes.ActionHealthStatusHealthy),
			},
			want: []bool{true, true, true, true},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			healthStatuses := make(map[string]eventtypes.Action)

			var got []bool
			for _, e := range test.events {
				got = append(got, shouldHandleEvent(e, healthStatuses))
			}

			assert.Equal(t, test.want, got)
		})
	}
}

func TestShouldHandleEvent_cleanup(t *testing.T) {
	healthStatuses := make(map[string]eventtypes.Action)

	shouldHandleEvent(eventtypes.Message{Action: eventtypes.ActionHealthStatusHealthy, Actor: eventtypes.Actor{ID: "a"}}, healthStatuses)
	shouldHandleEvent(eventtypes.Message{Action: eventtypes.ActionDie, Actor: eventtypes.Actor{ID: "a"}}, healthStatuses)

	assert.Empty(t, healthStatuses)
}

func TestProviderReconnectsAfterEventRefreshError(t *testing.T) {
	t.Parallel()

	var versionCalls atomic.Int32
	var containerListCalls atomic.Int32
	var eventCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")

		switch {
		case req.URL.Path == "/_ping":
			rw.Header().Set("API-Version", "1.54")
			_, _ = fmt.Fprint(rw, "OK")

		case strings.HasSuffix(req.URL.Path, "/version"):
			versionCalls.Add(1)
			_, _ = fmt.Fprint(rw, `{"Version":"test","ApiVersion":"1.54","MinAPIVersion":"1.24"}`)

		case strings.HasSuffix(req.URL.Path, "/containers/json"):
			if containerListCalls.Add(1) == 2 {
				rw.WriteHeader(http.StatusInternalServerError)
				_, _ = fmt.Fprint(rw, `{"message":"transient list failure"}`)
				return
			}
			_, _ = fmt.Fprint(rw, `[]`)

		case strings.HasSuffix(req.URL.Path, "/events"):
			if eventCalls.Add(1) == 1 {
				_, _ = fmt.Fprintln(rw, `{"Type":"container","Action":"start","Actor":{"ID":"test"}}`)
			}
			_ = http.NewResponseController(rw).Flush()
			<-req.Context().Done()

		default:
			t.Errorf("unexpected Docker API request: %s %s", req.Method, req.URL.Path)
			http.NotFound(rw, req)
		}
	}))
	t.Cleanup(server.Close)

	provider := &Provider{}
	provider.SetDefaults()
	provider.Endpoint = "tcp://" + strings.TrimPrefix(server.URL, "http://")
	require.NoError(t, provider.Init())

	ctx, cancel := context.WithCancel(t.Context())
	pool := safe.NewPool(ctx)
	t.Cleanup(func() {
		cancel()
		pool.Stop()
	})

	require.NoError(t, provider.Provide(make(chan dynamic.Message, 2), pool))
	require.Eventually(t, func() bool {
		return containerListCalls.Load() >= 2
	}, 2*time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool {
		return versionCalls.Load() >= 2 && eventCalls.Load() >= 2
	}, 5*time.Second, 10*time.Millisecond)
}
