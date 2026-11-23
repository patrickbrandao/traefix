package mesh

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	pathHello = "/v1/hello"
	pathPing  = "/v1/ping"
	pathPeers = "/v1/peers"

	// maxControlBody bounds what an instance reads of a control message of another one.
	maxControlBody = 1 << 20
)

// hello is what two instances tell each other when a link opens.
type hello struct {
	NodeID string `json:"nodeID"`
	// Instance changes every time the process starts, which tells an instance that reached itself from one that restarted.
	Instance string `json:"instance"`
	Version  string `json:"version"`
	// Advertise are the addresses the instance is reached at.
	Advertise []string `json:"advertise,omitempty"`
	// Peers are the other instances the instance is in contact with: a new instance that knows one finds them all.
	Peers []peerInfo `json:"peers,omitempty"`
	// EntryPoints tell the other instances what the entry points of the instance do.
	EntryPoints map[string]EntryPointSummary `json:"entryPoints,omitempty"`
	// Role is member or satellite. The versions before the satellites tell none, and are members.
	Role string `json:"role,omitempty"`
}

// peersAnswer is the answer to a request for the instances an instance is in contact with.
type peersAnswer struct {
	Peers []peerInfo `json:"peers"`
}

// ServeHTTP answers the requests the other instances send through their links.
func (m *Mesh) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	m.observeSatellite(req)

	if m.conf.Satellite && req.ProtoMajor == 3 {
		m.markQUIC(peerFrom(req.Context()))
	}

	// A request that another instance forwards carries the header, whatever its host.
	if req.Header.Get(headerForward) != "" {
		m.serveForwarded(rw, req)
		return
	}

	if req.Host != controlHost {
		http.Error(rw, "unknown mesh request", http.StatusNotFound)
		return
	}

	switch {
	case req.Method == http.MethodPost && req.URL.Path == pathHello:
		m.serveHello(rw, req)

	case req.Method == http.MethodGet && req.URL.Path == pathPing:
		m.seen(peerFrom(req.Context()))
		rw.WriteHeader(http.StatusNoContent)

	case req.Method == http.MethodGet && req.URL.Path == pathRoutes:
		m.serveRoutes(rw, req)

	case req.Method == http.MethodGet && req.URL.Path == pathChallenge:
		m.serveChallenge(rw, req)

	case req.Method == http.MethodGet && req.URL.Path == pathCertificates:
		m.serveCertificates(rw, req)

	case req.Method == http.MethodGet && req.URL.Path == pathPeers:
		m.seen(peerFrom(req.Context()))
		rw.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(rw).Encode(peersAnswer{Peers: m.knownPeers()}); err != nil {
			m.logger.Debug().Err(err).Msg("Unable to answer a request for the instances of the mesh")
		}

	default:
		http.Error(rw, "unknown mesh request", http.StatusNotFound)
	}
}

func (m *Mesh) serveHello(rw http.ResponseWriter, req *http.Request) {
	var theirs hello
	if err := json.NewDecoder(io.LimitReader(req.Body, maxControlBody)).Decode(&theirs); err != nil {
		http.Error(rw, "unreadable hello", http.StatusBadRequest)
		return
	}

	// The certificate is what proves who the other instance is: what it says must agree with it.
	if theirs.NodeID != peerFrom(req.Context()) {
		http.Error(rw, "the hello does not match the certificate", http.StatusBadRequest)
		return
	}

	// An instance that reached itself, or one that took the nodeID of this one, gets the answer that tells it so,
	// and is not counted among the instances linked to this one. So does a satellite that reached another satellite:
	// the answer tells it that a satellite answers here, and it stops.
	if theirs.NodeID != m.member.NodeID() && (!m.conf.Satellite || roleOf(theirs) != RoleSatellite) {
		m.greeted(theirs, req.RemoteAddr)
		m.compareConfiguration(theirs)

		// A satellite is reached back over QUIC, through the mapping its NAT opened for its link,
		// or through a link over TCP it opens when its link comes over TCP.
		if roleOf(theirs) == RoleSatellite && !m.conf.Satellite {
			observed := ""
			if req.ProtoMajor == 3 {
				observed = req.RemoteAddr
			}

			m.learnSatellite(theirs.NodeID, observed)
		}

		// The instance that opened the link is linked back, at the addresses it says it is reached at,
		// and so are the instances it is in contact with.
		m.learn(sourceExchange, theirs.NodeID, theirs.Advertise)
		m.contact(theirs.NodeID)
		m.learnPeers(theirs.Peers)
	}

	rw.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(rw).Encode(m.hello()); err != nil {
		m.logger.Debug().Err(err).Str("peer", theirs.NodeID).Msg("Unable to answer the hello of an instance of the mesh")
	}
}

func (m *Mesh) hello() hello {
	m.mu.Lock()
	entryPoints := m.entryPointSummaries
	m.mu.Unlock()

	return hello{
		NodeID:      m.member.NodeID(),
		Instance:    m.member.Instance(),
		Version:     m.version,
		Advertise:   m.advertise,
		Peers:       m.knownPeers(),
		EntryPoints: entryPoints,
		Role:        m.role(),
	}
}

// askPeers asks the other end of a session for the instances it is in contact with.
func (m *Mesh) askPeers(ctx context.Context, s *session) ([]peerInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, m.linkTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+controlHost+pathPeers, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected answer to a request for the instances of the mesh: %s", resp.Status)
	}

	var answer peersAnswer
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxControlBody)).Decode(&answer); err != nil {
		return nil, err
	}

	return answer.Peers, nil
}

// sayHello introduces this instance to the one at the other end of a session, and returns what it answers.
func (m *Mesh) sayHello(ctx context.Context, s *session) (hello, error) {
	body, err := json.Marshal(m.hello())
	if err != nil {
		return hello{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+controlHost+pathHello, bytes.NewReader(body))
	if err != nil {
		return hello{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.RoundTrip(req)
	if err != nil {
		return hello{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return hello{}, fmt.Errorf("the other instance refused the hello: %s", resp.Status)
	}

	var theirs hello
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxControlBody)).Decode(&theirs); err != nil {
		return hello{}, fmt.Errorf("reading the hello of the other instance: %w", err)
	}

	if theirs.NodeID != s.peer {
		return hello{}, errors.New("the hello of the other instance does not match its certificate")
	}

	return theirs, nil
}

// ping asks the other end of a session for a sign of life, and returns how long the answer took.
func ping(ctx context.Context, s *session, timeout time.Duration) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+controlHost+pathPing, nil)
	if err != nil {
		return 0, err
	}

	start := time.Now()

	resp, err := s.RoundTrip(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		return 0, fmt.Errorf("unexpected answer to a ping: %s", resp.Status)
	}

	return time.Since(start), nil
}
