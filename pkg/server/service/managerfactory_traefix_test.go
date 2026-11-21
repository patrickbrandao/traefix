package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/config/static"
)

type suffixBuilder struct {
	suffix string
}

func (b suffixBuilder) BuildHTTP(_ context.Context, serviceName string) (http.Handler, error) {
	if !strings.HasSuffix(serviceName, b.suffix) {
		return nil, nil
	}

	return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) { rw.WriteHeader(http.StatusTeapot) }), nil
}

func TestManagerFactory_AddServiceBuilder(t *testing.T) {
	factory := NewManagerFactory(static.Configuration{}, nil, nil, nil, nil, nil, nil)
	factory.AddServiceBuilder(suffixBuilder{suffix: "@mesh"})

	manager := factory.Build(&runtime.Configuration{Services: map[string]*runtime.ServiceInfo{}})

	handler, err := manager.BuildHTTP(t.Context(), "app-docker@mesh")
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusTeapot, recorder.Code)

	// The other services are still built from their configuration, and one that has none is an error.
	_, err = manager.BuildHTTP(t.Context(), "app@docker")
	require.Error(t, err)
}
