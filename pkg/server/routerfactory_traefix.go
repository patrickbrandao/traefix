package server

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/server/router"
)

// MeshRouters is the part of the mesh that the routing is built with (Traefix): the mesh serves, by name, the routers
// of this instance to the other instances, and chooses where the requests for the routers they share are served.
type MeshRouters interface {
	router.MeshDecider

	// LocalHTTPRouters returns the routers the other instances may send requests for, by qualified name.
	LocalHTTPRouters() map[string]*dynamic.Router
	// SwitchHTTPHandlers replaces the handlers of these routers, with the configuration they were built from.
	SwitchHTTPHandlers(handlers map[string]http.Handler, conf *runtime.Configuration)
}

// SetMeshRouters makes the factory build the handlers of the routers that the mesh serves by name.
func (f *RouterFactory) SetMeshRouters(mesh MeshRouters) {
	f.meshRouters = mesh
}

// setMeshDecider gives the router manager the choice of the mesh, before it builds the routers.
func (f *RouterFactory) setMeshDecider(routerManager *router.Manager) {
	if f.meshRouters != nil {
		routerManager.SetMeshDecider(f.meshRouters)
	}
}

func (f *RouterFactory) buildMeshHandlers(ctx context.Context, routerManager *router.Manager, conf *runtime.Configuration) {
	if f.meshRouters == nil {
		return
	}

	f.meshRouters.SwitchHTTPHandlers(routerManager.BuildNamedHandlers(ctx, f.meshRouters.LocalHTTPRouters()), conf)
}
