package router

import (
	"context"
	"net/http"

	"github.com/containous/alice"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/middlewares/recovery"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	"github.com/traefik/traefik/v3/pkg/server/provider"
)

// MeshEntryPoint is the name the requests received from the mesh are observed under: they come through no entry point
// of the instance that serves them (Traefix).
const MeshEntryPoint = "traefix-mesh"

// MeshDecider chooses where a request is served, for the routers that this instance and other instances of the mesh
// both have (Traefix).
type MeshDecider interface {
	// LocalRoute tells whether a router is announced to the mesh. It returns the name it is announced under,
	// and how many of its last middlewares are its own: the ones before them were added by its entry point.
	LocalRoute(routerName, entryPointName string, middlewares []string) (name string, own int, ok bool)
	// Decide returns the handler that serves a request with the given handler, on this instance,
	// or sends it to another instance that has the router.
	Decide(name string, local http.Handler) http.Handler
}

// SetMeshDecider makes the manager put the choice of the mesh in the routers it builds.
func (m *Manager) SetMeshDecider(mesh MeshDecider) {
	m.mesh = mesh
}

// buildMiddlewaresAndService ends the chain of a router with its middlewares and its service (Traefix).
//
// For a router that the mesh announces, the choice of the mesh goes between the middlewares of the entry point,
// which run on the instance that received the request, and the middlewares of the router itself, which run
// on the instance that serves it, be it this one or another one.
func (m *Manager) buildMiddlewaresAndService(ctx context.Context, chain alice.Chain, router *runtime.RouterInfo, entryPointName, routerName string, service http.Handler) (http.Handler, error) {
	// A request that came through the mesh is served here: it was sent to this instance, and makes one hop only.
	if m.mesh != nil && entryPointName != MeshEntryPoint && len(router.ChildRefs) == 0 {
		if name, own, ok := m.mesh.LocalRoute(routerName, entryPointName, router.Middlewares); ok {
			split := len(router.Middlewares) - own

			local, err := m.middlewaresBuilder.BuildMiddlewareChain(ctx, router.Middlewares[split:]).Then(service)
			if err != nil {
				return nil, err
			}

			entryPoint := m.middlewaresBuilder.BuildMiddlewareChain(ctx, router.Middlewares[:split])

			return chain.Extend(*entryPoint).Then(m.mesh.Decide(name, local))
		}
	}

	return chain.Extend(*m.middlewaresBuilder.BuildMiddlewareChain(ctx, router.Middlewares)).Then(service)
}

// BuildNamedHandlers builds, for each router, the handler that runs its own middlewares and its service (Traefix).
// The routers are given as their providers wrote them, by qualified name: the middlewares of the entry points,
// which the instance that received the request already ran, are not part of them.
// A router that cannot be built is left out.
func (m *Manager) BuildNamedHandlers(ctx context.Context, routers map[string]*dynamic.Router) map[string]http.Handler {
	handlers := make(map[string]http.Handler, len(routers))

	for routerName, config := range routers {
		logger := log.Ctx(ctx).With().Str(logs.RouterName, routerName).Logger()
		ctxRouter := logger.WithContext(provider.AddInContext(ctx, routerName))

		router := config.DeepCopy()
		// The instance that received the request matched the rule and terminated TLS.
		router.EntryPoints = nil
		router.TLS = nil

		handler, err := m.buildHTTPHandler(ctxRouter, &runtime.RouterInfo{Router: router}, MeshEntryPoint, routerName)
		if err != nil {
			// The router itself reports the error on its entry points: this copy only serves the mesh.
			logger.Debug().Err(err).Msg("Unable to build the handler of a router for the mesh")
			continue
		}

		observability := dynamic.RouterObservabilityConfig{AccessLogs: new(true), Metrics: new(true), Tracing: new(true)}
		if router.Observability != nil {
			observability = *router.Observability
		}

		chain := alice.New(func(next http.Handler) (http.Handler, error) {
			return recovery.New(ctxRouter, next)
		})

		handler, err = chain.Extend(m.observabilityMgr.BuildEPChain(ctxRouter, MeshEntryPoint, false, observability)).Then(handler)
		if err != nil {
			logger.Debug().Err(err).Msg("Unable to build the handler of a router for the mesh")
			continue
		}

		handlers[routerName] = handler
	}

	return handlers
}
