package fingerprint

import (
	"net/http"
)

// Request is a request header block as read from the wire, before net/http turns it into a map.
type Request struct {
	Method string
	// Target is the HTTP/1 request target, or the HTTP/2 :path pseudo-header.
	Target string
	// Authority is the HTTP/2 :authority pseudo-header.
	Authority  string
	ProtoMajor int
	ProtoMinor int

	// Names holds the field names in the received order and case, without the HTTP/2 pseudo-headers.
	Names []string
	// PseudoNames holds the HTTP/2 pseudo-header names in the received order.
	PseudoNames []string

	// Fields holds the fields whose value is exported, in the received order.
	Fields []Field

	// Cookies holds the values of the Cookie fields, in the received order.
	Cookies           []string
	AcceptLanguage    string
	HasAcceptLanguage bool

	// Truncated reports that the header block exceeded the capture limits,
	// in which case Names may miss the last fields.
	Truncated bool

	streamID uint32
	ja4h     string
}

// matches reports whether the request read from the wire is the one net/http handed to the handler.
func (r *Request) matches(req *http.Request) bool {
	if r.Method != req.Method || r.Target != req.RequestURI {
		return false
	}

	return r.Authority == "" || r.Authority == req.Host
}
