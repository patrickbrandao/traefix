package server

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/server/router"
	tcprouter "github.com/traefik/traefik/v3/pkg/server/router/tcp"
	tcpsvc "github.com/traefik/traefik/v3/pkg/server/service/tcp"
	"github.com/traefik/traefik/v3/pkg/tcp"
)

// MeshRouters is the part of the mesh that the routing is built with (Traefix): the mesh serves, by name, the routers
// of this instance to the other instances, and chooses where the requests for the routers they share are served.
type MeshRouters interface {
	router.MeshDecider
	tcprouter.MeshDecider
	// The TCP services of the routers learned from the mesh are built by the mesh.
	tcpsvc.TraefixBuilder

	// LocalHTTPRouters returns the routers the other instances may send requests for, by qualified name.
	LocalHTTPRouters() map[string]*dynamic.Router
	// LocalTCPRouters returns the TCP routers the other instances may send connections for, by qualified name.
	LocalTCPRouters() map[string]*dynamic.TCPRouter
	// SwitchHandlers replaces the handlers of these routers, with the configuration they were built from.
	SwitchHandlers(httpHandlers map[string]http.Handler, tcpHandlers map[string]tcp.Handler, conf *runtime.Configuration)
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

// buildMeshHandlers builds the handlers of the HTTP routers that the mesh serves by name.
// It runs before the health checks of the services are launched, which the handlers may add to.
func (f *RouterFactory) buildMeshHandlers(ctx context.Context, routerManager *router.Manager) map[string]http.Handler {
	if f.meshRouters == nil {
		return nil
	}

	return routerManager.BuildNamedHandlers(ctx, f.meshRouters.LocalHTTPRouters())
}

// setMeshTCP gives the TCP managers the choice and the services of the mesh, before the routers are built.
func (f *RouterFactory) setMeshTCP(routerManager *tcprouter.Manager, serviceManager *tcpsvc.Manager) {
	if f.meshRouters == nil {
		return
	}

	routerManager.SetMeshDecider(f.meshRouters)
	serviceManager.SetTraefixBuilder(f.meshRouters)
}

// switchMeshHandlers gives the mesh the handlers of one routing configuration, all at once: a router must not look
// down to the other instances because the handlers of its kind are the ones of the previous configuration.
func (f *RouterFactory) switchMeshHandlers(ctx context.Context, httpHandlers map[string]http.Handler, routerManager *tcprouter.Manager, conf *runtime.Configuration) {
	if f.meshRouters == nil {
		return
	}

	f.meshRouters.SwitchHandlers(httpHandlers, routerManager.BuildNamedHandlers(ctx, f.meshRouters.LocalTCPRouters()), conf)
}
