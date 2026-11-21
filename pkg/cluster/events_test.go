package cluster

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCluster_Publish(t *testing.T) {
	server := miniredis.RunT(t)

	first := newTestCluster(t, server, "srv1", 100)
	second := newTestCluster(t, server, "srv2", 100)

	received := make(chan Event, 4)
	first.Subscribe("cert", func(event Event) { received <- event })
	second.Subscribe("cert", func(event Event) { received <- event })
	second.Subscribe("want", func(Event) { assert.Fail(t, "unexpected event") })

	runTestCluster(t, first)
	runTestCluster(t, second)

	require.NoError(t, first.Publish(t.Context(), "cert", "le", "a value with spaces and %"))
	// No instance listens to this kind.
	require.NoError(t, first.Publish(t.Context(), "other"))

	// Every instance receives the event, the sender included.
	for range 2 {
		select {
		case event := <-received:
			assert.Equal(t, Event{Sender: "srv1", Kind: "cert", Args: []string{"le", "a value with spaces and %"}}, event)
		case <-time.After(5 * time.Second):
			require.Fail(t, "event not received")
		}
	}
}

func TestCluster_OnConnect_afterReconnection(t *testing.T) {
	server := miniredis.RunT(t)

	c := newTestCluster(t, server, "srv1", 100)

	var connections atomic.Int32
	c.OnConnect(func() { connections.Add(1) })

	received := make(chan Event, 1)
	c.Subscribe("cert", func(event Event) { received <- event })

	runTestCluster(t, c)
	require.Equal(t, int32(1), connections.Load())

	server.Close()
	require.NoError(t, server.Restart())

	require.Eventually(t, func() bool { return connections.Load() >= 2 }, 10*time.Second, 20*time.Millisecond)

	// The events are received again.
	require.NoError(t, c.Publish(t.Context(), "cert"))

	select {
	case event := <-received:
		assert.Equal(t, "cert", event.Kind)
	case <-time.After(5 * time.Second):
		require.Fail(t, "event not received")
	}
}

func Test_parseEvent(t *testing.T) {
	testCases := []struct {
		desc     string
		payload  string
		expected Event
		ok       bool
	}{
		{desc: "no argument", payload: "v1 srv1 cert", expected: Event{Sender: "srv1", Kind: "cert"}, ok: true},
		{desc: "arguments", payload: "v1 srv1 cert le a+b%25", expected: Event{Sender: "srv1", Kind: "cert", Args: []string{"le", "a b%"}}, ok: true},
		{desc: "empty"},
		{desc: "too short", payload: "v1 srv1"},
		{desc: "unknown version", payload: "v2 srv1 cert"},
		{desc: "bad escape", payload: "v1 srv1 cert %zz"},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			event, ok := parseEvent(test.payload)
			assert.Equal(t, test.ok, ok)
			assert.Equal(t, test.expected, event)
		})
	}
}
