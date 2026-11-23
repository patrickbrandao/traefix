package redactor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
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

func TestRemoveCredentials_clusterSecrets(t *testing.T) {
	secrets := []string{"token-0123456789abcdef0123456789abcdef", "next-token-0123456789abcdef0123456789", "valkey-user", "valkey-pass", "sentinel-user", "sentinel-pass"}

	cfg := &static.Configuration{
		Cluster: &cluster.Configuration{
			NodeID:           "node1",
			Token:            secrets[0],
			AdditionalTokens: []string{secrets[1]},
			Redis: &cluster.Redis{
				Endpoints:        []string{"10.0.0.1:6379"},
				Username:         secrets[2],
				Password:         secrets[3],
				SentinelUsername: secrets[4],
				SentinelPassword: secrets[5],
			},
		},
	}

	removed, err := RemoveCredentials(cfg)
	require.NoError(t, err)

	anonymized, err := Anonymize(cfg)
	require.NoError(t, err)

	for _, secret := range secrets {
		assert.NotContains(t, removed, secret)
		assert.NotContains(t, anonymized, secret)
	}

	assert.Contains(t, removed, "node1")
}
