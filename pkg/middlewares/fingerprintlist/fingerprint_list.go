package fingerprintlist

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/fingerprint"
	"github.com/traefik/traefik/v3/pkg/middlewares"
	"github.com/traefik/traefik/v3/pkg/middlewares/observability"
)

const (
	typeName = "FingerprintList"

	modeAllow = "allow"
	modeDeny  = "deny"
)

type patterns struct {
	exact    map[string]struct{}
	prefixes []string
}

func newPatterns(values []string) (*patterns, error) {
	p := &patterns{exact: make(map[string]struct{})}
	for _, v := range values {
		prefix, isPrefix := strings.CutSuffix(v, "*")
		if prefix == "" {
			return nil, fmt.Errorf("invalid fingerprint %q: it must not be empty", v)
		}
		if isPrefix {
			p.prefixes = append(p.prefixes, prefix)
		} else {
			p.exact[v] = struct{}{}
		}
	}
	return p, nil
}

// match reports whether the fingerprint is listed. An empty fingerprint, one the entry point did not compute,
// is never listed.
func (p *patterns) match(fp string) bool {
	if fp == "" {
		return false
	}
	if _, ok := p.exact[fp]; ok {
		return true
	}
	for _, prefix := range p.prefixes {
		if strings.HasPrefix(fp, prefix) {
			return true
		}
	}
	return false
}

type fingerprintList struct {
	next             http.Handler
	name             string
	allow            bool
	rejectStatusCode int

	ja3, ja4, ja4h, http2 *patterns
}

// New builds a middleware denying or allowing the requests by their fingerprints.
func New(ctx context.Context, next http.Handler, config dynamic.FingerprintList, name string) (http.Handler, error) {
	logger := middlewares.GetLogger(ctx, name, typeName)
	logger.Debug().Msg("Creating middleware")

	f := &fingerprintList{next: next, name: name}

	switch config.Mode {
	case modeAllow:
		f.allow = true
	case modeDeny:
	default:
		return nil, fmt.Errorf("invalid mode %q: it must be %s or %s", config.Mode, modeAllow, modeDeny)
	}

	if len(config.JA3)+len(config.JA4)+len(config.JA4H)+len(config.HTTP2) == 0 {
		return nil, errors.New("no fingerprint listed")
	}

	f.rejectStatusCode = config.RejectStatusCode
	if f.rejectStatusCode == 0 {
		f.rejectStatusCode = http.StatusForbidden
	} else if http.StatusText(f.rejectStatusCode) == "" {
		return nil, fmt.Errorf("invalid HTTP status code %d", f.rejectStatusCode)
	}

	var err error
	if f.ja3, err = newPatterns(config.JA3); err != nil {
		return nil, err
	}
	if f.ja4, err = newPatterns(config.JA4); err != nil {
		return nil, err
	}
	if f.ja4h, err = newPatterns(config.JA4H); err != nil {
		return nil, err
	}
	if f.http2, err = newPatterns(config.HTTP2); err != nil {
		return nil, err
	}

	return f, nil
}

func (f *fingerprintList) GetTracingInformation() (string, string) {
	return f.name, typeName
}

func (f *fingerprintList) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fps, _ := fingerprint.FingerprintsFromRequest(req)

	listed := f.ja3.match(fps.JA3) || f.ja4.match(fps.JA4) || f.ja4h.match(fps.JA4H) || f.http2.match(fps.HTTP2)

	if listed != f.allow {
		logger := middlewares.GetLogger(req.Context(), f.name, typeName)
		logger.Debug().Str("ja4", fps.JA4).Str("ja4h", fps.JA4H).Bool("listed", listed).Msg("Rejecting request by fingerprint")
		observability.SetStatusErrorf(req.Context(), "Rejecting request by fingerprint, listed: %t", listed)

		rw.WriteHeader(f.rejectStatusCode)
		_, _ = rw.Write([]byte(http.StatusText(f.rejectStatusCode)))
		return
	}

	f.next.ServeHTTP(rw, req)
}
