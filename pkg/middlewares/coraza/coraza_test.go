package coraza

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	gokitmetrics "github.com/go-kit/kit/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/static"
	"github.com/traefik/traefik/v3/pkg/fingerprint"
)

// counts records the counts by label values.
type counts struct {
	mu     *sync.Mutex
	values map[string]float64
	labels string
}

func (c *counts) With(labelValues ...string) gokitmetrics.Counter {
	return &counts{mu: c.mu, values: c.values, labels: strings.Join(labelValues, ",")}
}

func (c *counts) Add(delta float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[c.labels] += delta
}

func (c *counts) get(labels string) float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.values[labels]
}

func useCounters(t *testing.T) (*counts, *counts) {
	t.Helper()

	requests := &counts{mu: &sync.Mutex{}, values: map[string]float64{}}
	matches := &counts{mu: &sync.Mutex{}, values: map[string]float64{}}
	requestsCounter.Store(gokitmetrics.Counter(requests))
	ruleMatchCounter.Store(gokitmetrics.Counter(matches))
	t.Cleanup(func() {
		requestsCounter.Store(gokitmetrics.Counter(&counts{mu: &sync.Mutex{}, values: map[string]float64{}}))
		ruleMatchCounter.Store(gokitmetrics.Counter(&counts{mu: &sync.Mutex{}, values: map[string]float64{}}))
	})

	return requests, matches
}

func echoBackend() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		rw.Header().Set("Content-Type", "text/plain")
		_, _ = rw.Write([]byte("backend:" + string(body)))
	})
}

func newWAF(t *testing.T, name string, cfg dynamic.Coraza, next http.Handler) http.Handler {
	t.Helper()

	h, err := New(t.Context(), next, cfg, name)
	require.NoError(t, err)
	return h
}

func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCoraza_customRule(t *testing.T) {
	testCases := []struct {
		desc     string
		engine   string
		path     string
		status   int
		body     string
		result   string
		ruleHits float64
	}{
		{desc: "blocked", engine: "On", path: "/admin", status: http.StatusForbidden, result: resultBlocked, ruleHits: 1},
		{desc: "allowed", engine: "On", path: "/public", status: http.StatusOK, body: "backend:", result: resultAllowed},
		{desc: "detected", engine: "DetectionOnly", path: "/admin", status: http.StatusOK, body: "backend:", result: resultDetected, ruleHits: 1},
		{desc: "engine off", engine: "Off", path: "/admin", status: http.StatusOK, body: "backend:", result: resultAllowed},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			requests, matches := useCounters(t)

			h := newWAF(t, "waf-"+test.desc, dynamic.Coraza{Directives: []string{
				"SecRuleEngine " + test.engine,
				`SecRule REQUEST_URI "@streq /admin" "id:101,phase:1,t:lowercase,log,deny,status:403,msg:'admin path'"`,
			}}, echoBackend())

			rec := serve(h, httptest.NewRequest(http.MethodGet, "http://example.com"+test.path, nil))

			assert.Equal(t, test.status, rec.Code)
			assert.Equal(t, test.body, rec.Body.String())
			assert.InDelta(t, 1, requests.get("middleware,waf-"+test.desc+",result,"+test.result), 0)
			assert.InDelta(t, test.ruleHits, matches.get("middleware,waf-"+test.desc+",rule_id,101"), 0)
		})
	}
}

func crsDirectives(engine string) []string {
	return []string{
		"Include @coraza.conf-recommended",
		"Include @crs-setup.conf.example",
		"Include @owasp_crs/*.conf",
		"SecRuleEngine " + engine,
		"SecRequestBodyAccess On",
		"SecResponseBodyAccess Off",
	}
}

func TestCoraza_coreRuleSet(t *testing.T) {
	sqli := "http://example.com/?id=1%27%20OR%20%271%27=%271"

	t.Run("blocks a SQL injection", func(t *testing.T) {
		requests, matches := useCounters(t)
		h := newWAF(t, "crs-on", dynamic.Coraza{CRSEnabled: true, Directives: crsDirectives("On")}, echoBackend())

		assert.Equal(t, http.StatusForbidden, serve(h, httptest.NewRequest(http.MethodGet, sqli, nil)).Code)
		assert.Equal(t, http.StatusOK, serve(h, httptest.NewRequest(http.MethodGet, "http://example.com/?id=42", nil)).Code)
		assert.InDelta(t, 1, requests.get("middleware,crs-on,result,blocked"), 0)
		assert.InDelta(t, 1, requests.get("middleware,crs-on,result,allowed"), 0)
		assert.InDelta(t, 1, matches.get("middleware,crs-on,rule_id,949110"), 0)
	})

	t.Run("detects only", func(t *testing.T) {
		requests, _ := useCounters(t)
		h := newWAF(t, "crs-detect", dynamic.Coraza{CRSEnabled: true, Directives: crsDirectives("DetectionOnly")}, echoBackend())

		assert.Equal(t, http.StatusOK, serve(h, httptest.NewRequest(http.MethodGet, sqli, nil)).Code)
		assert.InDelta(t, 1, requests.get("middleware,crs-detect,result,detected"), 0)
	})

	t.Run("needs crsEnabled", func(t *testing.T) {
		_, err := New(t.Context(), echoBackend(), dynamic.Coraza{Directives: crsDirectives("On")}, "crs-off")
		require.ErrorContains(t, err, "requires crsEnabled")
	})
}

func TestCoraza_invalidDirective(t *testing.T) {
	_, err := New(t.Context(), echoBackend(), dynamic.Coraza{Directives: []string{"SecRuleEngine Maybe"}}, "invalid")
	require.Error(t, err)

	_, err = New(t.Context(), echoBackend(), dynamic.Coraza{}, "empty")
	require.ErrorContains(t, err, "no directives")
}

func TestCoraza_requestBody(t *testing.T) {
	_, _ = useCounters(t)

	h := newWAF(t, "body", dynamic.Coraza{Directives: []string{
		"SecRuleEngine On",
		"SecRequestBodyAccess On",
		"SecRequestBodyInMemoryLimit 16",
		"SecRequestBodyLimit 1048576",
		`SecRule REQUEST_BODY "@contains attack" "id:201,phase:2,deny,status:403,msg:'attack in body'"`,
	}}, echoBackend())

	post := func(body string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "http://example.com/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req
	}

	rec := serve(h, post("q=an+attack+payload"))
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// A benign body longer than the in-memory limit reaches the backend whole.
	benign := "q=" + strings.Repeat("benign+", 1000)
	rec = serve(h, post(benign))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "backend:"+benign, rec.Body.String())
}

func TestCoraza_responseBody(t *testing.T) {
	_, _ = useCounters(t)

	h := newWAF(t, "resp", dynamic.Coraza{Directives: []string{
		"SecRuleEngine On",
		"SecResponseBodyAccess On",
		"SecResponseBodyMimeType text/plain",
		`SecRule RESPONSE_BODY "@contains secret-data" "id:301,phase:4,deny,status:502,msg:'leak'"`,
	}}, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		rw.Header().Set("X-Backend", "1")
		_, _ = io.WriteString(rw, "hello ")
		_, _ = io.WriteString(rw, req.URL.Query().Get("say"))
	}))

	rec := serve(h, httptest.NewRequest(http.MethodGet, "http://example.com/?say=secret-data", nil))
	assert.Equal(t, http.StatusBadGateway, rec.Code)
	assert.Empty(t, rec.Body.String())
	assert.Empty(t, rec.Header().Get("X-Backend"))

	rec = serve(h, httptest.NewRequest(http.MethodGet, "http://example.com/?say=world", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello world", rec.Body.String())
	assert.Equal(t, "1", rec.Header().Get("X-Backend"))
}

// flushRecorder records whether a flush reached it, through the http.ResponseController.
type flushRecorder struct {
	*httptest.ResponseRecorder

	flushedAt int
}

func (f *flushRecorder) FlushError() error {
	f.flushedAt = f.Body.Len()
	return nil
}

func TestCoraza_flush(t *testing.T) {
	_, _ = useCounters(t)

	streaming := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(rw, "chunk")
		require.NoError(t, http.NewResponseController(rw).Flush())
	})

	t.Run("without response body inspection, the flush goes through", func(t *testing.T) {
		h := newWAF(t, "flush-off", dynamic.Coraza{Directives: []string{"SecRuleEngine On", "SecResponseBodyAccess Off"}}, streaming)
		rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder(), flushedAt: -1}
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://example.com/", nil))

		assert.Equal(t, len("chunk"), rec.flushedAt)
	})

	t.Run("with response body inspection, the flush waits for the verdict", func(t *testing.T) {
		h := newWAF(t, "flush-on", dynamic.Coraza{Directives: []string{
			"SecRuleEngine On", "SecResponseBodyAccess On", "SecResponseBodyMimeType text/plain",
		}}, streaming)
		rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder(), flushedAt: -1}
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://example.com/", nil))

		assert.Equal(t, -1, rec.flushedAt)
		assert.Equal(t, "chunk", rec.Body.String())
	})
}

func TestCoraza_hijack(t *testing.T) {
	_, _ = useCounters(t)

	h := newWAF(t, "hijack", dynamic.Coraza{Directives: []string{"SecRuleEngine On", "SecResponseBodyAccess On"}},
		http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			conn, brw, err := http.NewResponseController(rw).Hijack()
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = brw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\nhi")
			_ = brw.Flush()
			_ = conn.Close()
		}))

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	conn, err := net.Dial("tcp", strings.TrimPrefix(srv.URL, "http://"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	_, err = io.WriteString(conn, "GET / HTTP/1.1\r\nHost: example.com\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
	require.NoError(t, err)

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
}

type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("client went away") }
func (failingBody) Close() error             { return nil }

func TestCoraza_processingError(t *testing.T) {
	testCases := []struct {
		desc     string
		failOpen bool
		status   int
	}{
		{desc: "fail closed", status: http.StatusInternalServerError},
		{desc: "fail open", failOpen: true, status: http.StatusOK},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			requests, _ := useCounters(t)

			reached := false
			h := newWAF(t, "err-"+test.desc, dynamic.Coraza{FailOpen: test.failOpen, Directives: []string{
				"SecRuleEngine On", "SecRequestBodyAccess On",
			}}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				reached = true
				rw.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodPost, "http://example.com/", nil)
			req.Body = failingBody{}
			req.ContentLength = 10

			assert.Equal(t, test.status, serve(h, req).Code)
			assert.Equal(t, test.failOpen, reached)
			assert.InDelta(t, 1, requests.get("middleware,err-"+test.desc+",result,error"), 0)
		})
	}
}

func TestCoraza_rulesFile(t *testing.T) {
	_, _ = useCounters(t)

	dir := t.TempDir()
	rules := filepath.Join(dir, "rules.conf")
	require.NoError(t, os.WriteFile(rules, []byte(`SecRule REQUEST_HEADERS:User-Agent "@contains sqlmap" "id:401,phase:1,deny,status:403,msg:'scanner'"`+"\n"), 0o600))

	h := newWAF(t, "file", dynamic.Coraza{Directives: []string{"SecRuleEngine On", "Include " + filepath.Join(dir, "*.conf")}}, echoBackend())

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("User-Agent", "sqlmap/1.8")
	assert.Equal(t, http.StatusForbidden, serve(h, req).Code)
	assert.Equal(t, http.StatusOK, serve(h, httptest.NewRequest(http.MethodGet, "http://example.com/", nil)).Code)
}

func TestCoraza_fingerprintHeaders(t *testing.T) {
	_, _ = useCounters(t)

	h := newWAF(t, "fp", dynamic.Coraza{Directives: []string{
		"SecRuleEngine On",
		`SecRule REQUEST_HEADERS:X-Traefix-Proto "@streq h1" "id:501,phase:1,deny,status:429,msg:'h1 client'"`,
	}}, echoBackend())

	// The fingerprint of the entry point runs first and sets its headers, which the rules read.
	ep, err := fingerprint.NewEntryPoint(t.Context(), "websecure", &static.Fingerprint{HeaderPrefix: "X-Traefix-"}, nil, nil)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("X-Traefix-Proto", "h2")
	assert.Equal(t, http.StatusTooManyRequests, serve(ep.Handler(h), req).Code)
}

func TestCoraza_sharedWAF(t *testing.T) {
	cfg := dynamic.Coraza{Directives: []string{"SecRuleEngine On"}}

	cached := func() int {
		wafs.Lock()
		defer wafs.Unlock()
		return len(wafs.byKey)
	}

	before := cached()
	_, err := getWAF("shared", cfg)
	require.NoError(t, err)
	_, err = getWAF("shared", cfg)
	require.NoError(t, err)
	assert.Equal(t, before+1, cached())

	_, err = getWAF("shared", dynamic.Coraza{Directives: []string{"SecRuleEngine DetectionOnly"}})
	require.NoError(t, err)
	assert.Equal(t, before+2, cached())
}

func TestConfigFromPlugin(t *testing.T) {
	testCases := []struct {
		desc     string
		raw      dynamic.PluginConf
		expected dynamic.Coraza
		err      bool
	}{
		{
			desc:     "file",
			raw:      dynamic.PluginConf{"directives": []any{"SecRuleEngine On", "Include @owasp_crs/*.conf"}, "crsEnabled": true},
			expected: dynamic.Coraza{Directives: []string{"SecRuleEngine On", "Include @owasp_crs/*.conf"}, CRSEnabled: true},
		},
		{
			desc:     "labels",
			raw:      dynamic.PluginConf{"directives": map[string]any{"1": "Include @owasp_crs/*.conf", "0": "SecRuleEngine On"}, "crsenabled": "true", "failopen": "true"},
			expected: dynamic.Coraza{Directives: []string{"SecRuleEngine On", "Include @owasp_crs/*.conf"}, CRSEnabled: true, FailOpen: true},
		},
		{
			desc:     "one directive per line",
			raw:      dynamic.PluginConf{"directives": "SecRuleEngine On\n\nSecRequestBodyAccess On\n"},
			expected: dynamic.Coraza{Directives: []string{"SecRuleEngine On", "SecRequestBodyAccess On"}},
		},
		{desc: "unknown option", raw: dynamic.PluginConf{"rules": "x"}, err: true},
		{desc: "invalid boolean", raw: dynamic.PluginConf{"crsEnabled": "maybe"}, err: true},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			cfg, err := ConfigFromPlugin(test.raw)
			if test.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expected, cfg)
		})
	}
}

func TestCoraza_timing(t *testing.T) {
	m := &meter{}
	m.add(time.Now().Add(-time.Millisecond))
	assert.GreaterOrEqual(t, m.elapsed, time.Millisecond)
}
