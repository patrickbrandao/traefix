package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	hdrProto     = "X-Traefix-Proto"
	hdrConnID    = "X-Traefix-Conn-Id"
	hdrTransport = "X-Traefix-Transport"
	connA        = "01CONNA"
)

type sessionReply struct {
	Requests []sessionRequest `json:"requests"`
}

func sessionGet(t *testing.T, h http.Handler, target, accept string, cookie *http.Cookie, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, http.NoBody)
	req.Header.Set("Accept", accept)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	return rw
}

func readSession(t *testing.T, h http.Handler, cookie *http.Cookie) []sessionRequest {
	t.Helper()
	rw := sessionGet(t, h, "/api/session", "*/*", cookie, nil)
	if strings.Contains(rw.Body.String(), cookie.Value) {
		t.Fatal("the session identifier leaks in the reply")
	}
	var reply sessionReply
	if err := json.Unmarshal(rw.Body.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	return reply.Requests
}

func newSessionMux(store *sessionStore) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/session", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(sessionReply{Requests: store.requests(sessionID(r))})
	})
	mux.HandleFunc("/assets/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotModified) })
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	return store.middleware(mux)
}

func TestSessionRecordsTheRequestsOfTheVisitor(t *testing.T) {
	store := newSessionStore()
	h := newSessionMux(store)

	page := sessionGet(t, h, "/", "text/html", nil, map[string]string{
		hdrProto:           "h2",
		hdrConnID:          connA,
		hdrTransport:       "proto=tcp,ipv=4,accepted=2026-11-20T13:29:10.510652Z,req_seq=1",
		"X-Traefix-Timing": "src=kernel,proto=tcp,tcp_rtt_us=36800,est_to_first_us=44,req_us=48700",
		"Sec-Fetch-Dest":   "document",
	})
	cookies := page.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookie || !cookies[0].HttpOnly {
		t.Fatalf("cookie: %+v", cookies)
	}
	cookie := cookies[0]

	sessionGet(t, h, "/assets/app.js", "*/*", cookie, map[string]string{
		hdrProto: "h2", hdrConnID: connA, hdrTransport: "proto=tcp,req_seq=2", "Sec-Fetch-Dest": "script",
	})
	sessionGet(t, h, "/", "text/html", cookie, map[string]string{
		hdrProto: "h3", hdrConnID: "01CONNB", hdrTransport: "proto=udp,req_seq=1",
	})

	reqs := readSession(t, h, cookie)
	if len(reqs) != 3 {
		t.Fatalf("requests: %+v", reqs)
	}
	checkSessionRequests(t, &reqs[0], &reqs[1], &reqs[2])
}

func checkSessionRequests(t *testing.T, first, asset, refresh *sessionRequest) {
	t.Helper()
	if first.N != 1 || first.Proto != "h2" || first.Accepted != "2026-11-20T13:29:10.510652Z" || first.ReqSeq != "1" || !strings.Contains(first.Timing, "req_us=48700") {
		t.Errorf("first: %+v", first)
	}
	if asset.Status != http.StatusNotModified || asset.ReqSeq != "2" || asset.Dest != "script" || asset.ConnID != connA {
		t.Errorf("asset: %+v", asset)
	}
	if refresh.Proto != "h3" || refresh.Transp != "udp" || refresh.ConnID != "01CONNB" {
		t.Errorf("refresh: %+v", refresh)
	}
}

func TestSessionNotOpenedWithoutNavigation(t *testing.T) {
	store := newSessionStore()
	h := newSessionMux(store)

	rw := sessionGet(t, h, "/", "*/*", nil, nil)
	if len(rw.Result().Cookies()) != 0 || len(store.sessions) != 0 {
		t.Fatalf("curl opened a session: %v", rw.Result().Cookies())
	}

	forged := &http.Cookie{Name: sessionCookie, Value: "short"}
	sessionGet(t, h, "/assets/app.js", "*/*", forged, nil)
	if len(store.sessions) != 0 {
		t.Fatal("a malformed cookie opened a session")
	}
}

func TestSessionBounds(t *testing.T) {
	store := newSessionStore()
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	id := newSessionID()
	for i := range maxSessionRequests + 5 {
		store.record(id, &sessionRequest{Path: "/" + strconv.Itoa(i)})
	}
	reqs := store.requests(id)
	if len(reqs) != maxSessionRequests || reqs[0].N != 6 || reqs[len(reqs)-1].N != maxSessionRequests+5 {
		t.Fatalf("kept %d requests, from #%d", len(reqs), reqs[0].N)
	}

	for range maxSessions + 10 {
		now = now.Add(time.Millisecond)
		store.record(newSessionID(), &sessionRequest{})
	}
	if len(store.sessions) > maxSessions {
		t.Fatalf("%d sessions", len(store.sessions))
	}
	if len(store.requests(id)) != 0 {
		t.Error("the least recently used session survived")
	}

	other := newSessionID()
	store.record(other, &sessionRequest{})
	now = now.Add(sessionTTL + time.Second)
	if len(store.requests(other)) != 0 {
		t.Error("an expired session is still served")
	}
}
