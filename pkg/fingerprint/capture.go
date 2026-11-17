package fingerprint

import (
	"bytes"
	"net/http"
	"sync"

	"github.com/rs/zerolog/log"
)

const (
	// maxH2Prefix bounds the raw HTTP/2 bytes kept from the end of the preface to the end of the first header block.
	maxH2Prefix = 16 << 10

	// maxPendingRequests bounds the requests parsed from the wire and not yet claimed by a handler,
	// e.g. the streams refused by the server or reset by the client.
	maxPendingRequests = 512
)

var h2Preface = []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")

type captureState uint8

const (
	stateDetect captureState = iota
	stateH1
	stateH2
	stateStopped
)

// Capture holds what is observed on a single connection.
type Capture struct {
	mu sync.Mutex

	state  captureState
	detect []byte

	h1 *h1Parser
	h2 *h2Parser

	// pending holds the parsed requests in wire order until a handler claims them.
	pending []*Request

	h2Prefix          []byte
	h2PrefixDone      bool
	h2PrefixTruncated bool
	firstPseudoNames  []string

	cfg Config
}

func newCapture(cfg Config) *Capture {
	return &Capture{cfg: cfg}
}

// Proto returns the protocol detected on the connection: "h1", "h2", or an empty string while undetermined.
func (c *Capture) Proto() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch {
	case c.h2 != nil:
		return "h2"
	case c.h1 != nil:
		return "h1"
	default:
		return ""
	}
}

// H2Prefix returns the raw HTTP/2 bytes from the end of the client preface to the end of the first header block,
// and whether they were truncated. It returns nil until the first header block is complete.
func (c *Capture) H2Prefix() ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.h2PrefixDone {
		return nil, false
	}

	return c.h2Prefix, c.h2PrefixTruncated
}

// FirstPseudoNames returns the pseudo-header names of the first HTTP/2 request, in the received order.
func (c *Capture) FirstPseudoNames() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.firstPseudoNames
}

func (c *Capture) releaseH2Prefix() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.h2PrefixDone {
		c.h2Prefix = nil
	}
}

func (c *Capture) observe(p []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == stateStopped {
		return
	}

	// The capture must never take the connection down: a parser bug stops the capture instead.
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Msg("Stopping connection fingerprint capture")
			c.stop()
		}
	}()

	if c.state == stateDetect {
		p = c.detectProto(p)
		if len(p) == 0 {
			return
		}
	}

	var err error
	switch c.state {
	case stateH1:
		err = c.h1.feed(p, c.addRequest)
	case stateH2:
		err = c.feedH2(p)
	}

	if err != nil {
		log.Debug().Err(err).Msg("Stopping connection fingerprint capture")
		c.stop()
	}
}

// detectProto buffers the first bytes until the HTTP/2 preface can be told apart from an HTTP/1 request.
// It returns the bytes that remain to be parsed once the protocol is known.
func (c *Capture) detectProto(p []byte) []byte {
	need := min(len(h2Preface)-len(c.detect), len(p))
	c.detect = append(c.detect, p[:need]...)
	p = p[need:]

	if !bytes.HasPrefix(h2Preface, c.detect) {
		c.state = stateH1
		c.h1 = newH1Parser(c.cfg.fields)
		buffered := c.detect
		c.detect = nil
		if err := c.h1.feed(buffered, c.addRequest); err != nil {
			c.stop()
			return nil
		}
		return p
	}

	if len(c.detect) < len(h2Preface) {
		return nil
	}

	c.state = stateH2
	c.h2 = newH2Parser(c.cfg.MaxDecoderHeaderTableSize, c.cfg.fields)
	c.detect = nil

	return p
}

func (c *Capture) feedH2(p []byte) error {
	if c.h2PrefixDone {
		return c.h2.feed(p, c.addRequest, c.resetStream)
	}

	// The parser reports how many bytes it consumed up to the end of the first header block,
	// so that the prefix stops exactly there.
	c.h2.onFirstBlockEnd = func(consumed int) {
		c.appendH2Prefix(p[:consumed])
		c.h2PrefixDone = true
	}
	err := c.h2.feed(p, c.addRequest, c.resetStream)
	c.h2.onFirstBlockEnd = nil

	if !c.h2PrefixDone {
		c.appendH2Prefix(p)
	}

	return err
}

func (c *Capture) appendH2Prefix(p []byte) {
	room := maxH2Prefix - len(c.h2Prefix)
	if len(p) > room {
		p = p[:room]
		c.h2PrefixTruncated = true
	}
	c.h2Prefix = append(c.h2Prefix, p...)
}

func (c *Capture) addRequest(r *Request) {
	if c.h2 != nil && c.firstPseudoNames == nil {
		c.firstPseudoNames = r.PseudoNames
	}
	if len(c.pending) >= maxPendingRequests {
		c.pending[0] = nil
		c.pending = c.pending[1:]
	}
	c.pending = append(c.pending, r)
}

func (c *Capture) resetStream(id uint32) {
	for i, r := range c.pending {
		if r.streamID == id {
			c.pending = append(c.pending[:i], c.pending[i+1:]...)
			return
		}
	}
}

func (c *Capture) stop() {
	c.state = stateStopped
	c.detect = nil
	c.h1 = nil
	if c.h2 != nil {
		c.h2.close()
	}
	c.pending = nil
}

// take returns the parsed request matching req, and forgets it.
func (c *Capture) take(req *http.Request) *Request {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, r := range c.pending {
		if !r.matches(req) {
			continue
		}

		if c.state == stateH1 {
			// HTTP/1 requests are served in order: the requests before the match were not handed to a handler.
			clear(c.pending[:i])
			c.pending = c.pending[i+1:]
		} else {
			c.pending = append(c.pending[:i], c.pending[i+1:]...)
		}

		return r
	}

	return nil
}
