package mesh

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"maps"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/fingerprint"
	"github.com/traefik/traefik/v3/pkg/middlewares/accesslog"
	"github.com/traefik/traefik/v3/pkg/tcp"
	"golang.org/x/net/http/httpguts"
)

const (
	// headerForward carries, on a request an instance forwards, what the instance that owns the router needs to serve it.
	// It only means something on a link of the mesh, which only the instances of the cluster open.
	headerForward = "Traefix-Mesh"
	// headerForwardError tells, on an answer, that the instance that owns the router did not serve the request,
	// as opposed to the service of the router answering with the same status.
	headerForwardError = "Traefix-Mesh-Error"

	errorUnknownRouter = "unknown-router"

	forwardHTTP    = "http"
	forwardUpgrade = "upgrade"

	// forwardFlushInterval is the one of the services of Traefik: a response that trickles reaches the client as it comes.
	forwardFlushInterval = 100 * time.Millisecond

	resultForwarded   = "forwarded"
	resultError       = "error"
	resultUnavailable = "unavailable"
	resultServed      = "served"
	resultUnknown     = "unknown"
)

// forwardInfo is what an instance tells the instance that owns a router about a request it forwards.
type forwardInfo struct {
	Version int    `json:"v"`
	Kind    string `json:"kind"`
	// Router is the name of the router on the instance that owns it.
	Router string `json:"router"`
	// Remote is the address of the client, as the entry point saw it.
	Remote string `json:"remote,omitempty"`
	// Local is the address the client connected to, for the TCP routers.
	Local string      `json:"local,omitempty"`
	TLS   *forwardTLS `json:"tls,omitempty"`
	// Fingerprint is what the entry point measured of the connection and of the request.
	Fingerprint *forwardFingerprint `json:"fingerprint,omitempty"`
}

// forwardTLS is the TLS connection of the client, which ends at the entry point.
type forwardTLS struct {
	Version     uint16 `json:"version,omitempty"`
	CipherSuite uint16 `json:"cipherSuite,omitempty"`
	ServerName  string `json:"serverName,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
}

type forwardFingerprint struct {
	ConnID string `json:"connID,omitempty"`
	JA3    string `json:"ja3,omitempty"`
	JA4    string `json:"ja4,omitempty"`
	JA4H   string `json:"ja4h,omitempty"`
	HTTP2  string `json:"http2,omitempty"`
	JA4T   string `json:"ja4t,omitempty"`
}

func (i forwardInfo) encode() string {
	data, err := json.Marshal(i)
	if err != nil {
		return ""
	}

	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeForwardInfo(value string) (forwardInfo, error) {
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return forwardInfo{}, err
	}

	var info forwardInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return forwardInfo{}, err
	}

	if info.Router == "" {
		return forwardInfo{}, errors.New("no router")
	}

	return info, nil
}

// newForwardInfo describes a request of a client for the instance that owns its router.
func newForwardInfo(req *http.Request, kind, router string) forwardInfo {
	info := forwardInfo{Version: 1, Kind: kind, Router: router, Remote: req.RemoteAddr}

	if req.TLS != nil {
		info.TLS = &forwardTLS{
			Version:     req.TLS.Version,
			CipherSuite: req.TLS.CipherSuite,
			ServerName:  req.TLS.ServerName,
			Protocol:    req.TLS.NegotiatedProtocol,
		}
	}

	if measured, ok := fingerprint.ForwardedFromRequest(req); ok {
		info.Fingerprint = &forwardFingerprint{
			ConnID: measured.ConnID,
			JA3:    measured.Fingerprints.JA3,
			JA4:    measured.Fingerprints.JA4,
			JA4H:   measured.Fingerprints.JA4H,
			HTTP2:  measured.Fingerprints.HTTP2,
			JA4T:   measured.Fingerprints.JA4T,
		}
	}

	return info
}

// forwardTarget is where the requests of a route go: the instances that own its router, this one included when it does.
type forwardTarget struct {
	kind string
	// router is the name of the router on the instances that own it.
	router string
	// candidates are the other instances that own the router.
	candidates []forwardCandidate
	// local is true when this instance owns the router too.
	local         bool
	localPriority int
}

type forwardCandidate struct {
	nodeID   string
	priority int
	// down is true when the instance says that the service of its router has no server to answer.
	down bool
}

// key names the route among the ones of every kind: a TCP router may have the name of an HTTP one.
func (t *forwardTarget) key() string {
	return routeKey(t.kind, t.router)
}

// forwardTables are the routes of the mesh as the requests need them. They are replaced as a whole when a route changes.
type forwardTables struct {
	// targets are the HTTP routes this instance does not have, by the name of their service.
	targets map[string]*forwardTarget
	// shared are the HTTP routes this instance has and other instances have too, by the name of the router.
	shared map[string]*forwardTarget
	// tcpTargets and tcpShared are the same for the TCP routes.
	tcpTargets map[string]*forwardTarget
	tcpShared  map[string]*forwardTarget
}

func newForwardTables() *forwardTables {
	return &forwardTables{
		targets:    map[string]*forwardTarget{},
		shared:     map[string]*forwardTarget{},
		tcpTargets: map[string]*forwardTarget{},
		tcpShared:  map[string]*forwardTarget{},
	}
}

func (t *forwardTables) targetsOf(kind string) map[string]*forwardTarget {
	if kind == KindTCP {
		return t.tcpTargets
	}

	return t.targets
}

func (t *forwardTables) sharedOf(kind string) map[string]*forwardTarget {
	if kind == KindTCP {
		return t.tcpShared
	}

	return t.shared
}

// choice is a place to serve a request: this instance, or another one.
type choice struct {
	nodeID   string
	local    bool
	priority int
}

// errNotServed is the answer of an instance that did not serve the request: it does not have the router any more.
var errNotServed = errors.New("the instance did not serve the request")

// meshService forwards the requests of a router learned from the mesh to an instance that owns it.
type meshService struct {
	mesh *Mesh
	// name is the name of the service, which is the one of the router of this instance.
	name string
}

func (s *meshService) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	target := s.mesh.tables.Load().targets[s.name]
	if target == nil {
		s.mesh.refuse(rw, req, "", http.StatusServiceUnavailable, "no instance of the mesh announces the router")
		return
	}

	if data := accesslog.GetLogData(req); data != nil {
		data.Core[accesslog.ServiceName] = s.name
	}

	s.mesh.dispatch(rw, req, target, nil)
}

// sharedRoute decides, for a router that this instance and other instances have, where a request is served.
// It sits between the middlewares of the entry point, which ran, and the middlewares of the router, which run where
// the request is served.
type sharedRoute struct {
	mesh *Mesh
	name string
	// local runs the middlewares of the router and its service on this instance.
	local http.Handler
}

func (s *sharedRoute) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Most routers live on one instance: nothing is decided for them.
	target := s.mesh.tables.Load().shared[s.name]
	if target == nil {
		s.local.ServeHTTP(rw, req)
		return
	}

	s.mesh.dispatch(rw, req, target, s.local)
}

// dispatch serves a request where the priorities say, and falls back to the next place when one cannot take it.
func (m *Mesh) dispatch(rw http.ResponseWriter, req *http.Request, target *forwardTarget, local http.Handler) {
	m.try(rw, req, target, local, m.plan(target, clientIP(req)))
}

// try serves a request at the first of the places, and at the following ones for as long as nothing of it was sent.
func (m *Mesh) try(rw http.ResponseWriter, req *http.Request, target *forwardTarget, local http.Handler, choices []choice) {
	if len(choices) == 0 {
		m.refuse(rw, req, "", http.StatusServiceUnavailable, "no instance that owns the router can be reached")
		return
	}

	next, rest := choices[0], choices[1:]

	if next.local {
		local.ServeHTTP(rw, req)
		return
	}

	nodeID := next.nodeID

	session := m.sessionTo(nodeID)
	if session == nil {
		// The link went down since the places were chosen.
		m.try(rw, req, target, local, rest)
		return
	}

	if data := accesslog.GetLogData(req); data != nil {
		data.Core[accesslog.ServiceURL] = "mesh://" + nodeID
		data.Core[accesslog.ServiceAddr] = nodeID
	}

	if isUpgrade(req) {
		if retry := m.forwardUpgrade(rw, req, session, nodeID, target); retry {
			m.try(rw, req, target, local, rest)
		}

		return
	}

	info := newForwardInfo(req, forwardHTTP, target.router)

	// The client of the link tells when the request starts to leave this instance: until then, another place can serve it.
	var sent atomic.Bool

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
		WroteHeaders: func() { sent.Store(true) },
	}))

	proxy := &httputil.ReverseProxy{
		Transport:     session,
		FlushInterval: forwardFlushInterval,
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme = "https"
			pr.Out.URL.Host = pr.In.Host
			pr.Out.Host = pr.In.Host

			// The entry point already settled these, from what it trusts of the client.
			// The instance that owns the router adds the address of the client when it sends the request to the server.
			for _, name := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto"} {
				if values, ok := pr.In.Header[name]; ok {
					pr.Out.Header[name] = values
				}
			}

			pr.Out.Header.Set(headerForward, info.encode())
		},
		ModifyResponse: func(resp *http.Response) error {
			// The instance answered, but not with the router: its answer is not the one of the service.
			if cause := resp.Header.Get(headerForwardError); cause != "" {
				return fmt.Errorf("%w (%s): %s", errNotServed, nodeID, cause)
			}

			m.succeeded(target.key(), nodeID)
			m.countRequest(nodeID, forwardHTTP, resultForwarded)

			return nil
		},
		ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
			m.failed(target.key(), nodeID)
			m.countRequest(nodeID, forwardHTTP, resultError)

			// Nothing of the request left this instance, e.g. the connection was just closed. Or the other instance told
			// that it did nothing with the request, which has no body that was consumed on the way.
			// In both cases another place can serve it. Any other failure may have reached the server.
			notSent := !sent.Load()
			notServed := errors.Is(err, errNotServed) && hasNoBody(req)

			if (notSent || notServed) && len(rest) > 0 {
				m.try(rw, req, target, local, rest)
				return
			}

			m.refuse(rw, req, nodeID, http.StatusBadGateway, err.Error())
		},
	}

	proxy.ServeHTTP(rw, req)
}

func hasNoBody(req *http.Request) bool {
	return req.Body == nil || req.Body == http.NoBody || req.ContentLength == 0
}

// refuse answers a request that could not be forwarded.
// The Proxy-Status header (RFC 9209) tells the client that the answer comes from the mesh, not from the service.
func (m *Mesh) refuse(rw http.ResponseWriter, req *http.Request, nodeID string, status int, cause string) {
	if nodeID == "" {
		m.countRequest("", forwardHTTP, resultUnavailable)
	}

	m.logger.Debug().Str("peer", nodeID).Str("host", req.Host).Str("cause", cause).Msg("Unable to forward a request through the mesh")

	proxyError := "destination_unavailable"
	if status == http.StatusBadGateway {
		proxyError = "connection_terminated"
	}

	rw.Header().Set("Proxy-Status", "traefix-mesh; error="+proxyError)
	http.Error(rw, http.StatusText(status), status)
}

// forwardUpgrade forwards a request that asks to switch protocols, such as a WebSocket handshake.
// The request and what follows it travel as bytes in a stream of the link: the instance that owns the router serves it
// as an HTTP/1.1 connection.
//
// It returns true when the stream could not be opened: nothing of the request was sent, and another place can serve it.
func (m *Mesh) forwardUpgrade(rw http.ResponseWriter, req *http.Request, s *session, nodeID string, target *forwardTarget) bool {
	info := newForwardInfo(req, forwardUpgrade, target.router)

	stream, err := s.OpenStream(req.Context(), http.Header{headerForward: {info.encode()}})
	if err != nil {
		m.failed(target.key(), nodeID)
		m.countRequest(nodeID, forwardUpgrade, resultError)
		m.logger.Debug().Err(err).Str("peer", nodeID).Msg("Unable to open a stream for a request through the mesh")

		return true
	}
	defer func() { _ = stream.Close() }()

	m.forwardUpgraded(rw, req, stream, nodeID, target.key())

	return false
}

// forwardUpgraded carries a request that asks to switch protocols, and what follows it, over an open stream.
// The route is named by its key.
func (m *Mesh) forwardUpgraded(rw http.ResponseWriter, req *http.Request, stream Stream, nodeID, route string) {
	out := req.Clone(req.Context())
	out.RequestURI = ""

	if err := out.Write(stream); err != nil {
		m.failed(route, nodeID)
		m.countRequest(nodeID, forwardUpgrade, resultError)
		m.refuse(rw, req, nodeID, http.StatusBadGateway, err.Error())

		return
	}

	reader := bufio.NewReader(stream)

	resp, err := http.ReadResponse(reader, out)
	if err != nil {
		m.failed(route, nodeID)
		m.countRequest(nodeID, forwardUpgrade, resultError)
		m.refuse(rw, req, nodeID, http.StatusBadGateway, err.Error())

		return
	}
	defer func() { _ = resp.Body.Close() }()

	m.succeeded(route, nodeID)
	m.countRequest(nodeID, forwardUpgrade, resultForwarded)

	if resp.StatusCode != http.StatusSwitchingProtocols {
		// The server did not switch: its answer is an ordinary one.
		maps.Copy(rw.Header(), resp.Header)

		rw.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(rw, resp.Body)

		return
	}

	conn, buffered, err := http.NewResponseController(rw).Hijack()
	if err != nil {
		m.refuse(rw, req, nodeID, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = conn.Close() }()

	if _, err := fmt.Fprintf(conn, "HTTP/1.1 %s\r\n", resp.Status); err != nil {
		return
	}

	if err := resp.Header.Write(conn); err != nil {
		return
	}

	if _, err := io.WriteString(conn, "\r\n"); err != nil {
		return
	}

	// Each side may have sent bytes right after its part of the handshake: the buffered readers hold them.
	done := make(chan struct{}, 2)

	go func() {
		_, _ = io.Copy(stream, buffered.Reader)
		_ = stream.CloseWrite()
		done <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(conn, reader)
		done <- struct{}{}
	}()

	// One side is done: the other one has nothing left to answer to.
	<-done
}

// isUpgrade tells whether a request asks to switch protocols on its connection.
func isUpgrade(req *http.Request) bool {
	return req.Header.Get("Upgrade") != "" && httpguts.HeaderValuesContainsToken(req.Header["Connection"], "Upgrade")
}

func clientIP(req *http.Request) string {
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}

	return host
}

// sessionTo returns the session of the link to an instance, nil when the link is down.
func (m *Mesh) sessionTo(nodeID string) *session {
	m.mu.Lock()
	entry := m.byNode[nodeID]
	m.mu.Unlock()

	if entry == nil {
		return nil
	}

	s := entry.link.current()
	if s == nil {
		return nil
	}

	// A connection that was just closed is still the one of the link until its pings notice:
	// a request must not be lost to it when another place can serve it.
	select {
	case <-s.Done():
		return nil
	default:
		return s
	}
}

// plan returns the places to serve a request of a route, in the order they are tried.
//
// The instances that can serve it now come first, the highest priority first. Among the instances of one priority,
// the balance option chooses, except that the default priority, 0, prefers this instance.
// When its health check says that the local router is down, this instance comes last: it still answers as it would alone.
func (m *Mesh) plan(target *forwardTarget, client string) []choice {
	var choices []choice

	route := target.key()

	localUp := target.local && !m.isLocalDown(route)
	if localUp {
		choices = append(choices, choice{local: true, nodeID: m.member.NodeID(), priority: target.localPriority})
	}

	for _, candidate := range target.candidates {
		if candidate.down || m.quarantined(route, candidate.nodeID) || m.sessionTo(candidate.nodeID) == nil {
			continue
		}

		choices = append(choices, choice{nodeID: candidate.nodeID, priority: candidate.priority})
	}

	roundRobin := m.conf.Balance == cluster.MeshBalanceRoundRobin

	slices.SortStableFunc(choices, func(a, b choice) int {
		if a.priority != b.priority {
			return b.priority - a.priority
		}

		if a.priority == 0 && a.local != b.local {
			if a.local {
				return -1
			}

			return 1
		}

		if roundRobin {
			return strings.Compare(a.nodeID, b.nodeID)
		}

		// The same client goes to the same instance, whichever instance it reaches: every instance computes the same order.
		return compareUint64(rendezvous(client, b.nodeID), rendezvous(client, a.nodeID))
	})

	if roundRobin && len(choices) > 1 {
		// Each request starts one instance further among the ones of the highest priority, unless this instance is preferred.
		top := 0
		for top < len(choices) && choices[top].priority == choices[0].priority {
			top++
		}

		if top > 1 && !(choices[0].priority == 0 && choices[0].local) {
			shift := int(m.turn.Add(1) % uint64(top)) //nolint:gosec // The remainder of a division by a small count.
			copy(choices, slices.Concat(choices[shift:top], choices[:shift]))
		}
	}

	if target.local && !localUp {
		choices = append(choices, choice{local: true, nodeID: m.member.NodeID(), priority: target.localPriority})
	}

	return choices
}

const (
	// quarantineAfter is how many requests in a row an instance may fail before it is left alone for a route.
	quarantineAfter = 3

	quarantineFirst   = 5 * time.Second
	quarantineLongest = time.Minute
)

// quarantine counts the failures of an instance for a route, and keeps the instance away from it for a while after too many.
type quarantine struct {
	failures int
	span     time.Duration
	until    time.Time
}

func quarantineKey(route, nodeID string) string {
	return route + "\x00" + nodeID
}

// failed records that a request for a route, named by its key, could not be delivered to an instance.
func (m *Mesh) failed(route, nodeID string) {
	now := m.now()

	m.mu.Lock()
	defer m.mu.Unlock()

	q := m.quarantines[quarantineKey(route, nodeID)]
	if q == nil {
		q = &quarantine{}
		m.quarantines[quarantineKey(route, nodeID)] = q
	}

	q.failures++
	if q.failures < quarantineAfter {
		return
	}

	// An instance that fails again right after its time away stays away twice as long.
	q.failures = 0
	q.span = min(max(2*q.span, quarantineFirst), quarantineLongest)
	q.until = now.Add(q.span)
}

// succeeded records that an instance served a request for a route.
func (m *Mesh) succeeded(route, nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.quarantines, quarantineKey(route, nodeID))
}

func (m *Mesh) quarantined(route, nodeID string) bool {
	now := m.now()

	m.mu.Lock()
	defer m.mu.Unlock()

	q := m.quarantines[quarantineKey(route, nodeID)]

	return q != nil && now.Before(q.until)
}

func rendezvous(client, nodeID string) uint64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(client))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(nodeID))

	return hash.Sum64()
}

func compareUint64(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// LocalHTTPRouters returns the routers the other instances may send requests for, as their providers wrote them.
func (m *Mesh) LocalHTTPRouters() map[string]*dynamic.Router {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.localHTTP
}

// serveForwarded serves a request that another instance received and forwarded, with the router it names.
func (m *Mesh) serveForwarded(rw http.ResponseWriter, req *http.Request) {
	peer := peerFrom(req.Context())

	info, err := decodeForwardInfo(req.Header.Get(headerForward))
	if err != nil {
		http.Error(rw, "unreadable mesh request", http.StatusBadRequest)
		return
	}

	var (
		handler    http.Handler
		tcpHandler tcp.Handler
	)

	if current := m.routing.Load(); current != nil {
		handler, tcpHandler = current.http[info.Router], current.tcp[info.Router]
	}

	switch info.Kind {
	case forwardHTTP, forwardUpgrade:
		tcpHandler = nil
	case forwardTCP:
		handler = nil
	default:
		http.Error(rw, "unknown kind of mesh request", http.StatusBadRequest)
		return
	}

	if handler == nil && tcpHandler == nil {
		// The router went away between the announcement and the request, e.g. its container stopped.
		m.countReceived(peer, info.Kind, resultUnknown)

		rw.Header().Set(headerForwardError, errorUnknownRouter)
		http.Error(rw, "unknown router", http.StatusNotFound)

		return
	}

	m.seen(peer)
	m.countReceived(peer, info.Kind, resultServed)

	switch info.Kind {
	case forwardHTTP:
		handler.ServeHTTP(rw, m.asReceived(req, info))
	case forwardUpgrade:
		m.serveUpgrade(rw, req, info, handler)
	case forwardTCP:
		m.serveTCP(rw, req, info, tcpHandler)
	}
}

// asReceived gives a forwarded request what it would have if the client had connected to this instance:
// the address of the client, its TLS connection and its fingerprints.
func (m *Mesh) asReceived(req *http.Request, info forwardInfo) *http.Request {
	ctx := req.Context()

	if f := info.Fingerprint; f != nil {
		ctx = fingerprint.ContextWithForwarded(ctx, fingerprint.Forwarded{
			ConnID:       f.ConnID,
			Fingerprints: fingerprint.Fingerprints{JA3: f.JA3, JA4: f.JA4, JA4H: f.JA4H, HTTP2: f.HTTP2, JA4T: f.JA4T},
		})
	}

	req = req.WithContext(ctx)
	req.Header.Del(headerForward)

	if info.Remote != "" {
		req.RemoteAddr = info.Remote
	}

	// The TLS state of the request is the one of the link between the instances: the one of the client replaces it.
	req.TLS = nil
	if info.TLS != nil {
		req.TLS = &tls.ConnectionState{
			Version:            info.TLS.Version,
			CipherSuite:        info.TLS.CipherSuite,
			ServerName:         info.TLS.ServerName,
			NegotiatedProtocol: info.TLS.Protocol,
			HandshakeComplete:  true,
		}
	}

	return req
}

// serveUpgrade serves a request that asks to switch protocols: the stream is an HTTP/1.1 connection of its own,
// which the handler can take over as it does with the connection of a client.
func (m *Mesh) serveUpgrade(rw http.ResponseWriter, req *http.Request, info forwardInfo, handler http.Handler) {
	stream, err := acceptStream(rw, req)
	if err != nil {
		return
	}

	conn := newStreamConn(stream, info.Remote)
	listener := newSingleListener(conn)

	server := &http.Server{
		ReadHeaderTimeout: m.linkTimeout,
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, upgraded *http.Request) {
			handler.ServeHTTP(rw, m.asReceived(upgraded, info))
		}),
	}
	// One request travels in the stream: the connection ends with it, unless the handler takes it over.
	server.SetKeepAlivesEnabled(false)

	go func() { _ = server.Serve(listener) }()

	select {
	case <-conn.closed:
	case <-req.Context().Done():
		// The other instance gave the stream up. Nothing may write to the response once this handler returns:
		// both directions are waited for.
		_ = conn.Close()
		_ = stream.Close()

		<-conn.closed
	}

	_ = listener.Close()
}

// streamConn makes a stream of a link the connection of an HTTP/1.1 server.
// The server needs deadlines, to interrupt a pending read when a handler takes the connection over, which a stream
// over TCP does not have: the server gets one end of an in-memory connection, and the other end is copied to the stream.
type streamConn struct {
	net.Conn

	remote net.Addr
	// closed is closed once nothing flows any more, in either direction.
	closed chan struct{}
}

func newStreamConn(stream Stream, remote string) *streamConn {
	near, far := net.Pipe()

	conn := &streamConn{Conn: near, closed: make(chan struct{}), remote: meshAddr(remote)}

	// The HTTP server gives the address to the request, which the handlers read as the one of a TCP connection.
	if host, port, err := net.SplitHostPort(remote); err == nil {
		if ip := net.ParseIP(host); ip != nil {
			if number, err := strconv.Atoi(port); err == nil {
				conn.remote = &net.TCPAddr{IP: ip, Port: number}
			}
		}
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		_, _ = io.Copy(far, stream)
		// The other instance is done sending: the server reads the end of its connection.
		_ = far.Close()
	})

	wg.Go(func() {
		_, _ = io.Copy(stream, far)
		// The server closed its connection: the stream has nothing left to carry.
		_ = stream.Close()
	})

	go func() {
		wg.Wait()
		close(conn.closed)
	}()

	return conn
}

func (c *streamConn) LocalAddr() net.Addr  { return meshAddr(ProviderName) }
func (c *streamConn) RemoteAddr() net.Addr { return c.remote }

type meshAddr string

func (a meshAddr) Network() string { return ProviderName }
func (a meshAddr) String() string  { return string(a) }

// singleListener gives an HTTP server one connection to serve.
type singleListener struct {
	conn   chan net.Conn
	closed chan struct{}
	once   sync.Once
}

func newSingleListener(conn net.Conn) *singleListener {
	l := &singleListener{conn: make(chan net.Conn, 1), closed: make(chan struct{})}
	l.conn <- conn

	return l
}

func (l *singleListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.conn:
		return conn, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *singleListener) Close() error {
	l.once.Do(func() { close(l.closed) })

	return nil
}

func (l *singleListener) Addr() net.Addr { return meshAddr(ProviderName) }
