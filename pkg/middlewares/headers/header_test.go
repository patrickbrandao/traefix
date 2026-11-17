package headers

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
)

func TestNewHeader_customRequestHeader(t *testing.T) {
	testCases := []struct {
		desc     string
		cfg      dynamic.Headers
		expected http.Header
	}{
		{
			desc: "adds a header",
			cfg: dynamic.Headers{
				CustomRequestHeaders: map[string]string{
					"X-Custom-Request-Header": "test_request",
				},
			},
			expected: http.Header{"Foo": []string{"bar"}, "X-Custom-Request-Header": []string{"test_request"}},
		},
		{
			desc: "delete a header",
			cfg: dynamic.Headers{
				CustomRequestHeaders: map[string]string{
					"X-Forwarded-For":         "",
					"X-Custom-Request-Header": "",
					"Foo":                     "",
				},
			},
			expected: http.Header{
				"X-Forwarded-For": nil,
			},
		},
		{
			desc: "override a header",
			cfg: dynamic.Headers{
				CustomRequestHeaders: map[string]string{
					"Foo": "test",
				},
			},
			expected: http.Header{"Foo": []string{"test"}},
		},
	}

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			mid, err := NewHeader(emptyHandler, test.cfg)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, "/foo", nil)
			req.Header.Set("Foo", "bar")

			rw := httptest.NewRecorder()

			mid.ServeHTTP(rw, req)

			assert.Equal(t, http.StatusOK, rw.Code)
			assert.Equal(t, test.expected, req.Header)
		})
	}
}

func TestNewHeader_customRequestHeader_Host(t *testing.T) {
	testCases := []struct {
		desc            string
		customHeaders   map[string]string
		expectedHost    string
		expectedURLHost string
	}{
		{
			desc:            "standard Host header",
			customHeaders:   map[string]string{},
			expectedHost:    "example.org",
			expectedURLHost: "example.org",
		},
		{
			desc: "custom Host header",
			customHeaders: map[string]string{
				"Host": "example.com",
			},
			expectedHost:    "example.com",
			expectedURLHost: "example.org",
		},
	}

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			mid, err := NewHeader(emptyHandler, dynamic.Headers{CustomRequestHeaders: test.customHeaders})
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, "http://example.org/foo", nil)

			rw := httptest.NewRecorder()

			mid.ServeHTTP(rw, req)

			assert.Equal(t, http.StatusOK, rw.Code)
			assert.Equal(t, test.expectedHost, req.Host)
			assert.Equal(t, test.expectedURLHost, req.URL.Host)
		})
	}
}

func TestNewHeader_CORSPreflights(t *testing.T) {
	testCases := []struct {
		desc           string
		cfg            dynamic.Headers
		requestHeaders http.Header
		expected       http.Header
	}{
		{
			desc: "Test Simple Preflight",
			cfg: dynamic.Headers{
				AccessControlAllowMethods:    []string{"GET", "OPTIONS", "PUT"},
				AccessControlAllowOriginList: []string{"https://foo.bar.org"},
				AccessControlMaxAge:          new(int64(600)),
			},
			requestHeaders: map[string][]string{
				"Access-Control-Request-Headers": {"origin"},
				"Access-Control-Request-Method":  {"GET", "OPTIONS"},
				"Origin":                         {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Content-Length":               {"0"},
				"Access-Control-Allow-Origin":  {"https://foo.bar.org"},
				"Access-Control-Max-Age":       {"600"},
				"Access-Control-Allow-Methods": {"GET,OPTIONS,PUT"},
			},
		},
		{
			desc: "Wildcard origin Preflight",
			cfg: dynamic.Headers{
				AccessControlAllowMethods:    []string{"GET", "OPTIONS", "PUT"},
				AccessControlAllowOriginList: []string{"*"},
				AccessControlMaxAge:          new(int64(600)),
			},
			requestHeaders: map[string][]string{
				"Access-Control-Request-Headers": {"origin"},
				"Access-Control-Request-Method":  {"GET", "OPTIONS"},
				"Origin":                         {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Content-Length":               {"0"},
				"Access-Control-Allow-Origin":  {"*"},
				"Access-Control-Max-Age":       {"600"},
				"Access-Control-Allow-Methods": {"GET,OPTIONS,PUT"},
			},
		},
		{
			desc: "Allow Credentials Preflight",
			cfg: dynamic.Headers{
				AccessControlAllowMethods:     []string{"GET", "OPTIONS", "PUT"},
				AccessControlAllowOriginList:  []string{"*"},
				AccessControlAllowCredentials: true,
				AccessControlMaxAge:           new(int64(600)),
			},
			requestHeaders: map[string][]string{
				"Access-Control-Request-Headers": {"origin"},
				"Access-Control-Request-Method":  {"GET", "OPTIONS"},
				"Origin":                         {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Content-Length":                   {"0"},
				"Access-Control-Allow-Origin":      {"https://foo.bar.org"},
				"Access-Control-Max-Age":           {"600"},
				"Access-Control-Allow-Methods":     {"GET,OPTIONS,PUT"},
				"Access-Control-Allow-Credentials": {"true"},
			},
		},
		{
			desc: "Allow Headers Preflight",
			cfg: dynamic.Headers{
				AccessControlAllowMethods:    []string{"GET", "OPTIONS", "PUT"},
				AccessControlAllowOriginList: []string{"*"},
				AccessControlAllowHeaders:    []string{"origin", "X-Forwarded-For"},
				AccessControlMaxAge:          new(int64(600)),
			},
			requestHeaders: map[string][]string{
				"Access-Control-Request-Headers": {"origin"},
				"Access-Control-Request-Method":  {"GET", "OPTIONS"},
				"Origin":                         {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Content-Length":               {"0"},
				"Access-Control-Allow-Origin":  {"*"},
				"Access-Control-Max-Age":       {"600"},
				"Access-Control-Allow-Methods": {"GET,OPTIONS,PUT"},
				"Access-Control-Allow-Headers": {"origin,X-Forwarded-For"},
			},
		},
		{
			desc: "No Request Headers Preflight",
			cfg: dynamic.Headers{
				AccessControlAllowMethods:    []string{"GET", "OPTIONS", "PUT"},
				AccessControlAllowOriginList: []string{"*"},
				AccessControlAllowHeaders:    []string{"origin", "X-Forwarded-For"},
				AccessControlMaxAge:          new(int64(600)),
			},
			requestHeaders: map[string][]string{
				"Access-Control-Request-Method": {"GET", "OPTIONS"},
				"Origin":                        {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Content-Length":               {"0"},
				"Access-Control-Allow-Origin":  {"*"},
				"Access-Control-Max-Age":       {"600"},
				"Access-Control-Allow-Methods": {"GET,OPTIONS,PUT"},
				"Access-Control-Allow-Headers": {"origin,X-Forwarded-For"},
			},
		},
		{
			desc: "Nil MaxAge Preflight",
			cfg: dynamic.Headers{
				AccessControlAllowMethods:    []string{"GET", "OPTIONS", "PUT"},
				AccessControlAllowOriginList: []string{"*"},
				AccessControlMaxAge:          nil,
			},
			requestHeaders: map[string][]string{
				"Access-Control-Request-Method": {"GET", "OPTIONS"},
				"Origin":                        {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Content-Length":               {"0"},
				"Access-Control-Allow-Origin":  {"*"},
				"Access-Control-Allow-Methods": {"GET,OPTIONS,PUT"},
			},
		},
		{
			desc: "Non-matching origin Preflight",
			cfg: dynamic.Headers{
				AccessControlAllowMethods:     []string{"GET", "OPTIONS", "PUT"},
				AccessControlAllowOriginList:  []string{"https://foo.bar.org"},
				AccessControlAllowHeaders:     []string{"origin"},
				AccessControlAllowCredentials: true,
				AccessControlExposeHeaders:    []string{"X-Custom"},
				AccessControlMaxAge:           new(int64(600)),
			},
			requestHeaders: map[string][]string{
				"Access-Control-Request-Method": {"GET", "OPTIONS"},
				"Origin":                        {"https://not-foo.bar.org"},
			},
			expected: map[string][]string{
				"Content-Length": {"0"},
			},
		},
	}

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			mid, err := NewHeader(emptyHandler, test.cfg)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodOptions, "/foo", nil)
			req.Header = test.requestHeaders

			rw := httptest.NewRecorder()

			mid.ServeHTTP(rw, req)

			assert.Equal(t, test.expected, rw.Result().Header)
		})
	}
}

func TestNewHeader_CORSResponses(t *testing.T) {
	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	testCases := []struct {
		desc           string
		next           http.Handler
		cfg            dynamic.Headers
		requestHeaders http.Header
		expected       http.Header
		expectedError  bool
	}{
		{
			desc: "Test Simple Request",
			next: emptyHandler,
			cfg: dynamic.Headers{
				AccessControlAllowOriginList: []string{"https://foo.bar.org"},
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Access-Control-Allow-Origin": {"https://foo.bar.org"},
			},
		},
		{
			desc: "Wildcard origin Request",
			next: emptyHandler,
			cfg: dynamic.Headers{
				AccessControlAllowOriginList: []string{"*"},
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Access-Control-Allow-Origin": {"*"},
			},
		},
		{
			desc: "Regexp Origin Request",
			next: emptyHandler,
			cfg: dynamic.Headers{
				AccessControlAllowOriginListRegex: []string{"^https?://([a-z]+)\\.bar\\.org$"},
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Access-Control-Allow-Origin": {"https://foo.bar.org"},
			},
		},
		{
			desc: "Partial Regexp Origin Request",
			next: emptyHandler,
			cfg: dynamic.Headers{
				AccessControlAllowOriginListRegex: []string{"([a-z]+)\\.bar"},
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Access-Control-Allow-Origin": {"https://foo.bar.org"},
			},
		},
		{
			desc: "Regexp Malformed Origin Request",
			next: emptyHandler,
			cfg: dynamic.Headers{
				AccessControlAllowOriginListRegex: []string{"a(b"},
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expectedError: true,
		},
		{
			desc: "Regexp Origin Request without matching",
			next: emptyHandler,
			cfg: dynamic.Headers{
				AccessControlAllowOriginListRegex: []string{"([a-z]+)\\.bar\\.org"},
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://bar.org"},
			},
			expected: map[string][]string{},
		},
		{
			desc: "Empty origin Request",
			next: emptyHandler,
			cfg: dynamic.Headers{
				AccessControlAllowOriginList: []string{"https://foo.bar.org"},
			},
			requestHeaders: map[string][]string{},
			expected:       map[string][]string{},
		},
		{
			desc:           "Not Defined origin Request",
			next:           emptyHandler,
			requestHeaders: map[string][]string{},
			expected:       map[string][]string{},
		},
		{
			desc: "Allow Credentials Request",
			next: emptyHandler,
			cfg: dynamic.Headers{
				AccessControlAllowOriginList:  []string{"*"},
				AccessControlAllowCredentials: true,
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Access-Control-Allow-Origin":      {"https://foo.bar.org"},
				"Access-Control-Allow-Credentials": {"true"},
			},
		},
		{
			desc: "Expose Headers Request",
			next: emptyHandler,
			cfg: dynamic.Headers{
				AccessControlAllowOriginList: []string{"*"},
				AccessControlExposeHeaders:   []string{"origin", "X-Forwarded-For"},
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Access-Control-Allow-Origin":   {"*"},
				"Access-Control-Expose-Headers": {"origin,X-Forwarded-For"},
			},
		},
		{
			desc: "Non-matching origin Request",
			next: emptyHandler,
			cfg: dynamic.Headers{
				AccessControlAllowOriginList:  []string{"https://foo.bar.org"},
				AccessControlAllowCredentials: true,
				AccessControlExposeHeaders:    []string{"X-Custom"},
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://not-foo.bar.org"},
			},
			expected: map[string][]string{},
		},
		{
			desc: "Non-matching origin Request with Vary Headers",
			next: emptyHandler,
			cfg: dynamic.Headers{
				AccessControlAllowOriginList:  []string{"https://foo.bar.org"},
				AccessControlAllowCredentials: true,
				AddVaryHeader:                 true,
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://not-foo.bar.org"},
			},
			expected: map[string][]string{
				"Vary": {"Origin"},
			},
		},
		{
			desc: "Test Simple Request with Vary Headers",
			next: emptyHandler,
			cfg: dynamic.Headers{
				AccessControlAllowOriginList: []string{"https://foo.bar.org"},
				AddVaryHeader:                true,
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Access-Control-Allow-Origin": {"https://foo.bar.org"},
				"Vary":                        {"Origin"},
			},
		},
		{
			desc: "Test Simple Request with Vary Headers and non-empty response",
			next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// nonEmptyHandler
				w.Header().Set("Vary", "Testing")
				w.WriteHeader(http.StatusOK)
			}),
			cfg: dynamic.Headers{
				AccessControlAllowOriginList: []string{"https://foo.bar.org"},
				AddVaryHeader:                true,
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Access-Control-Allow-Origin": {"https://foo.bar.org"},
				"Vary":                        {"Testing,Origin"},
			},
		},
		{
			desc: "Test Simple Request with Vary Headers and existing vary:origin response",
			next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// existingOriginHandler
				w.Header().Set("Vary", "Origin")
				w.WriteHeader(http.StatusOK)
			}),
			cfg: dynamic.Headers{
				AccessControlAllowOriginList: []string{"https://foo.bar.org"},
				AddVaryHeader:                true,
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Access-Control-Allow-Origin": {"https://foo.bar.org"},
				"Vary":                        {"Origin"},
			},
		},
		{
			desc: "Test Simple Request with Vary Headers and multiple existing vary response",
			next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Vary", "Accept-Encoding")
				w.Header().Add("Vary", "Cookie, Authorization")
				w.WriteHeader(http.StatusOK)
			}),
			cfg: dynamic.Headers{
				AccessControlAllowOriginList: []string{"https://foo.bar.org"},
				AddVaryHeader:                true,
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Access-Control-Allow-Origin": {"https://foo.bar.org"},
				"Vary":                        {"Accept-Encoding,Cookie, Authorization,Origin"},
			},
		},
		{
			desc: "Test Simple Request with Vary Headers and existing vary origin in a list",
			next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Vary", "Accept-Encoding")
				w.Header().Add("Vary", "Cookie, origin")
				w.WriteHeader(http.StatusOK)
			}),
			cfg: dynamic.Headers{
				AccessControlAllowOriginList: []string{"https://foo.bar.org"},
				AddVaryHeader:                true,
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Access-Control-Allow-Origin": {"https://foo.bar.org"},
				"Vary":                        {"Accept-Encoding", "Cookie, origin"},
			},
		},
		{
			desc: "Test Simple Request with Vary Headers and a header name containing origin",
			next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Vary", "X-Origin-Id")
				w.WriteHeader(http.StatusOK)
			}),
			cfg: dynamic.Headers{
				AccessControlAllowOriginList: []string{"https://foo.bar.org"},
				AddVaryHeader:                true,
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Access-Control-Allow-Origin": {"https://foo.bar.org"},
				"Vary":                        {"X-Origin-Id,Origin"},
			},
		},
		{
			desc: "Test Simple Request with Vary Headers and existing vary wildcard",
			next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Vary", "*")
				w.WriteHeader(http.StatusOK)
			}),
			cfg: dynamic.Headers{
				AccessControlAllowOriginList: []string{"https://foo.bar.org"},
				AddVaryHeader:                true,
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Access-Control-Allow-Origin": {"https://foo.bar.org"},
				"Vary":                        {"*"},
			},
		},
		{
			desc: "Test Simple Request with non-empty response: set ACAO",
			next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// existingAccessControlAllowOriginHandlerSet
				w.Header().Set("Access-Control-Allow-Origin", "http://foo.bar.org")
				w.WriteHeader(http.StatusOK)
			}),
			cfg: dynamic.Headers{
				AccessControlAllowOriginList: []string{"*"},
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Access-Control-Allow-Origin": {"*"},
			},
		},
		{
			desc: "Test Simple Request with non-empty response: add ACAO",
			next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// existingAccessControlAllowOriginHandlerAdd
				w.Header().Add("Access-Control-Allow-Origin", "http://foo.bar.org")
				w.WriteHeader(http.StatusOK)
			}),
			cfg: dynamic.Headers{
				AccessControlAllowOriginList: []string{"*"},
			},
			requestHeaders: map[string][]string{
				"Origin": {"https://foo.bar.org"},
			},
			expected: map[string][]string{
				"Access-Control-Allow-Origin": {"*"},
			},
		},
		{
			desc: "Test Simple CustomRequestHeaders Not Hijacked by CORS",
			next: emptyHandler,
			cfg: dynamic.Headers{
				CustomRequestHeaders: map[string]string{"foo": "bar"},
			},
			requestHeaders: map[string][]string{
				"Access-Control-Request-Headers": {"origin"},
				"Access-Control-Request-Method":  {"GET", "OPTIONS"},
				"Origin":                         {"https://foo.bar.org"},
			},
			expected: map[string][]string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			mid, err := NewHeader(test.next, test.cfg)
			if test.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, "/foo", nil)
			req.Header = test.requestHeaders

			rw := httptest.NewRecorder()

			mid.ServeHTTP(rw, req)

			assert.Equal(t, test.expected, rw.Result().Header)
		})
	}
}

func TestNewHeader_customResponseHeaders(t *testing.T) {
	testCases := []struct {
		desc     string
		config   map[string]string
		expected http.Header
	}{
		{
			desc: "Test Simple Response",
			config: map[string]string{
				"Testing":  "foo",
				"Testing2": "bar",
			},
			expected: map[string][]string{
				"Foo":      {"bar"},
				"Testing":  {"foo"},
				"Testing2": {"bar"},
			},
		},
		{
			desc: "empty Custom Header",
			config: map[string]string{
				"Testing":  "foo",
				"Testing2": "",
			},
			expected: map[string][]string{
				"Foo":     {"bar"},
				"Testing": {"foo"},
			},
		},
		{
			desc: "Deleting Custom Header",
			config: map[string]string{
				"Testing": "foo",
				"Foo":     "",
			},
			expected: map[string][]string{
				"Testing": {"foo"},
			},
		},
	}

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Foo", "bar")
		w.WriteHeader(http.StatusOK)
	})

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			mid, err := NewHeader(emptyHandler, dynamic.Headers{CustomResponseHeaders: test.config})
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, "/foo", nil)

			rw := httptest.NewRecorder()

			mid.ServeHTTP(rw, req)

			assert.Equal(t, test.expected, rw.Result().Header)
		})
	}
}

func TestNewHeader_replaceRequestHeadersRegex(t *testing.T) {
	testCases := []struct {
		desc     string
		replace  map[string]dynamic.ReplaceHeaderRegex
		custom   map[string]string
		header   http.Header
		expected http.Header
	}{
		{
			desc: "replace a header",
			replace: map[string]dynamic.ReplaceHeaderRegex{
				"Foo": {Regex: "^b(.{2})$", Replacement: "f$1"},
			},
			header:   http.Header{"Foo": {"bar"}},
			expected: http.Header{"Foo": {"far"}},
		},
		{
			desc: "non canonical header name",
			replace: map[string]dynamic.ReplaceHeaderRegex{
				"foo": {Regex: "^b(.{2})$", Replacement: "f$1"},
			},
			header:   http.Header{"Foo": {"bar"}},
			expected: http.Header{"Foo": {"far"}},
		},
		{
			desc: "no match",
			replace: map[string]dynamic.ReplaceHeaderRegex{
				"Foo": {Regex: "^b(.{3})$", Replacement: "f$1"},
			},
			header:   http.Header{"Foo": {"bar"}},
			expected: http.Header{"Foo": {"bar"}},
		},
		{
			desc: "absent header is not added",
			replace: map[string]dynamic.ReplaceHeaderRegex{
				"Too": {Regex: ".*", Replacement: "foo"},
			},
			header:   http.Header{"Foo": {"bar"}},
			expected: http.Header{"Foo": {"bar"}},
		},
		{
			desc: "every value is replaced",
			replace: map[string]dynamic.ReplaceHeaderRegex{
				"Foo": {Regex: "^b", Replacement: "f"},
			},
			header:   http.Header{"Foo": {"bar", "baz", "qux"}},
			expected: http.Header{"Foo": {"far", "faz", "qux"}},
		},
		{
			desc: "value replaced by an empty string is removed",
			replace: map[string]dynamic.ReplaceHeaderRegex{
				"Foo": {Regex: "^bar$", Replacement: ""},
			},
			header:   http.Header{"Foo": {"bar", "baz"}},
			expected: http.Header{"Foo": {"baz"}},
		},
		{
			desc: "header removed when every value is replaced by an empty string",
			replace: map[string]dynamic.ReplaceHeaderRegex{
				"Foo": {Regex: ".+", Replacement: ""},
			},
			header:   http.Header{"Foo": {"bar"}},
			expected: http.Header{},
		},
		{
			desc:   "replacement applied after custom headers",
			custom: map[string]string{"Foo": "bar"},
			replace: map[string]dynamic.ReplaceHeaderRegex{
				"Foo": {Regex: "^b", Replacement: "f"},
			},
			header:   http.Header{},
			expected: http.Header{"Foo": {"far"}},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			mid, err := NewHeader(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {}), dynamic.Headers{
				CustomRequestHeaders:       test.custom,
				ReplaceRequestHeadersRegex: test.replace,
			})
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, "/foo", nil)
			req.Header = test.header

			rw := httptest.NewRecorder()
			mid.ServeHTTP(rw, req)

			assert.Equal(t, http.StatusOK, rw.Code)
			assert.Equal(t, test.expected, req.Header)
		})
	}
}

func TestNewHeader_replaceResponseHeadersRegex(t *testing.T) {
	testCases := []struct {
		desc     string
		replace  map[string]dynamic.ReplaceHeaderRegex
		header   http.Header
		expected http.Header
	}{
		{
			desc: "replace the Location host",
			replace: map[string]dynamic.ReplaceHeaderRegex{
				"Location": {Regex: `^https?://backend\.internal(:\d+)?/`, Replacement: "https://example.com/"},
			},
			header:   http.Header{"Location": {"http://backend.internal:8080/login"}},
			expected: http.Header{"Location": {"https://example.com/login"}},
		},
		{
			desc: "every Set-Cookie is kept",
			replace: map[string]dynamic.ReplaceHeaderRegex{
				"Set-Cookie": {Regex: `; Domain=backend\.internal`, Replacement: "; Domain=example.com"},
			},
			header: http.Header{"Set-Cookie": {
				"a=1; Domain=backend.internal",
				"b=2; Path=/",
			}},
			expected: http.Header{"Set-Cookie": {
				"a=1; Domain=example.com",
				"b=2; Path=/",
			}},
		},
		{
			desc: "absent header is not added",
			replace: map[string]dynamic.ReplaceHeaderRegex{
				"Location": {Regex: ".*", Replacement: "foo"},
			},
			header:   http.Header{"Foo": {"bar"}},
			expected: http.Header{"Foo": {"bar"}},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				maps.Copy(rw.Header(), test.header)
				rw.WriteHeader(http.StatusOK)
			})

			mid, err := NewHeader(next, dynamic.Headers{ReplaceResponseHeadersRegex: test.replace})
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, "/foo", nil)
			rw := httptest.NewRecorder()
			mid.ServeHTTP(rw, req)

			assert.Equal(t, http.StatusOK, rw.Code)
			assert.Equal(t, test.expected, rw.Result().Header)
		})
	}
}

func TestNewHeader_replaceHeadersRegexInvalid(t *testing.T) {
	testCases := []struct {
		desc string
		cfg  dynamic.Headers
	}{
		{
			desc: "invalid request regex",
			cfg: dynamic.Headers{ReplaceRequestHeadersRegex: map[string]dynamic.ReplaceHeaderRegex{
				"Foo": {Regex: "(", Replacement: "bar"},
			}},
		},
		{
			desc: "empty response regex",
			cfg: dynamic.Headers{ReplaceResponseHeadersRegex: map[string]dynamic.ReplaceHeaderRegex{
				"Foo": {Replacement: "bar"},
			}},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			_, err := NewHeader(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {}), test.cfg)
			require.Error(t, err)
		})
	}
}
