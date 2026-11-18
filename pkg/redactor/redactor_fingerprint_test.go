package redactor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/static"
)

func TestRemoveCredentials_fingerprintExportHeaders(t *testing.T) {
	cfg := &static.Configuration{
		EntryPoints: static.EntryPoints{
			"websecure": &static.EntryPoint{
				Fingerprint: &static.Fingerprint{
					Export: &static.FingerprintExport{
						URL:     "http://10.0.0.1:9000/ingest",
						Headers: map[string]string{"Authorization": "Bearer s3cr3t-key"},
					},
				},
			},
		},
	}

	out, err := RemoveCredentials(cfg)
	require.NoError(t, err)

	assert.NotContains(t, out, "s3cr3t-key")
	assert.Contains(t, out, "http://10.0.0.1:9000/ingest")
}

func TestRemoveCredentials_forwardAuthRequestHeaders(t *testing.T) {
	cfg := &dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Middlewares: map[string]*dynamic.Middleware{
				"bot": {ForwardAuth: &dynamic.ForwardAuth{
					Address:        "http://10.0.0.1:9000/auth",
					RequestHeaders: map[string]string{"Authorization": "Bearer s3cr3t-key"},
				}},
			},
		},
	}

	out, err := RemoveCredentials(cfg)
	require.NoError(t, err)

	assert.NotContains(t, out, "s3cr3t-key")
	assert.Contains(t, out, "http://10.0.0.1:9000/auth")
}
