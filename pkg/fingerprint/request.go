package fingerprint

import (
	"context"
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

type requestKey struct{}

// Handler attaches to each request the header block read from the wire for it.
func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if capture := captureFromContext(req.Context()); capture != nil {
			if r := capture.take(req); r != nil {
				req = req.WithContext(context.WithValue(req.Context(), requestKey{}, r))
			}
		}

		next.ServeHTTP(rw, req)
	})
}

// FromRequest returns the header block read from the wire for req, if it was captured.
func FromRequest(req *http.Request) (*Request, bool) {
	r, ok := req.Context().Value(requestKey{}).(*Request)
	return r, ok
}
