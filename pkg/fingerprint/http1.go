package fingerprint

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
)

const (
	// maxH1Line bounds the bytes kept from a single line; the rest of a longer line is skipped.
	maxH1Line = 8 << 10

	// maxH1Names bounds the field names kept from a single header block.
	maxH1Names = 256
)

type h1State uint8

const (
	h1RequestLine h1State = iota
	h1Headers
	h1Body
	h1ChunkSize
	h1ChunkData
	h1ChunkDataEnd
	h1Trailers
)

// h1Parser follows the HTTP/1 request stream read by the server: it records each header block,
// and skips the bodies according to their framing so that the next header block is found.
type h1Parser struct {
	state h1State

	line         []byte
	lineOverflow bool

	remaining int64

	cur           *Request
	contentLength int64
	chunked       bool

	// The strings of the current header block are gathered in arena, and cut from a single string at its end,
	// which saves an allocation per field.
	arena          []byte
	method, target span
	names          []span
	fields         []fieldSpan
	cookies        []span
	language       span

	matcher *fieldMatcher
}

// span locates a string in the arena of the parser.
type span struct {
	start, end int
}

type fieldSpan struct {
	name  int
	value span
}

// maxH1Arena is the arena capacity kept between two header blocks.
const maxH1Arena = 16 << 10

func (p *h1Parser) keep(b []byte) span {
	start := len(p.arena)
	p.arena = append(p.arena, b...)
	return span{start: start, end: len(p.arena)}
}

func newH1Parser(fields *fieldMatcher) *h1Parser {
	return &h1Parser{matcher: fields}
}

func (p *h1Parser) feed(b []byte, emit func(*Request)) error {
	for len(b) > 0 {
		switch p.state {
		case h1Body, h1ChunkData:
			n := min(int64(len(b)), p.remaining)
			b = b[n:]
			p.remaining -= n
			if p.remaining == 0 {
				if p.state == h1Body {
					p.state = h1RequestLine
				} else {
					p.state = h1ChunkDataEnd
				}
			}

		default:
			i := bytes.IndexByte(b, '\n')
			if i < 0 {
				p.appendLine(b)
				return nil
			}
			p.appendLine(b[:i])
			b = b[i+1:]

			line := p.line
			if n := len(line); n > 0 && line[n-1] == '\r' {
				line = line[:n-1]
			}
			overflow := p.lineOverflow
			p.lineOverflow = false

			err := p.processLine(line, overflow, emit)

			// A long line leaves a large buffer behind, which the next, usually short, lines do not need.
			if cap(p.line) > 1024 {
				p.line = nil
			} else {
				p.line = p.line[:0]
			}

			if err != nil {
				return err
			}
			continue
		}
	}

	return nil
}

func (p *h1Parser) appendLine(b []byte) {
	room := maxH1Line - len(p.line)
	if len(b) > room {
		b = b[:room]
		p.lineOverflow = true
	}
	p.line = append(p.line, b...)
}

func (p *h1Parser) processLine(line []byte, overflow bool, emit func(*Request)) error {
	switch p.state {
	case h1RequestLine:
		// A server ignores the empty lines preceding a request line (RFC 9112 section 2.2).
		if len(line) == 0 {
			return nil
		}
		if overflow {
			return errors.New("request line too long")
		}
		return p.startRequest(line)

	case h1Headers:
		if len(line) == 0 {
			p.endHeaders(emit)
			return nil
		}
		p.addField(line, overflow)
		return nil

	case h1ChunkSize:
		size, _, _ := bytes.Cut(line, []byte(";"))
		n, err := strconv.ParseInt(string(bytes.TrimSpace(size)), 16, 64)
		if err != nil || n < 0 {
			return errors.New("invalid chunk size")
		}
		if n == 0 {
			p.state = h1Trailers
			return nil
		}
		p.remaining = n
		p.state = h1ChunkData
		return nil

	case h1ChunkDataEnd:
		if len(line) != 0 {
			return errors.New("invalid chunk end")
		}
		p.state = h1ChunkSize
		return nil

	case h1Trailers:
		if len(line) == 0 {
			p.state = h1RequestLine
		}
		return nil
	}

	return nil
}

func (p *h1Parser) startRequest(line []byte) error {
	method, rest, ok1 := bytes.Cut(line, []byte(" "))
	target, version, ok2 := bytes.Cut(rest, []byte(" "))
	if !ok1 || !ok2 || len(method) == 0 || len(target) == 0 {
		return errors.New("invalid request line")
	}

	major, minor, ok := parseHTTPVersion(version)
	if !ok {
		return errors.New("invalid HTTP version")
	}

	if cap(p.arena) > maxH1Arena {
		p.arena = nil
	}
	p.arena = p.arena[:0]
	p.names = p.names[:0]
	p.fields = p.fields[:0]
	p.cookies = p.cookies[:0]
	p.language = span{}
	p.method = p.keep(method)
	p.target = p.keep(target)

	p.cur = &Request{
		ProtoMajor: major,
		ProtoMinor: minor,
	}
	p.contentLength = 0
	p.chunked = false
	p.state = h1Headers

	return nil
}

func parseHTTPVersion(v []byte) (int, int, bool) {
	if len(v) != len("HTTP/1.1") || !bytes.HasPrefix(v, []byte("HTTP/")) || v[6] != '.' {
		return 0, 0, false
	}
	if v[5] < '0' || v[5] > '9' || v[7] < '0' || v[7] > '9' {
		return 0, 0, false
	}

	return int(v[5] - '0'), int(v[7] - '0'), true
}

func (p *h1Parser) addField(line []byte, overflow bool) {
	r := p.cur
	if overflow {
		r.Truncated = true
	}

	// An obsolete line folding continues the value of the previous field, and names no field.
	if line[0] == ' ' || line[0] == '\t' {
		return
	}

	name, value, ok := bytes.Cut(line, []byte(":"))
	if !ok {
		return
	}
	value = bytes.Trim(value, " \t")

	if len(p.names) < maxH1Names {
		p.names = append(p.names, p.keep(name))
		if p.matcher.matchBytes(name) {
			p.fields = append(p.fields, fieldSpan{name: len(p.names) - 1, value: p.keep(value[:min(len(value), maxFieldValue)])})
		}
	} else {
		r.Truncated = true
	}

	switch {
	case asciiEqualFold(name, "cookie"):
		p.cookies = append(p.cookies, p.keep(value))
	case asciiEqualFold(name, "accept-language"):
		if !r.HasAcceptLanguage {
			p.language = p.keep(value)
			r.HasAcceptLanguage = true
		}
	case asciiEqualFold(name, "content-length"):
		if n, err := strconv.ParseInt(string(value), 10, 64); err == nil && n >= 0 && p.contentLength == 0 {
			p.contentLength = n
		}
	case asciiEqualFold(name, "transfer-encoding"):
		// The server only accepts chunked as the final transfer coding of a request.
		codings := strings.Split(string(value), ",")
		p.chunked = strings.EqualFold(strings.TrimSpace(codings[len(codings)-1]), "chunked")
	}
}

func (p *h1Parser) endHeaders(emit func(*Request)) {
	r := p.cur
	all := string(p.arena)
	str := func(sp span) string { return all[sp.start:sp.end] }

	r.Method = str(p.method)
	r.Target = str(p.target)
	if len(p.names) > 0 {
		r.Names = make([]string, len(p.names))
		for i, sp := range p.names {
			r.Names[i] = str(sp)
		}
	}
	if len(p.fields) > 0 {
		r.Fields = make([]Field, len(p.fields))
		for i, f := range p.fields {
			r.Fields[i] = Field{Name: r.Names[f.name], Value: str(f.value)}
		}
	}
	if len(p.cookies) > 0 {
		r.Cookies = make([]string, len(p.cookies))
		for i, sp := range p.cookies {
			r.Cookies[i] = str(sp)
		}
	}
	if r.HasAcceptLanguage {
		r.AcceptLanguage = str(p.language)
	}

	emit(r)
	p.cur = nil

	switch {
	case p.chunked:
		p.state = h1ChunkSize
	case p.contentLength > 0:
		p.remaining = p.contentLength
		p.state = h1Body
	default:
		p.state = h1RequestLine
	}
}

func asciiEqualFold(b []byte, s string) bool {
	if len(b) != len(s) {
		return false
	}
	for i := range b {
		c := b[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != s[i] {
			return false
		}
	}

	return true
}
