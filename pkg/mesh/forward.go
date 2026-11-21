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
	"net/http/httputil"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/fingerprint"
	"github.com/traefik/traefik/v3/pkg/middlewares/accesslog"
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
	Remote string      `json:"remote,omitempty"`
	TLS    *forwardTLS `json:"tls,omitempty"`
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

// forwardTarget is where the requests of a router learned from the mesh go.
type forwardTarget struct {
	// router is the name of the router on the instances that own it.
	router     string
	candidates []forwardCandidate
}

type forwardCandidate struct {
	nodeID   string
	priority int
}

// meshService forwards the requests of a router learned from the mesh to an instance that owns it.
type meshService struct {
	mesh *Mesh
	// name is the name of the service, which is the one of the router of this instance.
	name string
}

func (s *meshService) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	m := s.mesh

	target := m.targetOf(s.name)
	if target == nil {
		m.refuse(rw, req, "", http.StatusServiceUnavailable, "no instance of the mesh announces the router")
		return
	}

	// The first instance, in the order of preference, that this instance has a link to.
	var session *session
	var nodeID string

	for _, candidate := range m.order(target, clientIP(req)) {
		if session = m.sessionTo(candidate); session != nil {
			nodeID = candidate
			break
		}
	}

	if session == nil {
		m.refuse(rw, req, "", http.StatusServiceUnavailable, "no link to an instance that owns the router")
		return
	}

	if data := accesslog.GetLogData(req); data != nil {
		data.Core[accesslog.ServiceName] = s.name
		data.Core[accesslog.ServiceURL] = "mesh://" + nodeID
		data.Core[accesslog.ServiceAddr] = nodeID
	}

	if isUpgrade(req) {
		m.forwardUpgrade(rw, req, session, nodeID, target.router)
		return
	}

	info := newForwardInfo(req, forwardHTTP, target.router)

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
				return fmt.Errorf("the instance %s did not serve the request: %s", nodeID, cause)
			}

			m.countRequest(nodeID, forwardHTTP, resultForwarded)

			return nil
		},
		ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
			m.countRequest(nodeID, forwardHTTP, resultError)
			m.refuse(rw, req, nodeID, http.StatusBadGateway, err.Error())
		},
	}

	proxy.ServeHTTP(rw, req)
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
func (m *Mesh) forwardUpgrade(rw http.ResponseWriter, req *http.Request, s *session, nodeID, router string) {
	info := newForwardInfo(req, forwardUpgrade, router)

	stream, err := s.OpenStream(req.Context(), http.Header{headerForward: {info.encode()}})
	if err != nil {
		m.countRequest(nodeID, forwardUpgrade, resultError)
		m.refuse(rw, req, nodeID, http.StatusBadGateway, err.Error())

		return
	}
	defer func() { _ = stream.Close() }()

	out := req.Clone(req.Context())
	out.RequestURI = ""

	if err := out.Write(stream); err != nil {
		m.countRequest(nodeID, forwardUpgrade, resultError)
		m.refuse(rw, req, nodeID, http.StatusBadGateway, err.Error())

		return
	}

	reader := bufio.NewReader(stream)

	resp, err := http.ReadResponse(reader, out)
	if err != nil {
		m.countRequest(nodeID, forwardUpgrade, resultError)
		m.refuse(rw, req, nodeID, http.StatusBadGateway, err.Error())

		return
	}
	defer func() { _ = resp.Body.Close() }()

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

// targetOf returns where the requests of a service of the mesh go.
func (m *Mesh) targetOf(serviceName string) *forwardTarget {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.targets[serviceName]
}

// sessionTo returns the session of the link to an instance, nil when the link is down.
func (m *Mesh) sessionTo(nodeID string) *session {
	m.mu.Lock()
	entry := m.byNode[nodeID]
	m.mu.Unlock()

	if entry == nil {
		return nil
	}

	return entry.link.current()
}

// order returns the instances that own a router in the order they are tried:
// the highest priority first, and among the instances of one priority, the choice of the balance option.
func (m *Mesh) order(target *forwardTarget, client string) []string {
	candidates := slices.Clone(target.candidates)

	var turn uint64
	if m.conf.Balance == cluster.MeshBalanceRoundRobin {
		turn = m.turn.Add(1)
	}

	slices.SortStableFunc(candidates, func(a, b forwardCandidate) int {
		if a.priority != b.priority {
			return b.priority - a.priority
		}

		if m.conf.Balance == cluster.MeshBalanceRoundRobin {
			return 0
		}

		// The same client goes to the same instance, whichever instance it reaches: every instance computes the same order.
		return compareUint64(rendezvous(client, b.nodeID), rendezvous(client, a.nodeID))
	})

	nodeIDs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		nodeIDs = append(nodeIDs, candidate.nodeID)
	}

	if m.conf.Balance != cluster.MeshBalanceRoundRobin {
		return nodeIDs
	}

	// Each request starts one instance further among the ones of the highest priority.
	first := 0
	for first < len(candidates) && candidates[first].priority == candidates[0].priority {
		first++
	}

	if first > 1 {
		shift := int(turn % uint64(first)) //nolint:gosec // The remainder of a division by a small count.
		rotated := slices.Concat(nodeIDs[shift:first], nodeIDs[:shift])
		copy(nodeIDs, rotated)
	}

	return nodeIDs
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

// SwitchHTTPHandlers replaces the handlers that serve these routers.
func (m *Mesh) SwitchHTTPHandlers(handlers map[string]http.Handler) {
	m.handlers.Store(&handlers)
}

// serveForwarded serves a request that another instance received and forwarded, with the router it names.
func (m *Mesh) serveForwarded(rw http.ResponseWriter, req *http.Request) {
	peer := peerFrom(req.Context())

	info, err := decodeForwardInfo(req.Header.Get(headerForward))
	if err != nil {
		http.Error(rw, "unreadable mesh request", http.StatusBadRequest)
		return
	}

	handler := m.handlerOf(info.Router)
	if handler == nil {
		// The router went away between the announcement and the request, e.g. its container stopped.
		m.countReceived(peer, info.Kind, resultUnknown)

		rw.Header().Set(headerForwardError, errorUnknownRouter)
		http.Error(rw, "unknown router", http.StatusNotFound)

		return
	}

	m.seen(peer)

	switch info.Kind {
	case forwardHTTP:
		m.countReceived(peer, info.Kind, resultServed)
		handler.ServeHTTP(rw, m.asReceived(req, info))

	case forwardUpgrade:
		m.countReceived(peer, info.Kind, resultServed)
		m.serveUpgrade(rw, req, info, handler)

	default:
		http.Error(rw, "unknown kind of mesh request", http.StatusBadRequest)
	}
}

func (m *Mesh) handlerOf(router string) http.Handler {
	handlers := m.handlers.Load()
	if handlers == nil {
		return nil
	}

	return (*handlers)[router]
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
