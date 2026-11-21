package server

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/server/router"
)

// MeshRouters is the part of the mesh that serves, by name, the routers of this instance to the other instances (Traefix).
type MeshRouters interface {
	// LocalHTTPRouters returns the routers the other instances may send requests for, by qualified name.
	LocalHTTPRouters() map[string]*dynamic.Router
	// SwitchHTTPHandlers replaces the handlers of these routers.
	SwitchHTTPHandlers(handlers map[string]http.Handler)
}

// SetMeshRouters makes the factory build the handlers of the routers that the mesh serves by name.
func (f *RouterFactory) SetMeshRouters(mesh MeshRouters) {
	f.meshRouters = mesh
}

func (f *RouterFactory) buildMeshHandlers(ctx context.Context, routerManager *router.Manager) {
	if f.meshRouters == nil {
		return
	}

	f.meshRouters.SwitchHTTPHandlers(routerManager.BuildNamedHandlers(ctx, f.meshRouters.LocalHTTPRouters()))
}
