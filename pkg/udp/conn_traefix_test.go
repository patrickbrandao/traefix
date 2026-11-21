package udp

import (
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFlowConn(t *testing.T) {
	remote := &net.UDPAddr{IP: net.IPv4(198, 51, 100, 7), Port: 5353}
	sent := make(chan []byte, 1)

	var ended atomic.Int32

	conn := NewFlowConn(remote, time.Hour, func(p []byte) error {
		sent <- append([]byte(nil), p...)
		return nil
	}, func() { ended.Add(1) })

	assert.Equal(t, remote, conn.RemoteAddr())
	assert.Equal(t, time.Hour, conn.Timeout())

	require.True(t, conn.Deliver([]byte("first")))
	require.True(t, conn.Deliver([]byte("second")))

	// Each read gets one packet, in the order they came.
	buf := make([]byte, 64)

	n, err := conn.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "first", string(buf[:n]))

	n, err = conn.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "second", string(buf[:n]))

	n, err = conn.Write([]byte("answer"))
	require.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, "answer", string(<-sent))

	require.NoError(t, conn.Close())
	require.NoError(t, conn.Close())

	assert.Equal(t, int32(1), ended.Load())
	assert.False(t, conn.Deliver([]byte("late")))

	_, err = conn.Read(buf)
	require.ErrorIs(t, err, io.EOF)

	select {
	case <-conn.Done():
	default:
		t.Fatal("the session should be done")
	}
}

func TestNewFlowConn_timeout(t *testing.T) {
	ended := make(chan struct{})

	conn := NewFlowConn(&net.UDPAddr{IP: net.IPv4(198, 51, 100, 7), Port: 5353}, 100*time.Millisecond, func([]byte) error { return nil }, func() { close(ended) })

	// Without a packet for its timeout, the session ends as the ones of a listener do.
	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		t.Fatal("the session should have ended")
	}

	_, err := conn.Read(make([]byte, 8))
	require.ErrorIs(t, err, io.EOF)
}
