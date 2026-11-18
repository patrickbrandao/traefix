package label

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/paerser/file"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
)

func TestForwardAuthFailOpen_decode(t *testing.T) {
	expected := &dynamic.ForwardAuthFailOpen{
		DenyRequiresResponseHeader: "X-Verdict",
		Header:                     &dynamic.ForwardAuthFailOpenHeader{Name: "X-Verdict", Value: "unavailable"},
	}

	t.Run("labels", func(t *testing.T) {
		cfg, err := DecodeConfiguration(map[string]string{
			"traefik.http.middlewares.bot.forwardauth.address":                             "http://10.0.0.1:9000/auth",
			"traefik.http.middlewares.bot.forwardauth.requestheaders.Authorization":        "Bearer key",
			"traefik.http.middlewares.bot.forwardauth.failopen.denyrequiresresponseheader": "X-Verdict",
			"traefik.http.middlewares.bot.forwardauth.failopen.header.name":                "X-Verdict",
			"traefik.http.middlewares.bot.forwardauth.failopen.header.value":               "unavailable",
			"traefik.http.middlewares.empty.forwardauth.address":                           "http://10.0.0.1:9000/auth",
			"traefik.http.middlewares.empty.forwardauth.failopen":                          "true",
		})
		require.NoError(t, err)

		fa := cfg.HTTP.Middlewares["bot"].ForwardAuth
		require.NotNil(t, fa)
		assert.Equal(t, expected, fa.FailOpen)
		assert.Equal(t, map[string]string{"Authorization": "Bearer key"}, fa.RequestHeaders)

		assert.Equal(t, &dynamic.ForwardAuthFailOpen{}, cfg.HTTP.Middlewares["empty"].ForwardAuth.FailOpen)
	})

	t.Run("file", func(t *testing.T) {
		cfg := &dynamic.Configuration{}
		require.NoError(t, file.DecodeContent(`
http:
  middlewares:
    bot:
      forwardAuth:
        address: http://10.0.0.1:9000/auth
        requestHeaders:
          Authorization: Bearer key
        failOpen:
          denyRequiresResponseHeader: X-Verdict
          header:
            name: X-Verdict
            value: unavailable
`, ".yaml", cfg))

		fa := cfg.HTTP.Middlewares["bot"].ForwardAuth
		require.NotNil(t, fa)
		assert.Equal(t, expected, fa.FailOpen)
		assert.Equal(t, map[string]string{"Authorization": "Bearer key"}, fa.RequestHeaders)
	})
}

func TestFingerprintList_decode(t *testing.T) {
	cfg, err := DecodeConfiguration(map[string]string{
		"traefik.http.middlewares.bots.fingerprintlist.mode":             "deny",
		"traefik.http.middlewares.bots.fingerprintlist.ja4":              "t13d4907*,t13d1516h2_8daaf6152771_e5627efa2ab1",
		"traefik.http.middlewares.bots.fingerprintlist.ja4h":             "ge11nn*",
		"traefik.http.middlewares.bots.fingerprintlist.rejectstatuscode": "429",
	})
	require.NoError(t, err)

	assert.Equal(t, &dynamic.FingerprintList{
		Mode:             "deny",
		JA4:              []string{"t13d4907*", "t13d1516h2_8daaf6152771_e5627efa2ab1"},
		JA4H:             []string{"ge11nn*"},
		RejectStatusCode: 429,
	}, cfg.HTTP.Middlewares["bots"].FingerprintList)
}

func TestCoraza_decode(t *testing.T) {
	cfg, err := DecodeConfiguration(map[string]string{
		"traefik.http.middlewares.waf.coraza.crsEnabled": "true",
		"traefik.http.middlewares.waf.coraza.directives": "SecRuleEngine On,Include @owasp_crs/*.conf",
	})
	require.NoError(t, err)

	assert.Equal(t, &dynamic.Coraza{
		CRSEnabled: true,
		Directives: []string{"SecRuleEngine On", "Include @owasp_crs/*.conf"},
	}, cfg.HTTP.Middlewares["waf"].Coraza)
}
