package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/config/static"
	"github.com/traefik/traefik/v3/pkg/fingerprint"
	httpmuxer "github.com/traefik/traefik/v3/pkg/muxer/http"
	"github.com/traefik/traefik/v3/pkg/server/middleware"
	"github.com/traefik/traefik/v3/pkg/server/service"
	traefiktls "github.com/traefik/traefik/v3/pkg/tls"
)

func TestFingerprintMatchers_entryPoint(t *testing.T) {
	// Only "fp" has the fingerprint option.
	ep, err := fingerprint.NewEntryPoint(t.Context(), "fp", &static.Fingerprint{HeaderPrefix: "X-Traefix-"}, nil, nil)
	require.NoError(t, err)

	rtConf := runtime.NewConfig(dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Services: map[string]*dynamic.Service{
				"svc@file": {LoadBalancer: &dynamic.ServersLoadBalancer{Servers: []dynamic.Server{{URL: "http://127.0.0.1:1"}}}},
			},
			Routers: map[string]*dynamic.Router{
				"bots@file":  {EntryPoints: []string{"fp"}, Service: "svc@file", Rule: "JA4(`t13*`)"},
				"plain@file": {EntryPoints: []string{"web"}, Service: "svc@file", Rule: "Host(`example.com`) && JA4(`t13*`)"},
			},
		},
	})

	transportManager := service.NewTransportManager(nil)
	transportManager.Update(map[string]*dynamic.ServersTransport{"default@internal": {}})
	serviceManager := service.NewManager(rtConf.Services, nil, nil, transportManager, proxyBuilderMock{})
	middlewaresBuilder := middleware.NewBuilder(rtConf.Middlewares, serviceManager, nil)

	parser, err := httpmuxer.NewSyntaxParser()
	require.NoError(t, err)

	routerManager := NewManager(rtConf, serviceManager, middlewaresBuilder, nil, traefiktls.NewManager(nil), parser, []string{})
	handlers := routerManager.BuildHandlers(t.Context(), []string{"fp", "web"}, false)

	assert.Empty(t, rtConf.Routers["bots@file"].Err)
	assert.Equal(t, []string{"the JA4 matcher requires the fingerprint option on the entry point web"}, rtConf.Routers["plain@file"].Err)

	// The rule matches through the fingerprint handler of the entry point, which has no TLS here: no JA4, no match.
	rec := httptest.NewRecorder()
	ep.Handler(handlers["fp"]).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://example.com/", nil))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
