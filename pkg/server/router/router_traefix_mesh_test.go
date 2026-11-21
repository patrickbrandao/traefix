package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	httpmuxer "github.com/traefik/traefik/v3/pkg/muxer/http"
	"github.com/traefik/traefik/v3/pkg/server/middleware"
	"github.com/traefik/traefik/v3/pkg/server/service"
	"github.com/traefik/traefik/v3/pkg/testhelpers"
	traefiktls "github.com/traefik/traefik/v3/pkg/tls"
)

func TestManager_BuildNamedHandlers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(server.Close)

	// The configuration as the routing has it: the middleware of the entry point is on the router.
	rtConf := runtime.NewConfig(dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Services: map[string]*dynamic.Service{
				"app@docker": {LoadBalancer: &dynamic.ServersLoadBalancer{Servers: []dynamic.Server{{URL: server.URL}}}},
			},
			Routers: map[string]*dynamic.Router{
				"app@docker": {EntryPoints: []string{"web"}, Rule: "Host(`app.example.com`)", Service: "app", Middlewares: []string{"entry@file", "own"}},
			},
			Middlewares: map[string]*dynamic.Middleware{
				"entry@file": {Headers: &dynamic.Headers{CustomRequestHeaders: map[string]string{"X-Entry": "ran"}}},
				"own@docker": {Headers: &dynamic.Headers{CustomRequestHeaders: map[string]string{"X-Own": "ran"}}},
			},
		},
	})

	transportManager := service.NewTransportManager(nil)
	transportManager.Update(map[string]*dynamic.ServersTransport{"default@internal": {}})

	serviceManager := service.NewManager(rtConf.Services, nil, nil, transportManager, proxyBuilderMock{})
	middlewaresBuilder := middleware.NewBuilder(rtConf.Middlewares, serviceManager, nil)

	parser, err := httpmuxer.NewSyntaxParser()
	require.NoError(t, err)

	manager := NewManager(rtConf, serviceManager, middlewaresBuilder, nil, traefiktls.NewManager(nil), parser, []string{})

	// The routers as their providers wrote them: with their own middlewares only.
	handlers := manager.BuildNamedHandlers(t.Context(), map[string]*dynamic.Router{
		"app@docker": {
			EntryPoints: []string{"web"}, Rule: "Host(`app.example.com`)", Service: "app", Middlewares: []string{"own"},
			TLS: &dynamic.RouterTLSConfig{Options: "missing"},
		},
		"broken@docker": {Rule: "Host(`broken.example.com`)", Service: "missing"},
	})

	// A router that cannot be built is left out, and does not take the others with it.
	require.Len(t, handlers, 1)
	require.Contains(t, handlers, "app@docker")

	// The rule is not matched again: the request is for the router, whatever its host.
	recorder := httptest.NewRecorder()
	req := testhelpers.MustNewRequest(http.MethodGet, "http://other.example.com/", nil)

	handlers["app@docker"].ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "ran", req.Header.Get("X-Own"))
	// The instance that received the request ran the middlewares of its entry point: they are not run a second time here.
	assert.Empty(t, req.Header.Get("X-Entry"))
}
