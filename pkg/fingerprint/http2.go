package fingerprint

import (
	"encoding/binary"
	"errors"
	"fmt"

	"golang.org/x/net/http2/hpack"
)

const (
	h2FrameHeaderLen = 9

	h2FrameHeaders      = 0x1
	h2FrameRSTStream    = 0x3
	h2FrameContinuation = 0x9

	h2FlagEndHeaders = 0x4
	h2FlagPadded     = 0x8
	h2FlagPriority   = 0x20

	// defaultHeaderTableSize is the HPACK dynamic table size of RFC 9113 section 6.5.2,
	// also used by net/http when MaxDecoderHeaderTableSize is not set.
	defaultHeaderTableSize = 4096

	// maxH2HeaderBlock bounds the bytes of a single header block the capture decodes.
	// net/http rejects far smaller header lists, so a longer block only comes from a client being cut off.
	maxH2HeaderBlock = 1 << 20

	// maxH2Fields bounds the fields kept from a single header block.
	maxH2Fields = 256
)

// h2Parser follows the HTTP/2 frames read by the server after the client preface,
// and decodes the header blocks with its own HPACK decoder, kept in step with the one of the server.
type h2Parser struct {
	hdr  [h2FrameHeaderLen]byte
	hdrN int

	frameType byte
	flags     byte
	streamID  uint32
	length    int
	// remaining is the count of payload bytes of the current frame left to read.
	remaining int

	// preamble holds the HEADERS payload bytes preceding the header block fragment: pad length and priority.
	preamble     [6]byte
	preambleN    int
	preambleNeed int
	fragmentLeft int

	dec        *hpack.Decoder
	cur        *Request
	blockBytes int

	firstBlockDone bool
	// onFirstBlockEnd is called with the count of bytes of the current feed call consumed up to the end of the
	// first header block.
	onFirstBlockEnd func(consumed int)
}

func newH2Parser(maxTableSize uint32) *h2Parser {
	if maxTableSize == 0 {
		maxTableSize = defaultHeaderTableSize
	}

	p := &h2Parser{}
	p.dec = hpack.NewDecoder(maxTableSize, p.onField)

	return p
}

func (p *h2Parser) close() {
	p.dec = nil
	p.cur = nil
}

func (p *h2Parser) onField(f hpack.HeaderField) {
	r := p.cur
	if r == nil {
		// Trailers and blocks past the limits are decoded to keep the HPACK state, but not recorded.
		return
	}

	if len(f.Name) > 0 && f.Name[0] == ':' {
		r.PseudoNames = append(r.PseudoNames, f.Name)
		switch f.Name {
		case ":method":
			r.Method = f.Value
		case ":path":
			r.Target = f.Value
		case ":authority":
			r.Authority = f.Value
		}
		return
	}

	if len(r.Names) >= maxH2Fields {
		r.Truncated = true
		return
	}
	r.Names = append(r.Names, f.Name)

	switch f.Name {
	case "cookie":
		r.Cookies = append(r.Cookies, f.Value)
	case "accept-language":
		if !r.HasAcceptLanguage {
			r.AcceptLanguage = f.Value
			r.HasAcceptLanguage = true
		}
	}
}

func (p *h2Parser) feed(b []byte, emit func(*Request), reset func(uint32)) error {
	total := len(b)

	for len(b) > 0 {
		if p.hdrN < h2FrameHeaderLen {
			n := copy(p.hdr[p.hdrN:], b)
			p.hdrN += n
			b = b[n:]
			if p.hdrN < h2FrameHeaderLen {
				return nil
			}
			if err := p.startFrame(reset); err != nil {
				return err
			}
			if p.remaining == 0 {
				if err := p.endFrame(emit, total-len(b)); err != nil {
					return err
				}
			}
			continue
		}

		n := min(p.remaining, len(b))
		if err := p.payload(b[:n]); err != nil {
			return err
		}
		b = b[n:]
		p.remaining -= n

		if p.remaining == 0 {
			if err := p.endFrame(emit, total-len(b)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *h2Parser) startFrame(reset func(uint32)) error {
	length := int(p.hdr[0])<<16 | int(p.hdr[1])<<8 | int(p.hdr[2])
	p.frameType = p.hdr[3]
	p.flags = p.hdr[4]
	p.streamID = binary.BigEndian.Uint32(p.hdr[5:]) & 0x7fffffff
	p.length = length
	p.remaining = length

	switch p.frameType {
	case h2FrameHeaders:
		p.preambleN = 0
		p.preambleNeed = 0
		if p.flags&h2FlagPadded != 0 {
			p.preambleNeed++
		}
		if p.flags&h2FlagPriority != 0 {
			p.preambleNeed += 5
		}
		if p.preambleNeed > length {
			return errors.New("HEADERS frame too short")
		}
		p.fragmentLeft = length - p.preambleNeed

		p.blockBytes = 0
		p.cur = &Request{ProtoMajor: 2, streamID: p.streamID}

	case h2FrameContinuation:
		if p.streamID == 0 {
			return errors.New("CONTINUATION frame on stream 0")
		}

	case h2FrameRSTStream:
		reset(p.streamID)
	}

	return nil
}

func (p *h2Parser) payload(b []byte) error {
	switch p.frameType {
	case h2FrameHeaders:
		if p.preambleN < p.preambleNeed {
			n := copy(p.preamble[p.preambleN:p.preambleNeed], b)
			p.preambleN += n
			b = b[n:]
			if p.preambleN < p.preambleNeed {
				return nil
			}

			padding := 0
			if p.flags&h2FlagPadded != 0 {
				padding = int(p.preamble[0])
			}
			p.fragmentLeft = p.length - p.preambleNeed - padding
			if p.fragmentLeft < 0 {
				return errors.New("HEADERS frame padding too long")
			}
		}

		// The bytes past the header block fragment are the padding.
		n := min(p.fragmentLeft, len(b))
		p.fragmentLeft -= n
		return p.fragment(b[:n])

	case h2FrameContinuation:
		return p.fragment(b)
	}

	return nil
}

func (p *h2Parser) fragment(b []byte) error {
	if len(b) == 0 || p.dec == nil {
		return nil
	}

	p.blockBytes += len(b)
	if p.blockBytes > maxH2HeaderBlock {
		return fmt.Errorf("header block longer than %d bytes", maxH2HeaderBlock)
	}

	if _, err := p.dec.Write(b); err != nil {
		return fmt.Errorf("decoding header block: %w", err)
	}

	return nil
}

func (p *h2Parser) endFrame(emit func(*Request), consumed int) error {
	p.hdrN = 0

	isBlock := p.frameType == h2FrameHeaders || p.frameType == h2FrameContinuation
	if !isBlock || p.flags&h2FlagEndHeaders == 0 {
		return nil
	}

	if err := p.dec.Close(); err != nil {
		return fmt.Errorf("decoding header block: %w", err)
	}

	r := p.cur
	p.cur = nil
	// A header block without :method is a trailer section.
	if r != nil && r.Method != "" {
		emit(r)
	}

	if !p.firstBlockDone {
		p.firstBlockDone = true
		if p.onFirstBlockEnd != nil {
			p.onFirstBlockEnd(consumed)
		}
	}

	return nil
}
