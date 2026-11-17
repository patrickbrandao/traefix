package accesslog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/containous/alice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/middlewares/capture"
	"github.com/traefik/traefik/v3/pkg/middlewares/observability"
	otypes "github.com/traefik/traefik/v3/pkg/observability/types"
)

func TestMaskClientAddr(t *testing.T) {
	anon := &otypes.AccessLogAnonymization{IPv4Subnet: 24, IPv6Subnet: 80}

	assert.Equal(t, "203.0.113.0:8080", maskClientAddr("203.0.113.42:8080", anon))
	assert.Equal(t, "[2001:db8:0:0:abcd::]:8080", maskClientAddr("[2001:db8::abcd:ffff:c0a8:1]:8080", anon))
	assert.Equal(t, "203.0.113.0", maskClientAddr("203.0.113.42", anon))
}

func TestNewHandlerAnonymizationValidation(t *testing.T) {
	testCases := []struct {
		desc      string
		anon      *otypes.AccessLogAnonymization
		expectErr bool
	}{
		{
			desc: "valid subnets",
			anon: &otypes.AccessLogAnonymization{IPv4Subnet: 24, IPv6Subnet: 80},
		},
		{
			desc:      "IPv4 subnet too large",
			anon:      &otypes.AccessLogAnonymization{IPv4Subnet: 33},
			expectErr: true,
		},
		{
			desc:      "IPv6 subnet too large",
			anon:      &otypes.AccessLogAnonymization{IPv6Subnet: 129},
			expectErr: true,
		},
		{
			desc:      "negative subnet",
			anon:      &otypes.AccessLogAnonymization{IPv4Subnet: -1},
			expectErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			config := &otypes.AccessLog{Format: CommonFormat, Anonymization: test.anon}
			handler, err := NewHandler(context.Background(), config)
			if test.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NoError(t, handler.Close())
		})
	}
}

func TestMaskIPList(t *testing.T) {
	anon := &otypes.AccessLogAnonymization{IPv4Subnet: 24, IPv6Subnet: 48}

	assert.Equal(t, "203.0.113.0", maskIPList("203.0.113.42", anon))
	assert.Equal(t, "203.0.113.0, 198.51.100.0", maskIPList("203.0.113.42, 198.51.100.7", anon))
	assert.Equal(t, "2001:db8:1::", maskIPList("2001:db8:1:2::1", anon))
	assert.Equal(t, "203.0.113.0:1234", maskIPList("203.0.113.42:1234", anon))
	assert.Equal(t, "unknown", maskIPList("unknown", anon))
}

func TestMaskForwarded(t *testing.T) {
	anon := &otypes.AccessLogAnonymization{IPv4Subnet: 24, IPv6Subnet: 48}

	assert.Equal(t, "for=203.0.113.0;proto=https", maskForwarded("for=203.0.113.42;proto=https", anon))
	assert.Equal(t, `for="[2001:db8:1::]:4711", for=198.51.100.0`, maskForwarded(`for="[2001:db8:1:2::1]:4711", for=198.51.100.7`, anon))
	assert.Equal(t, "for=_hidden;by=203.0.113.0", maskForwarded("for=_hidden;by=203.0.113.9", anon))
}

func TestAnonymization_xForwardedFor(t *testing.T) {
	logFilePath := filepath.Join(t.TempDir(), "access.log")
	config := &otypes.AccessLog{
		FilePath:      logFilePath,
		Format:        JSONFormat,
		Anonymization: &otypes.AccessLogAnonymization{IPv4Subnet: 24, IPv6Subnet: 48},
		Fields: &otypes.AccessLogFields{
			DefaultMode: otypes.AccessLogKeep,
			Headers:     &otypes.FieldHeaders{DefaultMode: otypes.AccessLogKeep},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	req.RemoteAddr = "192.0.2.10:4321"
	req.Header.Set("X-Forwarded-For", "203.0.113.42, 10.0.0.1")
	req.Header.Set("X-Real-Ip", "203.0.113.42")
	req.Header.Set("User-Agent", "agent-203.0.113.42")

	handler, err := NewHandler(t.Context(), config)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, handler.Close()) })

	chain, err := alice.New(capture.Wrap).
		Append(func(next http.Handler) (http.Handler, error) {
			return observability.WithObservabilityHandler(next, observability.Observability{AccessLogsEnabled: true}), nil
		}).
		Append(handler.AliceConstructor()).
		Then(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusOK)
		}))
	require.NoError(t, err)
	chain.ServeHTTP(httptest.NewRecorder(), req)

	logData, err := os.ReadFile(logFilePath)
	require.NoError(t, err)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(logData, &entry))

	assert.Equal(t, "203.0.113.0, 10.0.0.0", entry[ClientHost])
	assert.Equal(t, "192.0.2.0:4321", entry[ClientAddr])
	assert.Equal(t, "203.0.113.0, 10.0.0.0", entry["request_X-Forwarded-For"])
	assert.Equal(t, "203.0.113.0", entry["request_X-Real-Ip"])
	assert.Equal(t, "agent-203.0.113.42", entry["request_User-Agent"])
	assert.NotContains(t, string(logData), "203.0.113.42,")
}
