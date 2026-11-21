package udp

import (
	"net"
	"sync"
	"time"
)

// flowHooks are what a session made outside a listener sends its packets through, and tells when it ends (Traefix).
type flowHooks struct {
	send  func(p []byte) error
	ended func()
	once  sync.Once
}

func (h *flowHooks) write(p []byte) (int, error) {
	if err := h.send(p); err != nil {
		return 0, err
	}

	return len(p), nil
}

func (h *flowHooks) end() {
	h.once.Do(h.ended)
}

// NewFlowConn returns a session whose packets come from Deliver and go to send, outside any listener (Traefix):
// the packets of a UDP session that another instance of the mesh received. The session ends after the timeout
// without a packet, as the ones of a listener do, and ended is called once when it ends.
func NewFlowConn(remote net.Addr, timeout time.Duration, send func(p []byte) error, ended func()) *Conn {
	c := &Conn{
		rAddr:        remote,
		receiveCh:    make(chan []byte),
		readCh:       make(chan []byte),
		sizeCh:       make(chan int),
		doneCh:       make(chan struct{}),
		timeout:      timeout,
		lastActivity: time.Now(),
		flow:         &flowHooks{send: send, ended: ended},
	}

	go c.readLoop()

	return c
}

// Deliver gives a session made by NewFlowConn a packet received for it. It returns false once the session ended.
func (c *Conn) Deliver(p []byte) bool {
	select {
	case c.receiveCh <- p:
		return true
	case <-c.doneCh:
		return false
	}
}

// Done is closed when the session ends (Traefix).
func (c *Conn) Done() <-chan struct{} {
	return c.doneCh
}

// RemoteAddr returns the address of the client of the session (Traefix).
func (c *Conn) RemoteAddr() net.Addr {
	return c.rAddr
}

// Timeout returns how long the session lasts without a packet (Traefix).
func (c *Conn) Timeout() time.Duration {
	return c.timeout
}
