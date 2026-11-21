package main

import (
	"slices"

	"github.com/traefik/traefik/v3/pkg/api"
	"github.com/traefik/traefik/v3/pkg/config/static"
	"github.com/traefik/traefik/v3/pkg/fingerprint"
	"github.com/traefik/traefik/v3/pkg/mesh"
	"github.com/traefik/traefik/v3/pkg/provider/aggregator"
	"github.com/traefik/traefik/v3/pkg/safe"
	"github.com/traefik/traefik/v3/pkg/server"
	"github.com/traefik/traefik/v3/pkg/server/service"
)

// wireMesh connects the mesh to the routing, and runs it (Traefix):
// the mesh reads the routers of the local providers, gives the routers of the other instances, and serves their services.
func wireMesh(node *mesh.Mesh, conf *static.Configuration, providerAggregator *aggregator.ProviderAggregator, managerFactory *service.ManagerFactory, routerFactory *server.RouterFactory, watcher *server.ConfigurationWatcher, routinesPool *safe.Pool) error {
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
	node.SetEntryPointSummaries(entryPointSummaries(conf))

	if err := providerAggregator.AddProvider(node.Provider()); err != nil {
		return err
	}

	managerFactory.AddServiceBuilder(node.ServiceBuilder())
	routerFactory.SetMeshRouters(node)

	// The fingerprints of the requests the other instances forward are recorded even without an entry point that measures them here.
	fingerprint.EnableForwarded()
	watcher.AddTransformer(node.Transform)
	api.AddTraefixRoute("/api/mesh", node.APIHandler())

	routinesPool.GoCtx(node.Run)

	return nil
}

// entryPointSummaries describes the entry points for the other instances of the mesh, which compare them with theirs.
func entryPointSummaries(conf *static.Configuration) map[string]mesh.EntryPointSummary {
	summaries := make(map[string]mesh.EntryPointSummary, len(conf.EntryPoints))

	for name, entryPoint := range conf.EntryPoints {
		protocol, err := entryPoint.GetProtocol()
		if err != nil {
			continue
		}

		summaries[name] = mesh.EntryPointSummary{
			Protocol:    protocol,
			Fingerprint: entryPoint.Fingerprint != nil,
			Middlewares: slices.Clone(entryPoint.HTTP.Middlewares),
		}
	}

	return summaries
}
