package mesh

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/types"
)

// testConfigurations is what the providers of an instance give, before the routing merges it.
func testConfigurations() dynamic.Configurations {
	return dynamic.Configurations{
		"docker": {
			HTTP: &dynamic.HTTPConfiguration{
				Routers: map[string]*dynamic.Router{
					"app": {
						Rule:        "Host(`app.example.com`)",
						EntryPoints: []string{"websecure"},
						Middlewares: []string{"auth"},
						Service:     "app",
						TLS:         &dynamic.RouterTLSConfig{CertResolver: "le", Options: "modern@file", Domains: []types.Domain{{Main: "app.example.com"}}},
						Mesh:        &dynamic.RouterMesh{Priority: 10},
					},
					// Without entry points, a router listens on the default ones.
					"blog":    {Rule: "Host(`blog.example.com`)", Service: "blog", Priority: 42},
					"private": {Rule: "Host(`private.example.com`)", Service: "private", Mesh: &dynamic.RouterMesh{Export: new(false)}},
					"child":   {Rule: "Path(`/child`)", Service: "app", ParentRefs: []string{"app"}},
				},
			},
			TCP: &dynamic.TCPConfiguration{
				Routers: map[string]*dynamic.TCPRouter{
					"pg": {Rule: "HostSNI(`pg.example.com`)", EntryPoints: []string{"postgres"}, Service: "pg", TLS: &dynamic.RouterTCPTLSConfig{Passthrough: true}},
				},
			},
			UDP: &dynamic.UDPConfiguration{
				Routers: map[string]*dynamic.UDPRouter{"resolver": {Service: "resolver"}},
			},
		},
		"file": {
			HTTP: &dynamic.HTTPConfiguration{
				Routers: map[string]*dynamic.Router{
					"wild": {Rule: "Host(`wild.example.com`)", EntryPoints: []string{"web", "web", "admin"}, Service: "wild", Mesh: &dynamic.RouterMesh{Priority: 5000}},
				},
			},
		},
		// The routers of the instance itself, and the ones that came from the mesh, are not announced.
		"internal": {
			HTTP: &dynamic.HTTPConfiguration{Routers: map[string]*dynamic.Router{"dashboard": {Rule: "PathPrefix(`/dashboard`)", Service: "dashboard"}}},
		},
		ProviderName: {
			HTTP: &dynamic.HTTPConfiguration{Routers: map[string]*dynamic.Router{"shop-docker": {Rule: "Host(`shop.example.com`)", Service: "shop-docker"}}},
		},
	}
}

func names(routes []Route) []string {
	var names []string
	for _, route := range routes {
		names = append(names, routeKey(route.Kind, route.Name))
	}

	return names
}

func TestMesh_localRoutes(t *testing.T) {
	node := newTestNode(t, "srv1", testToken, nil)

	routes, local := node.mesh.localRoutes(testConfigurations())

	assert.Equal(t, []string{"http/app@docker", "http/blog@docker", "http/wild@file", "tcp/pg@docker", "udp/resolver@docker"}, names(routes))

	// Every local router counts as local, announced or not: none of them is replaced by a route of another instance.
	assert.Len(t, local, 6)
	assert.Contains(t, local, "http/private@docker")
	assert.NotContains(t, local, "http/child@docker")

	assert.Equal(t, Route{
		Kind:         KindHTTP,
		Name:         "app@docker",
		Rule:         "Host(`app.example.com`)",
		EntryPoints:  []string{"websecure"},
		TLS:          &RouteTLS{CertResolver: "le", Options: "modern@file", Domains: []types.Domain{{Main: "app.example.com"}}},
		MeshPriority: 10,
	}, routes[0])

	assert.Equal(t, []string{"web", "websecure"}, routes[1].EntryPoints)
	assert.Equal(t, 42, routes[1].Priority)
	assert.Nil(t, routes[1].TLS)

	// A priority out of bounds is the default one.
	assert.Equal(t, []string{"admin", "web"}, routes[2].EntryPoints)
	assert.Zero(t, routes[2].MeshPriority)

	assert.Equal(t, &RouteTLS{Passthrough: true}, routes[3].TLS)
	assert.Equal(t, []string{"dns"}, routes[4].EntryPoints)
}

func TestMesh_localRoutes_labeled(t *testing.T) {
	node := newTestNode(t, "srv1", testToken, func(conf *cluster.Configuration) { conf.Mesh.Export = cluster.MeshExportLabeled })

	configurations := testConfigurations()
	configurations["docker"].HTTP.Routers["blog"].Mesh = &dynamic.RouterMesh{Export: new(true)}

	routes, local := node.mesh.localRoutes(configurations)

	assert.Equal(t, []string{"http/blog@docker"}, names(routes))
	assert.Len(t, local, 6)
}

func TestMesh_Transform(t *testing.T) {
	node := newTestNode(t, "srv1", testToken, nil)

	generation := func() uint64 {
		node.mesh.mu.Lock()
		defer node.mesh.mu.Unlock()

		return node.mesh.exportGeneration
	}

	configurations := testConfigurations()

	// The configurations go on to the routing as they came.
	assert.Equal(t, testConfigurations(), node.mesh.Transform(t.Context(), configurations))
	assert.Equal(t, uint64(1), generation())

	// The mesh sees every change of every provider: only a change of the routers counts.
	node.mesh.Transform(t.Context(), testConfigurations())
	assert.Equal(t, uint64(1), generation())

	configurations = testConfigurations()
	configurations["docker"].HTTP.Routers["app"].Middlewares = []string{"auth", "compress"}
	node.mesh.Transform(t.Context(), configurations)
	assert.Equal(t, uint64(1), generation())

	configurations["docker"].HTTP.Routers["app"].Rule = "Host(`www.example.com`)"
	node.mesh.Transform(t.Context(), configurations)
	assert.Equal(t, uint64(2), generation())
}

func TestRoute_signature(t *testing.T) {
	route := Route{Kind: KindHTTP, Name: "app@docker", Rule: "Host(`app.example.com`)", EntryPoints: []string{"web", "websecure"}}

	same := route
	same.Name = "other@file"
	same.MeshPriority = 10
	same.EntryPoints = []string{"websecure", "web"}

	assert.Equal(t, route.signature(), same.signature())
	assert.Len(t, route.signature(), 12)

	for _, change := range []func(*Route){
		func(r *Route) { r.Rule = "Host(`www.example.com`)" },
		func(r *Route) { r.Priority = 10 },
		func(r *Route) { r.EntryPoints = []string{"web"} },
		func(r *Route) { r.TLS = &RouteTLS{} },
		func(r *Route) { r.Kind = KindTCP },
	} {
		other := route
		change(&other)

		assert.NotEqual(t, route.signature(), other.signature())
	}
}

func TestMesh_buildConfiguration(t *testing.T) {
	node := newTestNode(t, "srv1", testToken, nil)

	app := Route{
		Kind: KindHTTP, Name: "app@docker", Rule: "Host(`app.example.com`)", Priority: 7, EntryPoints: []string{"websecure", "admin"},
		TLS: &RouteTLS{CertResolver: "le", Options: "modern@file", Domains: []types.Domain{{Main: "app.example.com"}}},
	}

	changed := app
	changed.Rule = "Host(`www.example.com`)"

	node.mesh.mu.Lock()
	node.mesh.localRouters = map[string]struct{}{"http/blog@docker": {}}
	node.mesh.imported = map[string]*importedTable{
		"srv2": {routes: []Route{
			app,
			{Kind: KindHTTP, Name: "blog@docker", Rule: "Host(`blog.example.com`)", EntryPoints: []string{"web"}},
			{Kind: KindHTTP, Name: "ops@file", Rule: "Host(`ops.example.com`)", EntryPoints: []string{"admin"}},
			{Kind: KindTCP, Name: "pg@docker", Rule: "HostSNI(`*`)", EntryPoints: []string{"web"}},
		}},
		"srv3": {routes: []Route{app}},
		"srv4": {routes: []Route{changed}},
	}
	node.mesh.mu.Unlock()

	conf, report := node.mesh.buildConfiguration()

	// The middlewares and the service of the router stay with the instance that owns it.
	assert.Equal(t, map[string]*dynamic.Router{
		"app-docker": {
			EntryPoints: []string{"websecure"},
			Rule:        "Host(`app.example.com`)",
			Priority:    7,
			Service:     "app-docker",
			TLS:         &dynamic.RouterTLSConfig{CertResolver: "le", Options: "modern@file", Domains: []types.Domain{{Main: "app.example.com"}}},
		},
		"app-docker-" + changed.signature(): {
			EntryPoints: []string{"websecure"},
			Rule:        "Host(`www.example.com`)",
			Priority:    7,
			Service:     "app-docker-" + changed.signature(),
			TLS:         &dynamic.RouterTLSConfig{CertResolver: "le", Options: "modern@file", Domains: []types.Domain{{Main: "app.example.com"}}},
		},
	}, conf.HTTP.Routers)

	assert.Len(t, conf.HTTP.Services, 2)

	byName := map[string]ImportedRoute{}
	for _, imported := range report {
		byName[imported.Kind+"/"+imported.Name+"/"+imported.Rule] = imported
	}

	require.Len(t, byName, 5)

	imported := byName["http/app@docker/Host(`app.example.com`)"]
	assert.Equal(t, ImportImported, imported.Status)
	assert.Equal(t, "app-docker@mesh", imported.Router)
	assert.Equal(t, []string{"srv2", "srv3"}, imported.Peers)

	assert.Equal(t, ImportImported, byName["http/app@docker/Host(`www.example.com`)"].Status)
	assert.Equal(t, []string{"srv4"}, byName["http/app@docker/Host(`www.example.com`)"].Peers)

	assert.Equal(t, ImportLocal, byName["http/blog@docker/Host(`blog.example.com`)"].Status)
	assert.Equal(t, ImportSkipped, byName["http/ops@file/Host(`ops.example.com`)"].Status)
	assert.Contains(t, byName["http/ops@file/Host(`ops.example.com`)"].Reason, "entry points")
	assert.Equal(t, ImportSkipped, byName["tcp/pg@docker/HostSNI(`*`)"].Status)
}

func TestMesh_buildConfiguration_empty(t *testing.T) {
	node := newTestNode(t, "srv1", testToken, nil)

	conf, report := node.mesh.buildConfiguration()

	// The maps are there, so that the routing takes the configuration as a change and drops the routers of before.
	require.NotNil(t, conf.HTTP)
	assert.NotNil(t, conf.HTTP.Routers)
	assert.Empty(t, conf.HTTP.Routers)
	assert.NotNil(t, conf.HTTP.Services)
	assert.Empty(t, report)
}

// nextRouters returns the routers of the next configuration the mesh gives, skipping the ones before the expected one.
func (n *testNode) waitRouters(expected ...string) map[string]*dynamic.Router {
	n.t.Helper()

	var last map[string]*dynamic.Router

	require.Eventually(n.t, func() bool {
		for {
			select {
			case message := <-n.configurations:
				last = message.Configuration.HTTP.Routers
			default:
				if last == nil || len(last) != len(expected) {
					return false
				}

				for _, name := range expected {
					if _, ok := last[name]; !ok {
						return false
					}
				}

				return true
			}
		}
	}, 10*time.Second, 10*time.Millisecond, "expected the routers %v, got %v", expected, last)

	return last
}

func TestMesh_routes(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv2", testToken, nil, first.address)

	first.waitLinked("srv2")
	second.waitLinked("srv1")

	// The second instance discovers routers: the first one gets a router for each of the ones it can serve.
	second.mesh.Transform(t.Context(), testConfigurations())

	routers := first.waitRouters("app-docker", "blog-docker", "wild-file")

	assert.Equal(t, "Host(`app.example.com`)", routers["app-docker"].Rule)
	assert.Equal(t, []string{"websecure"}, routers["app-docker"].EntryPoints)
	assert.Empty(t, routers["app-docker"].Middlewares)
	assert.Equal(t, "app-docker", routers["app-docker"].Service)
	assert.Equal(t, []string{"web"}, routers["wild-file"].EntryPoints)

	// The second instance has these routers itself: it takes nothing back.
	assert.Empty(t, second.mesh.importedFrom("srv1").routes)

	// A router changes, and another one goes.
	configurations := testConfigurations()
	configurations["docker"].HTTP.Routers["app"].Rule = "Host(`www.example.com`)"
	delete(configurations["docker"].HTTP.Routers, "blog")
	second.mesh.Transform(t.Context(), configurations)

	routers = first.waitRouters("app-docker", "wild-file")
	assert.Equal(t, "Host(`www.example.com`)", routers["app-docker"].Rule)

	// The first instance gets a router of the same name: its own is the one that serves.
	first.mesh.Transform(t.Context(), dynamic.Configurations{
		"file": {HTTP: &dynamic.HTTPConfiguration{Routers: map[string]*dynamic.Router{"wild": {Rule: "Host(`wild.example.com`)", Service: "wild"}}}},
	})

	first.waitRouters("app-docker")

	// The instance that owns the routers goes away: so do its routes, at once.
	second.stop()

	first.waitRouters()
}

func TestMesh_routes_afterARestart(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv2", testToken, nil, first.address)

	second.mesh.Transform(t.Context(), testConfigurations())
	first.waitRouters("app-docker", "blog-docker", "wild-file")

	// The instance restarts and its generations start again: what it announces now is taken, whatever its number.
	second.stop()
	first.waitRouters()

	second.start()
	second.mesh.Transform(t.Context(), dynamic.Configurations{
		"docker": {HTTP: &dynamic.HTTPConfiguration{Routers: map[string]*dynamic.Router{"app": {Rule: "Host(`app.example.com`)", Service: "app"}}}},
	})

	first.waitRouters("app-docker")
}

func TestMesh_APIHandler(t *testing.T) {
	first := newTestNode(t, "srv1", testToken, nil)
	second := newTestNode(t, "srv2", testToken, nil, first.address)

	second.mesh.Transform(t.Context(), testConfigurations())
	first.waitRouters("app-docker", "blog-docker", "wild-file")

	recorder := httptest.NewRecorder()
	first.mesh.APIHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/mesh", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var status apiStatus
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &status))

	assert.Equal(t, "srv1", status.NodeID)
	assert.Empty(t, status.Exported)

	require.Len(t, status.Links, 1)
	assert.Equal(t, "srv2", status.Links[0].NodeID)
	assert.True(t, status.Links[0].Up)
	assert.Equal(t, 5, status.Links[0].Routes)

	require.Len(t, status.Imported, 5)
	assert.Equal(t, "app@docker", status.Imported[0].Name)
	assert.Equal(t, ImportImported, status.Imported[0].Status)
	assert.Equal(t, "app-docker@mesh", status.Imported[0].Router)
	assert.Equal(t, []string{"srv2"}, status.Imported[0].Peers)

	// The same question to the instance that owns the routers.
	recorder = httptest.NewRecorder()
	second.mesh.APIHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/mesh", nil))

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &status))
	assert.Len(t, status.Exported, 5)
	assert.Empty(t, status.Imported)
}

func TestServiceBuilder_BuildHTTP(t *testing.T) {
	node := newTestNode(t, "srv1", testToken, nil)

	handler, err := node.mesh.ServiceBuilder().BuildHTTP(t.Context(), "app@docker")
	require.NoError(t, err)
	assert.Nil(t, handler)

	handler, err = node.mesh.ServiceBuilder().BuildHTTP(t.Context(), "app-docker@mesh")
	require.NoError(t, err)
	require.NotNil(t, handler)
}
