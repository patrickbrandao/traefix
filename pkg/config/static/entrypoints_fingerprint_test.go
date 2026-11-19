package static

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/paerser/env"
	"github.com/traefik/paerser/file"
	"github.com/traefik/paerser/flag"
	ptypes "github.com/traefik/paerser/types"
)

func TestFingerprint_decode(t *testing.T) {
	expected := &Fingerprint{
		HeaderPrefix: "X-Edge-",
		HeaderDetail: FingerprintHeaderDetailFull,
		Export: &FingerprintExport{
			URL:           "http://10.0.0.10:9000/ingest",
			Headers:       map[string]string{"Authorization": "Bearer key"},
			RawBytes:      true,
			Fields:        []string{"User-Agent", "Accept", "Accept-Language", "Accept-Encoding", "Sec-CH-UA*", "Sec-Fetch-*"},
			QueueSize:     4096,
			QueueMaxBytes: 16 << 20,
			BatchSize:     32,
			FlushInterval: ptypes.Duration(100 * time.Millisecond),
			SendTimeout:   ptypes.Duration(2 * time.Second),
		},
	}

	testCases := []struct {
		desc   string
		decode func(*Configuration) error
	}{
		{
			desc: "file",
			decode: func(cfg *Configuration) error {
				return file.DecodeContent(`
entryPoints:
  websecure:
    address: ":443"
    fingerprint:
      headerPrefix: X-Edge-
      export:
        url: http://10.0.0.10:9000/ingest
        headers:
          Authorization: Bearer key
        rawBytes: true
        batchSize: 32
`, ".yaml", cfg)
			},
		},
		{
			desc: "flags",
			decode: func(cfg *Configuration) error {
				return flag.Decode([]string{
					"--entrypoints.websecure.address=:443",
					"--entrypoints.websecure.fingerprint.headerprefix=X-Edge-",
					"--entrypoints.websecure.fingerprint.export.url=http://10.0.0.10:9000/ingest",
					"--entrypoints.websecure.fingerprint.export.headers.Authorization=Bearer key",
					"--entrypoints.websecure.fingerprint.export.rawbytes=true",
					"--entrypoints.websecure.fingerprint.export.batchsize=32",
				}, cfg)
			},
		},
		{
			desc: "environment",
			decode: func(cfg *Configuration) error {
				return env.Decode([]string{
					"TRAEFIK_ENTRYPOINTS_WEBSECURE_ADDRESS=:443",
					"TRAEFIK_ENTRYPOINTS_WEBSECURE_FINGERPRINT_HEADERPREFIX=X-Edge-",
					"TRAEFIK_ENTRYPOINTS_WEBSECURE_FINGERPRINT_EXPORT_URL=http://10.0.0.10:9000/ingest",
					"TRAEFIK_ENTRYPOINTS_WEBSECURE_FINGERPRINT_EXPORT_HEADERS_Authorization=Bearer key",
					"TRAEFIK_ENTRYPOINTS_WEBSECURE_FINGERPRINT_EXPORT_RAWBYTES=true",
					"TRAEFIK_ENTRYPOINTS_WEBSECURE_FINGERPRINT_EXPORT_BATCHSIZE=32",
				}, env.DefaultNamePrefix, cfg)
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			cfg := &Configuration{}
			require.NoError(t, test.decode(cfg))

			require.Contains(t, cfg.EntryPoints, "websecure")
			got := cfg.EntryPoints["websecure"].Fingerprint
			require.NotNil(t, got)
			require.NotNil(t, got.Export)

			// The environment variables carry the header names in lowercase, which the export canonicalizes.
			headers := map[string]string{}
			for k, v := range got.Export.Headers {
				headers[http.CanonicalHeaderKey(k)] = v
			}
			got.Export.Headers = headers

			assert.Equal(t, expected, got)
		})
	}
}

func TestFingerprint_decodeEmpty(t *testing.T) {
	testCases := []struct {
		desc   string
		decode func(*Configuration) error
	}{
		{
			desc: "file",
			decode: func(cfg *Configuration) error {
				return file.DecodeContent("entryPoints:\n  web:\n    address: \":80\"\n    fingerprint: {}\n", ".yaml", cfg)
			},
		},
		{
			desc: "flags",
			decode: func(cfg *Configuration) error {
				return flag.Decode([]string{"--entrypoints.web.address=:80", "--entrypoints.web.fingerprint=true"}, cfg)
			},
		},
		{
			desc: "environment",
			decode: func(cfg *Configuration) error {
				return env.Decode([]string{"TRAEFIK_ENTRYPOINTS_WEB_ADDRESS=:80", "TRAEFIK_ENTRYPOINTS_WEB_FINGERPRINT=true"}, env.DefaultNamePrefix, cfg)
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			cfg := &Configuration{}
			require.NoError(t, test.decode(cfg))

			require.Contains(t, cfg.EntryPoints, "web")
			assert.Equal(t, &Fingerprint{HeaderPrefix: DefaultFingerprintHeaderPrefix, HeaderDetail: FingerprintHeaderDetailFull}, cfg.EntryPoints["web"].Fingerprint)
		})
	}
}
