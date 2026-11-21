package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/config/static"
)

func TestAddTraefixRoute(t *testing.T) {
	AddTraefixRoute("/api/traefix-test", http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		_, _ = rw.Write([]byte("traefix"))
	}))
	t.Cleanup(func() {
		traefixRoutesMu.Lock()
		defer traefixRoutesMu.Unlock()

		delete(traefixRoutes, "/api/traefix-test")
	})

	handler := NewBuilder(static.Configuration{API: &static.API{}, Global: &static.Global{}}, nil)(&runtime.Configuration{})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	resp, err := http.Get(server.URL + "/api/traefix-test")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "traefix", string(body))

	// The routes of the Traefix features only answer GET, like the rest of the API.
	post, err := http.Post(server.URL+"/api/traefix-test", "text/plain", nil)
	require.NoError(t, err)
	defer func() { _ = post.Body.Close() }()

	assert.Equal(t, http.StatusMethodNotAllowed, post.StatusCode)
}
