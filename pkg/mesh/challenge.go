package mesh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	pathChallenge = "/v1/challenge"

	// maxChallengeBody bounds what an instance reads of the answer of a challenge: a key authorization is short.
	maxChallengeBody = 4 << 10
)

// ChallengeAnswer returns the answer of an HTTP-01 challenge this instance presented, or nothing.
type ChallengeAnswer func(ctx context.Context, token, domain string) []byte

// SetChallengeAnswer gives the mesh the answers of the HTTP-01 challenges this instance presents, for the instances
// that the CA reaches in its place.
func (m *Mesh) SetChallengeAnswer(answer ChallengeAnswer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.challengeAnswer = answer
}

// serveChallenge answers another instance with the answer of a challenge this instance presented.
// It never asks further: the instance that asks already asks every instance it links to.
func (m *Mesh) serveChallenge(rw http.ResponseWriter, req *http.Request) {
	m.seen(peerFrom(req.Context()))

	m.mu.Lock()
	answer := m.challengeAnswer
	m.mu.Unlock()

	token, domain := req.URL.Query().Get("token"), req.URL.Query().Get("domain")

	var value []byte
	if answer != nil && token != "" && domain != "" {
		value = answer(req.Context(), token, domain)
	}

	if len(value) == 0 {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	_, _ = rw.Write(value)
}

// LookupChallenge asks the instances this one links to, all at once, for the answer of a challenge one of them presented.
// It returns the first answer, nothing when they all say they do not know the challenge, and an error when some did not
// answer. The satellites never ask a CA: they are not asked.
func (m *Mesh) LookupChallenge(ctx context.Context, token, domain string) ([]byte, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sessions := m.challengeSessions()
	if len(sessions) == 0 {
		return nil, nil
	}

	type result struct {
		value []byte
		err   error
	}

	results := make(chan result, len(sessions))

	for _, s := range sessions {
		go func() {
			value, err := askChallenge(ctx, s, token, domain)
			results <- result{value: value, err: err}
		}()
	}

	var failures []error

	for range sessions {
		r := <-results

		if len(r.value) > 0 {
			return r.value, nil
		}

		if r.err != nil {
			failures = append(failures, r.err)
		}
	}

	return nil, errors.Join(failures...)
}

// challengeSessions returns the sessions of the links up to the instances that are not satellites.
func (m *Mesh) challengeSessions() []*session {
	m.mu.Lock()
	entries := make([]*peerEntry, 0, len(m.byNode))
	for _, entry := range m.byNode {
		if !entry.satellite {
			entries = append(entries, entry)
		}
	}
	m.mu.Unlock()

	var sessions []*session

	for _, entry := range entries {
		if s := m.sessionTo(entry.nodeID); s != nil {
			sessions = append(sessions, s)
		}
	}

	return sessions
}

func askChallenge(ctx context.Context, s *session, token, domain string) ([]byte, error) {
	query := url.Values{"token": {token}, "domain": {domain}}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+controlHost+pathChallenge+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", s.peer, err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		return io.ReadAll(io.LimitReader(resp.Body, maxChallengeBody))
	case http.StatusNotFound:
		return nil, nil
	default:
		return nil, fmt.Errorf("%s: unexpected answer to a request for an ACME challenge: %s", s.peer, resp.Status)
	}
}
