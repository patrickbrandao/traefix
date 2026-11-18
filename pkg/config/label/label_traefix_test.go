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
		DenyRequiresResponseHeader: "X-Fs-Verdict",
		Header:                     &dynamic.ForwardAuthFailOpenHeader{Name: "X-Fs-Verdict", Value: "unavailable"},
	}

	t.Run("labels", func(t *testing.T) {
		cfg, err := DecodeConfiguration(map[string]string{
			"traefik.http.middlewares.bot.forwardauth.address":                             "http://10.0.0.1:9003/auth",
			"traefik.http.middlewares.bot.forwardauth.requestheaders.Authorization":        "Bearer key",
			"traefik.http.middlewares.bot.forwardauth.failopen.denyrequiresresponseheader": "X-Fs-Verdict",
			"traefik.http.middlewares.bot.forwardauth.failopen.header.name":                "X-Fs-Verdict",
			"traefik.http.middlewares.bot.forwardauth.failopen.header.value":               "unavailable",
			"traefik.http.middlewares.empty.forwardauth.address":                           "http://10.0.0.1:9003/auth",
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
        address: http://10.0.0.1:9003/auth
        requestHeaders:
          Authorization: Bearer key
        failOpen:
          denyRequiresResponseHeader: X-Fs-Verdict
          header:
            name: X-Fs-Verdict
            value: unavailable
`, ".yaml", cfg))

		fa := cfg.HTTP.Middlewares["bot"].ForwardAuth
		require.NotNil(t, fa)
		assert.Equal(t, expected, fa.FailOpen)
		assert.Equal(t, map[string]string{"Authorization": "Bearer key"}, fa.RequestHeaders)
	})
}
