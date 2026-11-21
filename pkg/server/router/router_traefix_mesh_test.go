package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/middlewares/requestdecorator"
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

// fakeMesh announces one router, and sends its requests away when asked to.
type fakeMesh struct {
	router string
	own    []string
	remote bool

	decided []string
}

func (f *fakeMesh) LocalRoute(routerName, _ string, middlewares []string) (string, int, bool) {
	if routerName != f.router || len(middlewares) < len(f.own) {
		return "", 0, false
	}

	return routerName, len(f.own), true
}

func (f *fakeMesh) Decide(name string, local http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		f.decided = append(f.decided, name)

		if f.remote {
			// What the middlewares of the entry point did is there when the request leaves this instance.
			rw.Header().Set("X-Sent-With-Entry", req.Header.Get("X-Entry"))
			rw.Header().Set("X-Sent-With-Own", req.Header.Get("X-Own"))
			rw.WriteHeader(http.StatusAccepted)

			return
		}

		local.ServeHTTP(rw, req)
	})
}

func TestManager_meshDecision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(server.Close)

	build := func(t *testing.T, mesh *fakeMesh) (*Manager, map[string]http.Handler) {
		t.Helper()

		rtConf := runtime.NewConfig(dynamic.Configuration{
			HTTP: &dynamic.HTTPConfiguration{
				Services: map[string]*dynamic.Service{
					"app@docker": {LoadBalancer: &dynamic.ServersLoadBalancer{Servers: []dynamic.Server{{URL: server.URL}}}},
				},
				Routers: map[string]*dynamic.Router{
					"app@docker":   {EntryPoints: []string{"web"}, Rule: "Host(`app.example.com`)", Service: "app", Middlewares: []string{"entry@file", "own"}},
					"other@docker": {EntryPoints: []string{"web"}, Rule: "Host(`other.example.com`)", Service: "app", Middlewares: []string{"entry@file", "own"}},
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
		manager.SetMeshDecider(mesh)

		return manager, manager.BuildHandlers(t.Context(), []string{"web"}, false)
	}

	ask := func(t *testing.T, handler http.Handler, host string) (*httptest.ResponseRecorder, *http.Request) {
		t.Helper()

		recorder := httptest.NewRecorder()
		req := testhelpers.MustNewRequest(http.MethodGet, "http://"+host+"/", nil)

		// The entry point gives the request its host, which the rules match.
		requestdecorator.New(nil).ServeHTTP(recorder, req, handler.ServeHTTP)

		return recorder, req
	}

	t.Run("served here", func(t *testing.T) {
		mesh := &fakeMesh{router: "app@docker", own: []string{"own@docker"}}
		_, handlers := build(t, mesh)

		recorder, req := ask(t, handlers["web"], "app.example.com")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, []string{"app@docker"}, mesh.decided)
		assert.Equal(t, "ran", req.Header.Get("X-Entry"))
		assert.Equal(t, "ran", req.Header.Get("X-Own"))
	})

	t.Run("sent to another instance", func(t *testing.T) {
		mesh := &fakeMesh{router: "app@docker", own: []string{"own@docker"}, remote: true}
		_, handlers := build(t, mesh)

		recorder, req := ask(t, handlers["web"], "app.example.com")

		// The middleware of the entry point ran before the choice. The one of the router is left to the instance
		// that serves the request.
		assert.Equal(t, http.StatusAccepted, recorder.Code)
		assert.Equal(t, "ran", recorder.Header().Get("X-Sent-With-Entry"))
		assert.Empty(t, recorder.Header().Get("X-Sent-With-Own"))
		assert.Empty(t, req.Header.Get("X-Own"))
	})

	t.Run("a router the mesh does not announce is not concerned", func(t *testing.T) {
		mesh := &fakeMesh{router: "app@docker", own: []string{"own@docker"}, remote: true}
		_, handlers := build(t, mesh)

		recorder, req := ask(t, handlers["web"], "other.example.com")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Empty(t, mesh.decided)
		assert.Equal(t, "ran", req.Header.Get("X-Own"))
	})

	t.Run("a request that came through the mesh is served here", func(t *testing.T) {
		mesh := &fakeMesh{router: "app@docker", own: []string{"own@docker"}, remote: true}
		manager, _ := build(t, mesh)

		handlers := manager.BuildNamedHandlers(t.Context(), map[string]*dynamic.Router{
			"app@docker": {Rule: "Host(`app.example.com`)", Service: "app", Middlewares: []string{"own"}},
		})

		recorder, req := ask(t, handlers["app@docker"], "app.example.com")

		// It makes one hop only: the choice of the mesh is not in its way.
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Empty(t, mesh.decided)
		assert.Equal(t, "ran", req.Header.Get("X-Own"))
	})
}
