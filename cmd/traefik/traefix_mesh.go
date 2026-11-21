package main

import (
	"slices"

	"github.com/traefik/traefik/v3/pkg/api"
	"github.com/traefik/traefik/v3/pkg/config/static"
	"github.com/traefik/traefik/v3/pkg/mesh"
	"github.com/traefik/traefik/v3/pkg/provider/aggregator"
	"github.com/traefik/traefik/v3/pkg/safe"
	"github.com/traefik/traefik/v3/pkg/server"
	"github.com/traefik/traefik/v3/pkg/server/service"
)

// wireMesh connects the mesh to the routing, and runs it (Traefix):
// the mesh reads the routers of the local providers, gives the routers of the other instances, and serves their services.
func wireMesh(node *mesh.Mesh, conf *static.Configuration, providerAggregator *aggregator.ProviderAggregator, managerFactory *service.ManagerFactory, watcher *server.ConfigurationWatcher, routinesPool *safe.Pool) error {
	if node == nil {
		return nil
	}

	var entryPoints, udpEntryPoints []string

	for name, entryPoint := range conf.EntryPoints {
		if protocol, err := entryPoint.GetProtocol(); err == nil && protocol == "udp" {
			udpEntryPoints = append(udpEntryPoints, name)
			continue
		}

		entryPoints = append(entryPoints, name)
	}

	slices.Sort(entryPoints)
	slices.Sort(udpEntryPoints)

	node.SetEntryPoints(entryPoints, udpEntryPoints, getDefaultsEntrypoints(conf))

	if err := providerAggregator.AddProvider(node.Provider()); err != nil {
		return err
	}

	managerFactory.AddServiceBuilder(node.ServiceBuilder())
	watcher.AddTransformer(node.Transform)
	api.AddTraefixRoute("/api/mesh", node.APIHandler())

	routinesPool.GoCtx(node.Run)

	return nil
}
