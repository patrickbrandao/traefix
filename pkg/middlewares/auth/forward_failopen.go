package auth

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gokitmetrics "github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"
	"github.com/rs/zerolog"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/fingerprint"
	"github.com/traefik/traefik/v3/pkg/middlewares"
	"github.com/traefik/traefik/v3/pkg/observability/metrics"
)

// The Traefix additions to forwardAuth (requestHeaders and failOpen) live here, so that forward.go only gets the calls.

const (
	failOpenTimeout = "timeout"
	failOpenNetwork = "network"
	failOpenHTTP5xx = "http_5xx"
	failOpenHTTP4xx = "http_4xx"

	// failOpenLogInterval bounds the fail-open logs, so that an authentication server down does not flood the logs.
	failOpenLogInterval = time.Minute
)

var failOpenCounter atomic.Value

// SetMetricsRegistry sets the registry of the forwardAuth metrics.
func SetMetricsRegistry(registry metrics.Registry) {
	if registry == nil {
		return
	}
	if c := registry.ForwardAuthFailOpenCounter(); c != nil {
		failOpenCounter.Store(c)
	}
}

func failOpenMetric() gokitmetrics.Counter {
	if c, ok := failOpenCounter.Load().(gokitmetrics.Counter); ok {
		return c
	}
	return discard.NewCounter()
}

type failOpen struct {
	denyRequiresHeader string
	headerName         string
	headerValue        string

	mu      sync.Mutex
	lastLog time.Time
	skipped int
}

// configureTraefix validates and applies the requestHeaders and failOpen options.
func (fa *forwardAuth) configureTraefix(logger *zerolog.Logger, config dynamic.ForwardAuth) error {
	if len(config.RequestHeaders) > 0 {
		if err := checkSecretTransport(config.Address); err != nil {
			return err
		}

		fa.requestHeaders = make(http.Header, len(config.RequestHeaders))
		for name, value := range config.RequestHeaders {
			if err := checkRequestHeaderValue(name, value); err != nil {
				return err
			}
			fa.requestHeaders.Set(name, value)
		}

		if _, ok := fa.requestHeaders["Authorization"]; ok && len(config.AuthRequestHeaders) == 0 {
			logger.Warn().Msg("requestHeaders sets Authorization while authRequestHeaders is empty: " +
				"the authentication request gets all the client headers, and the Authorization header of the client is replaced. " +
				"List the forwarded headers in authRequestHeaders.")
		}
	}

	if config.FailOpen != nil {
		fo := &failOpen{}
		if config.FailOpen.DenyRequiresResponseHeader != "" {
			fo.denyRequiresHeader = http.CanonicalHeaderKey(config.FailOpen.DenyRequiresResponseHeader)
		}
		if h := config.FailOpen.Header; h != nil && h.Name != "" {
			fo.headerName = http.CanonicalHeaderKey(h.Name)
			fo.headerValue = h.Value
		}
		fa.failOpen = fo
	}

	return nil
}

// checkRequestHeaderValue refuses an empty value, or a credentials header with a scheme and no credentials,
// which is what an unset environment variable gives, e.g. "Bearer ${KEY}". With failOpen, the authentication
// server rejecting it would go unnoticed.
func checkRequestHeaderValue(name, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("forward authentication requestHeaders %q has an empty value", name)
	}

	switch http.CanonicalHeaderKey(name) {
	case "Authorization", "Proxy-Authorization":
		if _, credentials, _ := strings.Cut(value, " "); strings.TrimSpace(credentials) == "" {
			return fmt.Errorf("forward authentication requestHeaders %q has no credentials after the scheme %q", name, value)
		}
	}

	return nil
}

// checkSecretTransport refuses the addresses where the requestHeaders values, which may carry a key,
// would cross a public network in clear text.
func checkSecretTransport(address string) error {
	if address == "" {
		// A Traefik service: its servers and their transport are configured apart.
		return nil
	}

	u, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("parsing forward authentication address: %w", err)
	}
	if u.Scheme == "https" {
		return nil
	}

	host := u.Hostname()
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return nil
	}
	if ip, err := netip.ParseAddr(host); err == nil && (ip.IsLoopback() || ip.IsPrivate()) {
		return nil
	}

	return fmt.Errorf("forward authentication requestHeaders would be sent in clear text to %q: "+
		"use an https address, or a loopback or private IP address", host)
}

// setTraefixHeaders sets on the authentication request the headers of the entry point fingerprint,
// which a selection in authRequestHeaders would leave out, then the requestHeaders.
func (fa *forwardAuth) setTraefixHeaders(forwardReq *http.Request) {
	fingerprint.SetHeaders(forwardReq.Context(), forwardReq.Header)
	maps.Copy(forwardReq.Header, fa.requestHeaders)
}

// failOpenOnError lets the request through after a failed authentication request, and reports whether it did.
func (fa *forwardAuth) failOpenOnError(rw http.ResponseWriter, req *http.Request, err error) bool {
	reason, ok := fa.failOpenReasonForError(err)
	if ok {
		fa.letThrough(rw, req, reason, err.Error())
	}
	return ok
}

// failOpenOnResponse lets the request through after a failure response, and reports whether it did.
func (fa *forwardAuth) failOpenOnResponse(rw http.ResponseWriter, req *http.Request, resp *http.Response) bool {
	reason, ok := fa.failOpenReasonForResponse(resp)
	if ok {
		fa.letThrough(rw, req, reason, "status code "+strconv.Itoa(resp.StatusCode))
	}
	return ok
}

// failOpenReasonForError returns why the request is let through after a failed authentication request,
// or false when it is not, e.g. when the client went away.
func (fa *forwardAuth) failOpenReasonForError(err error) (string, bool) {
	if fa.failOpen == nil || errors.Is(err, context.Canceled) {
		return "", false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return failOpenTimeout, true
	}
	if netErr, ok := errors.AsType[net.Error](err); ok && netErr.Timeout() {
		return failOpenTimeout, true
	}

	return failOpenNetwork, true
}

// failOpenReasonForResponse returns why the request is let through after the response of the authentication server.
func (fa *forwardAuth) failOpenReasonForResponse(resp *http.Response) (string, bool) {
	if fa.failOpen == nil {
		return "", false
	}

	switch {
	case resp.StatusCode >= http.StatusInternalServerError:
		return failOpenHTTP5xx, true
	case resp.StatusCode >= http.StatusBadRequest && fa.failOpen.denyRequiresHeader != "":
		if len(resp.Header[fa.failOpen.denyRequiresHeader]) == 0 {
			return failOpenHTTP4xx, true
		}
	}

	return "", false
}

// letThrough serves the request after a failure of the authentication server, as no verdict was given.
func (fa *forwardAuth) letThrough(rw http.ResponseWriter, req *http.Request, reason, detail string) {
	// The client could have sent the headers the authentication server sets on success.
	for _, name := range fa.authResponseHeaders {
		req.Header.Del(name)
	}
	if fa.authResponseHeadersRegex != nil {
		for name := range req.Header {
			if fa.authResponseHeadersRegex.MatchString(name) {
				req.Header.Del(name)
			}
		}
	}

	if fa.failOpen.headerName != "" {
		req.Header[fa.failOpen.headerName] = []string{fa.failOpen.headerValue}
	}

	failOpenMetric().With("middleware", fa.name, "reason", reason).Add(1)
	fa.failOpen.log(req, fa.name, reason, detail)

	req.RequestURI = req.URL.RequestURI()
	fa.next.ServeHTTP(rw, req)
}

func (f *failOpen) log(req *http.Request, name, reason, detail string) {
	f.mu.Lock()
	now := time.Now()
	if now.Sub(f.lastLog) < failOpenLogInterval {
		f.skipped++
		f.mu.Unlock()
		return
	}
	skipped := f.skipped
	f.lastLog = now
	f.skipped = 0
	f.mu.Unlock()

	middlewares.GetLogger(req.Context(), name, typeNameForward).Warn().
		Str("reason", reason).Str("detail", detail).Int("previousFailOpens", skipped).
		Msg("Authentication server failed, letting the request through")
}
