package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	gokitmetrics "github.com/go-kit/kit/metrics"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/static"
	"github.com/traefik/traefik/v3/pkg/fingerprint"
)

// failOpenCount records the fail-open counts by label values.
type failOpenCount struct {
	mu     *sync.Mutex
	values map[string]float64
	labels string
}

func (c *failOpenCount) With(labelValues ...string) gokitmetrics.Counter {
	return &failOpenCount{mu: c.mu, values: c.values, labels: strings.Join(labelValues, ",")}
}

func (c *failOpenCount) Add(delta float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[c.labels] += delta
}

func (c *failOpenCount) get(labels string) float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.values[labels]
}

func useFailOpenCounter(t *testing.T) *failOpenCount {
	t.Helper()

	c := &failOpenCount{mu: &sync.Mutex{}, values: map[string]float64{}}
	previous := failOpenCounter.Load()
	failOpenCounter.Store(gokitmetrics.Counter(c))
	t.Cleanup(func() {
		if previous != nil {
			failOpenCounter.Store(previous)
		}
	})

	return c
}

// backendEcho answers with the request headers the backend receives.
func backendEcho() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("X-Backend", "reached")
		if err := json.NewEncoder(rw).Encode(req.Header); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
		}
	})
}

func failOpenConfig(address string) dynamic.ForwardAuth {
	return dynamic.ForwardAuth{
		Address:             address,
		Timeout:             ptypes.Duration(200 * time.Millisecond),
		AuthResponseHeaders: []string{"X-Verdict", "X-Score"},
		FailOpen: &dynamic.ForwardAuthFailOpen{
			DenyRequiresResponseHeader: "X-Verdict",
			Header:                     &dynamic.ForwardAuthFailOpenHeader{Name: "X-Verdict", Value: "unavailable"},
		},
	}
}

func serve(t *testing.T, handler http.Handler, change func(*http.Request)) (*http.Response, http.Header) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/path", nil)
	// A client forging the verdict of the authentication server.
	req.Header.Set("X-Verdict", "human")
	if change != nil {
		change(req)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	resp := rec.Result()

	var seen http.Header
	if resp.Header.Get("X-Backend") == "reached" {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&seen))
	}

	return resp, seen
}

func TestForwardAuthFailOpen_letsThrough(t *testing.T) {
	down, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	downAddr := "http://" + down.Addr().String()
	require.NoError(t, down.Close())

	slow := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		select {
		case <-time.After(5 * time.Second):
		case <-req.Context().Done():
		}
	}))
	t.Cleanup(slow.Close)

	status := func(code int, header bool) string {
		srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			if header {
				rw.Header().Set("X-Verdict", "bot")
			}
			http.Error(rw, `{"error":"invalid_key"}`, code)
		}))
		t.Cleanup(srv.Close)
		return srv.URL
	}

	testCases := []struct {
		desc    string
		address string
		reason  string
	}{
		{desc: "network error", address: downAddr, reason: failOpenNetwork},
		{desc: "timeout", address: slow.URL, reason: failOpenTimeout},
		{desc: "500", address: status(http.StatusInternalServerError, false), reason: failOpenHTTP5xx},
		{desc: "503 with the verdict header", address: status(http.StatusServiceUnavailable, true), reason: failOpenHTTP5xx},
		{desc: "401 without the verdict header", address: status(http.StatusUnauthorized, false), reason: failOpenHTTP4xx},
		{desc: "403 without the verdict header", address: status(http.StatusForbidden, false), reason: failOpenHTTP4xx},
		{desc: "429 without the verdict header", address: status(http.StatusTooManyRequests, false), reason: failOpenHTTP4xx},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			counter := useFailOpenCounter(t)

			middleware, err := NewForward(t.Context(), backendEcho(), failOpenConfig(test.address), nil, "bot")
			require.NoError(t, err)

			start := time.Now()
			resp, seen := serve(t, middleware, nil)

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			require.NotNil(t, seen)
			assert.Equal(t, []string{"unavailable"}, seen["X-Verdict"])
			assert.InDelta(t, 1, counter.get("middleware,bot,reason,"+test.reason), 0)
			// The worst case adds the timeout, and no more.
			assert.Less(t, time.Since(start), time.Second)
		})
	}
}

func TestForwardAuthFailOpen_denies(t *testing.T) {
	testCases := []struct {
		desc   string
		config func(*dynamic.ForwardAuth)
		code   int
		header bool
	}{
		{desc: "403 with the verdict header", code: http.StatusForbidden, header: true},
		{
			desc:   "4xx without denyRequiresResponseHeader",
			config: func(c *dynamic.ForwardAuth) { c.FailOpen.DenyRequiresResponseHeader = "" },
			code:   http.StatusUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			counter := useFailOpenCounter(t)

			auth := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				if test.header {
					rw.Header().Set("X-Verdict", "bot")
				}
				http.Error(rw, "denied", test.code)
			}))
			t.Cleanup(auth.Close)

			cfg := failOpenConfig(auth.URL)
			if test.config != nil {
				test.config(&cfg)
			}
			middleware, err := NewForward(t.Context(), backendEcho(), cfg, nil, "bot")
			require.NoError(t, err)

			resp, seen := serve(t, middleware, nil)

			assert.Equal(t, test.code, resp.StatusCode)
			assert.Nil(t, seen)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, "denied\n", string(body))
			assert.Empty(t, counter.values)
		})
	}
}

func TestForwardAuthFailOpen_successUnchanged(t *testing.T) {
	auth := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("X-Verdict", "human")
		rw.Header().Set("X-Score", "0.1")
	}))
	t.Cleanup(auth.Close)

	middleware, err := NewForward(t.Context(), backendEcho(), failOpenConfig(auth.URL), nil, "bot")
	require.NoError(t, err)

	resp, seen := serve(t, middleware, func(req *http.Request) { req.Header.Set("X-Verdict", "forged") })

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, []string{"human"}, seen["X-Verdict"])
	assert.Equal(t, []string{"0.1"}, seen["X-Score"])
}

func TestForwardAuthFailOpen_forgedResponseHeadersRemoved(t *testing.T) {
	useFailOpenCounter(t)

	auth := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(auth.Close)

	cfg := failOpenConfig(auth.URL)
	cfg.FailOpen.Header = nil
	middleware, err := NewForward(t.Context(), backendEcho(), cfg, nil, "bot")
	require.NoError(t, err)

	_, seen := serve(t, middleware, func(req *http.Request) { req.Header.Set("X-Score", "0") })

	require.NotNil(t, seen)
	assert.NotContains(t, seen, "X-Verdict")
	assert.NotContains(t, seen, "X-Score")
}

func TestForwardAuth_withoutFailOpen(t *testing.T) {
	down, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	downAddr := "http://" + down.Addr().String()
	require.NoError(t, down.Close())

	failing := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		http.Error(rw, "boom", http.StatusServiceUnavailable)
	}))
	t.Cleanup(failing.Close)

	testCases := []struct {
		desc    string
		address string
		code    int
	}{
		{desc: "network error", address: downAddr, code: http.StatusInternalServerError},
		{desc: "503", address: failing.URL, code: http.StatusServiceUnavailable},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			cfg := failOpenConfig(test.address)
			cfg.FailOpen = nil
			middleware, err := NewForward(t.Context(), backendEcho(), cfg, nil, "bot")
			require.NoError(t, err)

			resp, seen := serve(t, middleware, nil)

			assert.Equal(t, test.code, resp.StatusCode)
			assert.Nil(t, seen)
		})
	}
}

func TestForwardAuth_requestHeaders(t *testing.T) {
	var got http.Header
	auth := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		got = req.Header.Clone()
	}))
	t.Cleanup(auth.Close)

	cfg := failOpenConfig(auth.URL)
	cfg.AuthRequestHeaders = []string{"User-Agent", "Accept"}
	cfg.RequestHeaders = map[string]string{"Authorization": "Bearer my-key"}
	middleware, err := NewForward(t.Context(), backendEcho(), cfg, nil, "bot")
	require.NoError(t, err)

	// The entry point fingerprint runs first, as on a real entry point.
	ep, err := fingerprint.NewEntryPoint(t.Context(), "websecure", &static.Fingerprint{HeaderPrefix: "X-Edge-"}, nil, nil)
	require.NoError(t, err)

	_, seen := serve(t, ep.Handler(middleware), func(req *http.Request) {
		req.Header.Set("User-Agent", "visitor-agent")
		req.Header.Set("Authorization", "Bearer visitor")
		req.Header.Set("Cookie", "sid=visitor")
	})

	assert.Equal(t, []string{"Bearer my-key"}, got["Authorization"])
	assert.NotContains(t, got, "Cookie")
	assert.Equal(t, []string{"visitor-agent"}, got["User-Agent"])
	assert.Equal(t, []string{"h1"}, got["X-Edge-Proto"])

	// The backend gets the client headers, not the key.
	require.NotNil(t, seen)
	assert.Equal(t, []string{"Bearer visitor"}, seen["Authorization"])
	assert.Equal(t, []string{"sid=visitor"}, seen["Cookie"])
}

func TestForwardAuth_requestHeadersTransport(t *testing.T) {
	testCases := []struct {
		desc    string
		config  dynamic.ForwardAuth
		wantErr bool
	}{
		{desc: "public IP in clear text", config: dynamic.ForwardAuth{Address: "http://8.8.8.8/auth"}, wantErr: true},
		{desc: "public name in clear text", config: dynamic.ForwardAuth{Address: "http://auth.example.com/auth"}, wantErr: true},
		{desc: "h2c to a public IP", config: dynamic.ForwardAuth{Address: "h2c://8.8.8.8/auth"}, wantErr: true},
		{desc: "private IP", config: dynamic.ForwardAuth{Address: "http://10.0.0.10:9000/auth"}},
		{desc: "private IPv6", config: dynamic.ForwardAuth{Address: "http://[fd00::1]:9000/auth"}},
		{desc: "loopback", config: dynamic.ForwardAuth{Address: "http://127.0.0.1:9000/auth"}},
		{desc: "localhost", config: dynamic.ForwardAuth{Address: "http://localhost:9000/auth"}},
		{desc: "https", config: dynamic.ForwardAuth{Address: "https://auth.example.com/auth"}},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			cfg := test.config
			cfg.RequestHeaders = map[string]string{"Authorization": "Bearer my-key"}
			cfg.AuthRequestHeaders = []string{"User-Agent"}

			_, err := NewForward(t.Context(), backendEcho(), cfg, nil, "bot")
			if test.wantErr {
				require.ErrorContains(t, err, "clear text")
				return
			}
			require.NoError(t, err)
		})
	}

	// Without requestHeaders, a public address in clear text stays allowed, as in Traefik.
	_, err := NewForward(t.Context(), backendEcho(), dynamic.ForwardAuth{Address: "http://8.8.8.8/auth"}, nil, "bot")
	require.NoError(t, err)
}

func TestForwardAuth_requestHeadersAuthorizationWarning(t *testing.T) {
	testCases := []struct {
		desc               string
		authRequestHeaders []string
		warn               bool
	}{
		{desc: "all the client headers forwarded", warn: true},
		{desc: "a selection forwarded", authRequestHeaders: []string{"User-Agent"}},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := zerolog.New(&buf).WithContext(context.Background())

			_, err := NewForward(ctx, backendEcho(), dynamic.ForwardAuth{
				Address:            "http://127.0.0.1:9000/auth",
				RequestHeaders:     map[string]string{"authorization": "Bearer my-key"},
				AuthRequestHeaders: test.authRequestHeaders,
			}, nil, "bot")
			require.NoError(t, err)

			assert.Equal(t, test.warn, strings.Contains(buf.String(), "requestHeaders sets Authorization"))
		})
	}
}
