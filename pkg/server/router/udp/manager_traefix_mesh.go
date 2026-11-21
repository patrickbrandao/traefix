package udp

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	"github.com/traefik/traefik/v3/pkg/server/provider"
	"github.com/traefik/traefik/v3/pkg/udp"
)

// MeshDecider chooses where a session is served, for the UDP routers that this instance and other instances
// of the mesh both have (Traefix).
type MeshDecider interface {
	// LocalUDPRoute tells whether a router is announced to the mesh.
	LocalUDPRoute(routerName string) bool
	// DecideUDP returns the handler that serves a session with the given handler, on this instance,
	// or sends it to another instance that has the router.
	DecideUDP(name string, local udp.Handler) udp.Handler
}

// SetMeshDecider makes the manager put the choice of the mesh in the routers it builds.
func (m *Manager) SetMeshDecider(mesh MeshDecider) {
	m.mesh = mesh
}

// meshHandler puts the choice of the mesh before the service of a router that the mesh announces (Traefix).
func (m *Manager) meshHandler(routerName string, service udp.Handler) udp.Handler {
	if m.mesh == nil || !m.mesh.LocalUDPRoute(routerName) {
		return service
	}

	return m.mesh.DecideUDP(routerName, service)
}

// BuildNamedHandlers returns, for each router, the handler of its service (Traefix).
// The routers are given as their providers wrote them, by qualified name. A router that cannot be built is left out.
func (m *Manager) BuildNamedHandlers(ctx context.Context, routers map[string]*dynamic.UDPRouter) map[string]udp.Handler {
	handlers := make(map[string]udp.Handler, len(routers))

	for routerName, config := range routers {
		logger := log.Ctx(ctx).With().Str(logs.RouterName, routerName).Logger()
		ctxRouter := logger.WithContext(provider.AddInContext(ctx, routerName))

		handler, err := m.serviceManager.BuildUDP(ctxRouter, config.Service)
		if err != nil {
			// The router itself reports the error on its entry points: this copy only serves the mesh.
			logger.Debug().Err(err).Msg("Unable to build the handler of a router for the mesh")
			continue
		}

		handlers[routerName] = handler
	}

	return handlers
}
