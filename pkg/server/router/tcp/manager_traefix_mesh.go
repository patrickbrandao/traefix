package tcp

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	"github.com/traefik/traefik/v3/pkg/server/provider"
	"github.com/traefik/traefik/v3/pkg/tcp"
)

// MeshDecider chooses where a connection is served, for the TCP routers that this instance and other instances
// of the mesh both have (Traefix).
type MeshDecider interface {
	// LocalTCPRoute tells whether a router is announced to the mesh.
	LocalTCPRoute(routerName string) bool
	// DecideTCP returns the handler that serves a connection with the given handler, on this instance,
	// or sends it to another instance that has the router.
	DecideTCP(name string, local tcp.Handler) tcp.Handler
}

// meshState is what the manager keeps for the mesh.
type meshState struct {
	decider MeshDecider
	// handlers run the middlewares of a router and its service, by the name of the router.
	// They are the ones of the entry points: a service built twice would have two load balancers,
	// and its health check would only follow the last one.
	handlers map[string]tcp.Handler
}

// SetMeshDecider makes the manager put the choice of the mesh in the routers it builds.
func (m *Manager) SetMeshDecider(mesh MeshDecider) {
	m.mesh.decider = mesh
}

type meshRouterKey struct{}

// withMeshRouter records the name of the router that is being built.
func withMeshRouter(ctx context.Context, routerName string) context.Context {
	return context.WithValue(ctx, meshRouterKey{}, routerName)
}

// buildMiddlewaresAndService ends the chain of a router with its middlewares and its service (Traefix).
//
// For a router that the mesh announces, the choice of the mesh comes first: the middlewares of the router run
// on the instance that serves the connection, be it this one or another one.
func (m *Manager) buildMiddlewaresAndService(ctx context.Context, middlewares *tcp.Chain, service tcp.Handler) (tcp.Handler, error) {
	handler, err := tcp.NewChain().Extend(*middlewares).Then(service)
	if err != nil {
		return nil, err
	}

	routerName, ok := ctx.Value(meshRouterKey{}).(string)
	if !ok || m.mesh.decider == nil || !m.mesh.decider.LocalTCPRoute(routerName) {
		return handler, nil
	}

	if m.mesh.handlers == nil {
		m.mesh.handlers = map[string]tcp.Handler{}
	}

	m.mesh.handlers[routerName] = handler

	return m.mesh.decider.DecideTCP(routerName, handler), nil
}

// BuildNamedHandlers returns, for each router, the handler that runs its middlewares and its service (Traefix).
// The routers are given as their providers wrote them, by qualified name. It is called after BuildHandlers:
// a router that an entry point of this instance serves keeps the handler it was given there.
// A router that cannot be built is left out.
func (m *Manager) BuildNamedHandlers(ctx context.Context, routers map[string]*dynamic.TCPRouter) map[string]tcp.Handler {
	handlers := make(map[string]tcp.Handler, len(routers))

	for routerName, config := range routers {
		if handler, ok := m.mesh.handlers[routerName]; ok {
			handlers[routerName] = handler
			continue
		}

		// The router has no entry point on this instance, or is in error on them. The instance that received
		// the connection matched the rule and dealt with TLS: the middlewares and the service are all that is left.
		logger := log.Ctx(ctx).With().Str(logs.RouterName, routerName).Logger()
		ctxRouter := logger.WithContext(provider.AddInContext(ctx, routerName))

		handler, err := m.buildTCPHandler(ctxRouter, &runtime.TCPRouterInfo{TCPRouter: config.DeepCopy()})
		if err != nil {
			// The router itself reports the error on its entry points: this copy only serves the mesh.
			logger.Debug().Err(err).Msg("Unable to build the handler of a router for the mesh")
			continue
		}

		handlers[routerName] = handler
	}

	return handlers
}
