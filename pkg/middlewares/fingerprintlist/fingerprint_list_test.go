package fingerprintlist

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/fingerprint"
)

var (
	curl = fingerprint.Fingerprints{
		JA3:  "a0e9f5d64349fb13191bc781f81f42e1",
		JA4:  "t13d4907h2_0d8feac7bc37_7395dae3b2f3",
		JA4H: "ge11nn030000_fe444ad14866_000000000000_000000000000",
		JA4T: "64240_2-4-8-1-3_1460_7",
	}
	chrome = fingerprint.Fingerprints{
		JA3:   "cd08e31494f9531f560d64c695473da9",
		JA4:   "t13d1516h2_8daaf6152771_e5627efa2ab1",
		JA4H:  "ge20cr08enus_308e4456ec31_b50932941748_1a7d4e15bf8f",
		HTTP2: "1:65536;2:0;4:6291456;6:262144|15663105|0|m,a,s,p",
	}
)

func TestNew_validation(t *testing.T) {
	testCases := []struct {
		desc   string
		config dynamic.FingerprintList
		err    string
	}{
		{desc: "valid", config: dynamic.FingerprintList{Mode: "deny", JA4: []string{"t13d*"}}},
		{desc: "no mode", config: dynamic.FingerprintList{JA4: []string{"t13d*"}}, err: "invalid mode"},
		{desc: "unknown mode", config: dynamic.FingerprintList{Mode: "block", JA4: []string{"t13d*"}}, err: "invalid mode"},
		{desc: "no fingerprint", config: dynamic.FingerprintList{Mode: "allow"}, err: "no fingerprint listed"},
		{desc: "lone wildcard", config: dynamic.FingerprintList{Mode: "allow", JA4: []string{"*"}}, err: "must not be empty"},
		{desc: "invalid status", config: dynamic.FingerprintList{Mode: "deny", JA3: []string{"x"}, RejectStatusCode: 999}, err: "invalid HTTP status code"},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			_, err := New(t.Context(), http.NotFoundHandler(), test.config, "list")
			if test.err != "" {
				require.ErrorContains(t, err, test.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestFingerprintList(t *testing.T) {
	testCases := []struct {
		desc     string
		config   dynamic.FingerprintList
		request  *fingerprint.Fingerprints
		h2       bool
		expected int
	}{
		{
			desc:     "deny a listed JA4",
			config:   dynamic.FingerprintList{Mode: "deny", JA4: []string{curl.JA4}},
			request:  &curl,
			expected: http.StatusForbidden,
		},
		{
			desc:     "deny a JA4 prefix, with a custom status",
			config:   dynamic.FingerprintList{Mode: "deny", JA4: []string{"t13d4907*"}, RejectStatusCode: http.StatusTooManyRequests},
			request:  &curl,
			expected: http.StatusTooManyRequests,
		},
		{
			desc:     "deny lets an unlisted fingerprint through",
			config:   dynamic.FingerprintList{Mode: "deny", JA4: []string{curl.JA4}, JA3: []string{curl.JA3}},
			request:  &chrome,
			expected: http.StatusOK,
		},
		{
			desc:     "deny by JA4H",
			config:   dynamic.FingerprintList{Mode: "deny", JA4H: []string{"ge11nn*"}},
			request:  &curl,
			expected: http.StatusForbidden,
		},
		{
			desc:     "deny by JA3",
			config:   dynamic.FingerprintList{Mode: "deny", JA3: []string{chrome.JA3}},
			request:  &chrome,
			expected: http.StatusForbidden,
		},
		{
			desc:     "deny by HTTP/2 fingerprint",
			config:   dynamic.FingerprintList{Mode: "deny", HTTP2: []string{chrome.HTTP2}},
			request:  &chrome,
			h2:       true,
			expected: http.StatusForbidden,
		},
		{
			desc:     "deny by JA4T",
			config:   dynamic.FingerprintList{Mode: "deny", JA4T: []string{"64240_2-4-8-1-3_*"}},
			request:  &curl,
			expected: http.StatusForbidden,
		},
		{
			desc:     "deny lets a request without JA4T through",
			config:   dynamic.FingerprintList{Mode: "deny", JA4T: []string{"64240_*"}},
			request:  &chrome,
			expected: http.StatusOK,
		},
		{
			desc:     "deny lets a request without fingerprint through",
			config:   dynamic.FingerprintList{Mode: "deny", JA4: []string{"t13*"}},
			expected: http.StatusOK,
		},
		{
			desc:     "allow a listed fingerprint",
			config:   dynamic.FingerprintList{Mode: "allow", JA4: []string{"t13d1516h2_*"}},
			request:  &chrome,
			expected: http.StatusOK,
		},
		{
			desc:     "allow rejects an unlisted fingerprint",
			config:   dynamic.FingerprintList{Mode: "allow", JA4: []string{"t13d1516h2_*"}},
			request:  &curl,
			expected: http.StatusForbidden,
		},
		{
			desc:     "allow rejects a request without fingerprint",
			config:   dynamic.FingerprintList{Mode: "allow", JA4: []string{"t13d1516h2_*"}},
			expected: http.StatusForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) { rw.WriteHeader(http.StatusOK) })
			handler, err := New(t.Context(), next, test.config, "list")
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
			if test.request != nil {
				req = req.WithContext(fingerprint.ContextWithFingerprints(req.Context(), *test.request))
			}
			if test.h2 {
				req.ProtoMajor = 2
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, test.expected, rec.Code)
		})
	}
}
