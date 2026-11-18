package fingerprint

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	httpmuxer "github.com/traefik/traefik/v3/pkg/muxer/http"
)

func TestUsedMatcher(t *testing.T) {
	testCases := []struct {
		rule     string
		expected string
	}{
		{rule: "Host(`example.com`)"},
		{rule: "JA4(`t13d1516h2_8daaf6152771_e5627efa2ab1`)", expected: "JA4"},
		{rule: "Host(`example.com`) && !JA3(`abc`)", expected: "JA3"},
		{rule: "PathPrefix(`/`) || JA4H (`ge11*`)", expected: "JA4H"},
		{rule: "Header(`X-Test`, `JA4(x)`)"},
		{rule: "Header(\"X-Test\", \"JA3(x)\")"},
		{rule: "HostRegexp(`^JA4\\(`)"},
		{rule: "MyJA4(`x`)"},
	}

	for _, test := range testCases {
		t.Run(test.rule, func(t *testing.T) {
			assert.Equal(t, test.expected, usedMatcher(test.rule))
		})
	}
}

func TestCheckRule(t *testing.T) {
	entryPoints.Store("with-fingerprint", struct{}{})
	t.Cleanup(func() { entryPoints.Delete("with-fingerprint") })

	require.NoError(t, CheckRule("with-fingerprint", "JA4(`t13*`)"))
	require.NoError(t, CheckRule("without", "Host(`example.com`)"))
	require.ErrorContains(t, CheckRule("without", "Host(`example.com`) && JA4(`t13*`)"), "the JA4 matcher requires the fingerprint option on the entry point without")
}

func TestMatchers(t *testing.T) {
	parser, err := httpmuxer.NewSyntaxParser()
	require.NoError(t, err)

	f := Fingerprints{
		JA3:  "e7d705a3286e19ea42f587b344ee6865",
		JA4:  "t13d1516h2_8daaf6152771_e5627efa2ab1",
		JA4H: "ge11cr07enus_3c666a2bb02e_86aa86fc6cd2_760a8ef12d1d",
	}

	testCases := []struct {
		rule      string
		state     *Fingerprints
		matches   bool
		wantError bool
	}{
		{rule: "JA4(`t13d1516h2_8daaf6152771_e5627efa2ab1`)", state: &f, matches: true},
		{rule: "JA4(`t13d1516h2_*`)", state: &f, matches: true},
		{rule: "JA4(`t12*`)", state: &f},
		{rule: "JA3(`e7d705a3286e19ea42f587b344ee6865`)", state: &f, matches: true},
		{rule: "JA4H(`ge11cr*`) && !JA4(`q13*`)", state: &f, matches: true},
		{rule: "JA4(`t13*`)", state: &Fingerprints{}},
		{rule: "JA4(`t13*`)"},
		{rule: "JA4(`*`)", wantError: true},
		{rule: "JA4(`a`, `b`)", wantError: true},
	}

	for _, test := range testCases {
		t.Run(test.rule, func(t *testing.T) {
			muxer := httpmuxer.NewMuxer(parser, nil)
			muxer.SetDefaultHandler(http.NotFoundHandler())

			err := muxer.AddRoute(test.rule, "", 0, "", http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				rw.WriteHeader(http.StatusNoContent)
			}))
			if test.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
			if test.state != nil {
				req = req.WithContext(ContextWithFingerprints(req.Context(), *test.state))
			}
			rec := httptest.NewRecorder()
			muxer.ServeHTTP(rec, req)

			assert.Equal(t, test.matches, rec.Code == http.StatusNoContent)
		})
	}
}

func TestParseDomains_withFingerprintMatcher(t *testing.T) {
	domains, err := httpmuxer.ParseDomains("Host(`example.com`) && JA4(`t13*`)")
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com"}, domains)
}
