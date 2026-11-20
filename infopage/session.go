package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// The session groups the requests of one visitor, across connections (HTTP/2 then HTTP/3, a refresh), so that the
// page can show the timeline of each one. It lives in memory only: the container has 64 MB, so both the number of
// sessions and the requests kept per session are bounded, and a restart forgets everything.
const (
	sessionCookie      = "infopage_sid"
	sessionTTL         = 30 * time.Minute
	maxSessions        = 500
	maxSessionRequests = 30
	maxSessionField    = 512
)

// sessionRequest is what the page needs to draw the timeline of one request: the X-Traefix-Timing of its
// connection, and from X-Traefix-Transport the accept time and the position of the request on the connection.
type sessionRequest struct {
	N        int    `json:"n"`
	Time     string `json:"time"`
	Method   string `json:"method"`
	Path     string `json:"path"`
	Dest     string `json:"dest,omitempty"`
	Status   int    `json:"status"`
	Proto    string `json:"proto"`
	ConnID   string `json:"connId,omitempty"`
	Accepted string `json:"accepted,omitempty"`
	ReqSeq   string `json:"reqSeq,omitempty"`
	Transp   string `json:"transport,omitempty"`
	Timing   string `json:"timing,omitempty"`
}

type session struct {
	last     time.Time
	next     int
	requests []sessionRequest
}

type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]*session
	now      func() time.Time
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]*session), now: time.Now}
}

var sessions = newSessionStore()

// statusWriter keeps the status code of the response, which the session shows (a 304 of a revalidated asset).
type statusWriter struct {
	http.ResponseWriter

	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
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

// middleware records each request of a session. Only a page navigation opens a session: a client without cookies,
// such as curl, would otherwise open one per request and push the visitors out of the store.
func (s *sessionStore) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == "/api/session" || strings.HasPrefix(r.URL.Path, "/infopage.Infopage/") {
			next.ServeHTTP(w, r)
			return
		}

		id := sessionID(r)
		if id == "" && wantsHTML(r) {
			id = newSessionID()
			http.SetCookie(w, &http.Cookie{
				Name: sessionCookie, Value: id, Path: "/", MaxAge: int(sessionTTL / time.Second),
				HttpOnly: true, Secure: isHTTPS(r), SameSite: http.SameSiteLaxMode,
			})
		}
		if id == "" {
			next.ServeHTTP(w, r)
			return
		}

		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		if sw.status == 0 {
			sw.status = http.StatusOK
		}
		req := newSessionRequest(r, sw.status)
		s.record(id, &req)
	})
}

func newSessionRequest(r *http.Request, status int) sessionRequest {
	h := func(name string) string { return clip(r.Header.Get(prefix + name)) }
	tr := kv(h("Transport"))

	req := sessionRequest{
		Time:     h("Time"),
		Method:   r.Method,
		Path:     clip(r.URL.Path),
		Dest:     clip(r.Header.Get("Sec-Fetch-Dest")),
		Status:   status,
		Proto:    h("Proto"),
		ConnID:   h("Conn-Id"),
		Accepted: tr["accepted"],
		ReqSeq:   tr["req_seq"],
		Transp:   tr["proto"],
		Timing:   h("Timing"),
	}
	if req.Time == "" {
		req.Time = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if req.Proto == "" {
		req.Proto = r.Proto
	}
	return req
}

func (s *sessionStore) record(id string, req *sessionRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	sess := s.sessions[id]
	if sess == nil {
		s.makeRoom(now)
		sess = &session{}
		s.sessions[id] = sess
	}
	sess.last = now
	sess.next++
	req.N = sess.next
	if len(sess.requests) >= maxSessionRequests {
		sess.requests = append(sess.requests[:0], sess.requests[1:]...)
	}
	sess.requests = append(sess.requests, *req)
}

// makeRoom drops the expired sessions, and the least recently used one when the store is still full.
func (s *sessionStore) makeRoom(now time.Time) {
	if len(s.sessions) < maxSessions {
		return
	}
	var oldest string
	var oldestAt time.Time
	for id, sess := range s.sessions {
		if now.Sub(sess.last) > sessionTTL {
			delete(s.sessions, id)
			continue
		}
		if oldest == "" || sess.last.Before(oldestAt) {
			oldest, oldestAt = id, sess.last
		}
	}
	if len(s.sessions) >= maxSessions {
		delete(s.sessions, oldest)
	}
}

func (s *sessionStore) requests(id string) []sessionRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess := s.sessions[id]
	if sess == nil || s.now().Sub(sess.last) > sessionTTL {
		return []sessionRequest{}
	}
	return append([]sessionRequest(nil), sess.requests...)
}

// sessionHandler returns the requests of the session of the cookie. The identifier itself is never returned.
func sessionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	reqs := sessions.requests(sessionID(r))
	err := json.NewEncoder(w).Encode(struct {
		TTLSeconds int              `json:"ttlSeconds"`
		Max        int              `json:"max"`
		Requests   []sessionRequest `json:"requests"`
	}{int(sessionTTL / time.Second), maxSessionRequests, reqs})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func sessionID(r *http.Request) string {
	c, err := r.Cookie(sessionCookie)
	if err != nil || len(c.Value) != 22 {
		return ""
	}
	return c.Value
}

func newSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func clip(s string) string {
	if len(s) > maxSessionField {
		return s[:maxSessionField]
	}
	return s
}
