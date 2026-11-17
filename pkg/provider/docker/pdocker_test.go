package docker

import (
	"testing"

	eventtypes "github.com/moby/moby/api/types/events"
	"github.com/stretchr/testify/assert"
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
