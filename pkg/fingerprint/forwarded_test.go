package fingerprint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/middlewares/accesslog"
)

func TestContextWithForwarded(t *testing.T) {
	forwarded := Forwarded{
		ConnID:       "01JCONN",
		Fingerprints: Fingerprints{JA3: "ja3", JA4: "ja4", JA4H: "ja4h", HTTP2: "akamai", JA4T: "ja4t"},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// The request travels between the instances in HTTP/3: the HTTP/2 fingerprint of the client still applies.
	req.ProtoMajor = 3
	req = req.WithContext(ContextWithForwarded(req.Context(), forwarded))

	fingerprints, ok := FingerprintsFromRequest(req)
	require.True(t, ok)
	assert.Equal(t, forwarded.Fingerprints, fingerprints)

	// An instance that forwards the request again would pass on what it was given.
	again, ok := ForwardedFromRequest(req)
	require.True(t, ok)
	assert.Equal(t, forwarded, again)
}

func TestForwardedFromRequest(t *testing.T) {
	_, ok := ForwardedFromRequest(httptest.NewRequest(http.MethodGet, "/", nil))
	assert.False(t, ok)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.ProtoMajor = 2
	req = req.WithContext(ContextWithFingerprints(req.Context(), Fingerprints{JA3: "ja3", JA4: "ja4", JA4H: "ja4h", HTTP2: "akamai"}))

	forwarded, ok := ForwardedFromRequest(req)
	require.True(t, ok)
	assert.Equal(t, Fingerprints{JA3: "ja3", JA4: "ja4", JA4H: "ja4h", HTTP2: "akamai"}, forwarded.Fingerprints)
}

func TestWrapRouterHandler_forwarded(t *testing.T) {
	previous := enabled.Load()
	EnableForwarded()
	t.Cleanup(func() { enabled.Store(previous) })

	require.True(t, Enabled())

	data := &accesslog.LogData{Core: accesslog.CoreLogData{}}

	handler, err := WrapRouterHandler("app@docker", "app@docker")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.ProtoMajor = 3

	ctx := ContextWithForwarded(req.Context(), Forwarded{ConnID: "01JCONN", Fingerprints: Fingerprints{JA4: "ja4", HTTP2: "akamai"}})
	req = req.WithContext(context.WithValue(ctx, accesslog.DataTableKey, data))

	handler.ServeHTTP(httptest.NewRecorder(), req)

	// The access log of the instance that serves the request has what the instance that received it measured.
	assert.Equal(t, "01JCONN", data.Core[AccessLogConnID])
	assert.Equal(t, "ja4", data.Core[AccessLogJA4])
	assert.Equal(t, "akamai", data.Core[AccessLogH2])
	assert.NotContains(t, data.Core, AccessLogJA3)
}
