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
