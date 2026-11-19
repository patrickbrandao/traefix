package fingerprint

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

type requestStateKey struct{}

// requestState is what the fingerprint knows of a request, filled along the handler chain.
type requestState struct {
	start   time.Time
	conn    *connInfo
	wire    *Request
	router  string
	service string

	// headerNames and headerValues are the headers the entry point set on the request.
	headerNames  []string
	headerValues []string
}

func requestStateFrom(req *http.Request) *requestState {
	st, _ := req.Context().Value(requestStateKey{}).(*requestState)
	return st
}

// SetHeaders sets on h the headers the fingerprint of the entry point set on the request of ctx,
// with the values the entry point set, so that a middleware sending a request of its own, e.g. to an authentication
// server, can send them as well.
func SetHeaders(ctx context.Context, h http.Header) {
	st, _ := ctx.Value(requestStateKey{}).(*requestState)
	if st == nil {
		return
	}
	for i, name := range st.headerNames {
		h[name] = []string{st.headerValues[i]}
	}
}

// FromRequest returns the header block read from the wire for req, if it was captured.
func FromRequest(req *http.Request) (*Request, bool) {
	if st := requestStateFrom(req); st != nil && st.wire != nil {
		return st.wire, true
	}
	return nil, false
}

// Handler is the first handler of the entry point: it removes the client headers carrying the prefix,
// adds the connection data and fingerprints as headers, and exports the request record once it is served.
func (e *EntryPoint) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		e.removeClientHeaders(req.Header)

		st := &requestState{start: time.Now(), conn: connInfoFromContext(req.Context())}
		if info := st.conn; info != nil {
			switch {
			case info.capture != nil:
				st.wire = info.capture.take(req)
			case info.h3 != nil:
				st.wire = info.h3.take(req)
				info.h3.waitSettings(maxSettingsWait)
			}
			e.finishConn(info, protoOf(req))
		}
		if st.wire != nil && st.wire.Truncated {
			e.truncated("headers")
		}

		e.setHeaders(req, st)

		req = req.WithContext(context.WithValue(req.Context(), requestStateKey{}, st))

		if e.exporter == nil {
			next.ServeHTTP(rw, req)
			return
		}

		sw := &statusWriter{ResponseWriter: rw}
		next.ServeHTTP(sw, req)
		e.exportRequest(st, req, sw.status)
	})
}

// removeClientHeaders removes the headers whose name starts with the prefix, in any case and with underscores
// in place of dashes, so that a client can never forge the values set by the entry point.
func (e *EntryPoint) removeClientHeaders(h http.Header) {
	for name := range h {
		if e.hasPrefix(name) {
			delete(h, name)
		}
	}
}

func (e *EntryPoint) hasPrefix(name string) bool {
	if len(name) < len(e.prefix) {
		return false
	}
	for i := range len(e.prefix) {
		c := name[i]
		switch {
		case c == '_':
			c = '-'
		case 'A' <= c && c <= 'Z':
			c += 'a' - 'A'
		}
		if c != e.prefix[i] {
			return false
		}
	}
	return true
}

// headerSetter assigns the headers from a single backing array, as each request gets its own values,
// which a later handler may change in place.
type headerSetter struct {
	h     http.Header
	vals  []string
	names []string
}

// setOpt sets a header with a value, and leaves out an empty one.
func (s *headerSetter) setOpt(name, value string) {
	if value != "" {
		s.set(name, value)
	}
}

// set assigns a canonical name: the names are canonicalized once, when the entry point is built.
func (s *headerSetter) set(name, value string) {
	s.vals = append(s.vals, value)
	s.names = append(s.names, name)
	n := len(s.vals)
	s.h[name] = s.vals[n-1 : n : n]
}

func (e *EntryPoint) setHeaders(req *http.Request, st *requestState) {
	s := headerSetter{h: req.Header, vals: make([]string, 0, 32), names: make([]string, 0, 32)}
	proto := protoOf(req)

	s.set(e.headers.timestamp, st.start.UTC().Format(time.RFC3339Nano))

	if info := st.conn; info != nil {
		s.set(e.headers.srcIP, info.srcIP)
		s.set(e.headers.srcPort, info.srcPort)
		s.set(e.headers.dstPort, info.dstPort)
		if info.dstIP != "" {
			s.set(e.headers.serverIP, info.dstIP)
		}
		s.set(e.headers.serverPort, info.dstPort)
		s.set(e.headers.connID, info.id)
		if info.ja3 != "" {
			s.set(e.headers.ja3, info.ja3)
		}
		if info.ja4 != "" {
			s.set(e.headers.ja4, info.ja4)
		}
		if info.alpn != "" {
			s.set(e.headers.alpn, info.alpn)
		}
		if info.ja4t != "" {
			s.set(e.headers.ja4t, info.ja4t)
		}
		if info.ja3n != "" {
			s.set(e.headers.ja3n, info.ja3n)
		}
		if info.synView != nil {
			s.set(e.headers.ttl, strconv.Itoa(info.synView.TTL))
		} else if info.quicIP.HasTTL {
			s.set(e.headers.ttl, strconv.Itoa(int(info.quicIP.TTL)))
		}
		if v := transportHeader(info, st.start, info.requests.Add(1)); v != "" {
			s.set(e.headers.transport, v)
		}
		if d := info.detail; d != nil {
			s.setOpt(e.headers.tls, d.tls)
			s.setOpt(e.headers.tlsHello, d.tlsHello)
			s.setOpt(e.headers.ja3String, d.ja3String)
			s.setOpt(e.headers.ja3nString, d.ja3nString)
			s.setOpt(e.headers.ja4r, d.ja4r)
			s.setOpt(e.headers.ja4o, d.ja4o)
			s.setOpt(e.headers.p0f, d.p0f)
			if proto == "h2" {
				s.setOpt(e.headers.h2Detail, d.h2Detail)
			}
			if proto == "h3" {
				s.setOpt(e.headers.h3, h3Detail(info))
			}
			s.setOpt(e.headers.syn, d.syn)
			s.setOpt(e.headers.clientHello, d.clientHello)
			s.set(e.headers.timing, timingHeader(info, st.start))
		}
		if proto == "h2" && info.akamai != "" {
			s.set(e.headers.h2, info.akamai)
		}
	}

	s.set(e.headers.proto, proto)

	if st.wire != nil {
		s.set(e.headers.ja4h, st.wire.JA4H())
		if e.headerDetail != detailBasic {
			s.set(e.headers.ja4hRaw, boundedList([]string{st.wire.JA4HRaw()}, "", maxDetailHeader))
			order, cookies := requestDetail(st.wire)
			s.setOpt(e.headers.headerOrder, order)
			s.setOpt(e.headers.cookieNames, cookies)
		}
	}

	st.headerNames = s.names
	st.headerValues = s.vals
}

func (e *EntryPoint) exportRequest(st *requestState, req *http.Request, status int) {
	rec := &requestRecord{
		Type:        recordRequest,
		Start:       st.start.UTC(),
		End:         time.Now().UTC(),
		EntryPoint:  e.name,
		Proto:       protoOf(req),
		Method:      req.Method,
		Authority:   req.Host,
		Path:        recordPath(req.URL.EscapedPath()),
		HTTPVersion: req.Proto,
		Status:      status,
		Router:      st.router,
		Service:     st.service,
	}
	if st.conn != nil {
		rec.ConnID = st.conn.id
	}
	if w := st.wire; w != nil {
		rec.FieldNames = w.Names
		rec.Fields = w.Fields
		rec.CookieNames = cookieNames(w.Cookies)
		rec.JA4H = w.JA4H()
		rec.HeadersTruncated = w.Truncated
	}

	e.exporter.enqueue(recordRequest, rec, rec.size())
}

func protoOf(req *http.Request) string {
	switch req.ProtoMajor {
	case 3:
		return "h3"
	case 2:
		return "h2"
	default:
		return "h1"
	}
}

// WrapRouterHandler records the router and the service serving the request, the innermost router winning.
func WrapRouterHandler(router, service string) func(http.Handler) (http.Handler, error) {
	return func(next http.Handler) (http.Handler, error) {
		if !Enabled() {
			return next, nil
		}

		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if st := requestStateFrom(req); st != nil {
				st.router = router
				st.service = service
				st.observe(req)
			}
			next.ServeHTTP(rw, req)
		}), nil
	}
}

// statusWriter records the status code of the response.
type statusWriter struct {
	http.ResponseWriter

	status int
}

func (w *statusWriter) WriteHeader(code int) {
	// The informational responses precede the final one, except for the switch of protocol.
	if w.status == 0 && (code >= http.StatusOK || code == http.StatusSwitchingProtocols) {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// Fingerprints are the fingerprints of a request. A fingerprint is empty when it was not computed,
// e.g. JA4 over plain HTTP, or JA4H over HTTP/3.
type Fingerprints struct {
	JA3   string
	JA4   string
	JA4H  string
	HTTP2 string
	JA4T  string
}

// FingerprintsFromRequest returns the fingerprints of req, and false when its entry point has no fingerprint option.
func FingerprintsFromRequest(req *http.Request) (Fingerprints, bool) {
	st := requestStateFrom(req)
	if st == nil {
		return Fingerprints{}, false
	}

	var f Fingerprints
	if c := st.conn; c != nil {
		f.JA3, f.JA4, f.JA4T = c.ja3, c.ja4, c.ja4t
		if req.ProtoMajor == 2 {
			f.HTTP2 = c.akamai
		}
	}
	if st.wire != nil {
		f.JA4H = st.wire.JA4H()
	}

	return f, true
}

// ContextWithFingerprints returns a context carrying the given fingerprints, as the entry point sets them,
// for the handlers served outside an entry point, such as in tests.
func ContextWithFingerprints(ctx context.Context, f Fingerprints) context.Context {
	st := &requestState{start: time.Now(), conn: &connInfo{ja3: f.JA3, ja4: f.JA4, akamai: f.HTTP2, ja4t: f.JA4T}}
	if f.JA4H != "" {
		st.wire = &Request{ja4h: f.JA4H}
	}
	return context.WithValue(ctx, requestStateKey{}, st)
}
