package main

import (
	"crypto/x509"
	"encoding/pem"
	"slices"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/api"
	"github.com/traefik/traefik/v3/pkg/config/static"
	"github.com/traefik/traefik/v3/pkg/fingerprint"
	"github.com/traefik/traefik/v3/pkg/mesh"
	"github.com/traefik/traefik/v3/pkg/provider/acme"
	"github.com/traefik/traefik/v3/pkg/provider/aggregator"
	"github.com/traefik/traefik/v3/pkg/safe"
	"github.com/traefik/traefik/v3/pkg/server"
	"github.com/traefik/traefik/v3/pkg/server/service"
)

// wireMesh connects the mesh to the routing, and runs it (Traefix):
// the mesh reads the routers of the local providers, gives the routers of the other instances, and serves their services.
func wireMesh(node *mesh.Mesh, conf *static.Configuration, providerAggregator *aggregator.ProviderAggregator, managerFactory *service.ManagerFactory, routerFactory *server.RouterFactory, watcher *server.ConfigurationWatcher, routinesPool *safe.Pool, acmeProviders []*acme.Provider, httpChallenge acme.HTTPChallengeSolver) error {
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
	node.SetCertificateSource(acmeCertificates(acmeProviders))

	// The CA reaches any instance behind a name: without a Redis server, the instances ask each other for the challenges.
	if solver, ok := httpChallenge.(*acme.ClusterChallengeHTTP); ok {
		node.SetChallengeAnswer(solver.LocalAnswer)
		solver.SetMeshLookup(node.LookupChallenge)
	}
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

// satelliteResolvers keeps the ACME resolvers of a satellite of the mesh from starting (Traefix): a satellite never asks
// a CA for a certificate, the instances it links to do. It returns the configuration the ACME resolvers are made from,
// and the names of the resolvers left out, which the routers of the satellite still name.
func satelliteResolvers(conf *static.Configuration) (*static.Configuration, []string) {
	if conf.Cluster == nil || conf.Cluster.Mesh == nil || !conf.Cluster.Mesh.Satellite {
		return conf, nil
	}

	var names []string

	resolvers := make(map[string]static.CertificateResolver, len(conf.CertificatesResolvers))

	for name, resolver := range conf.CertificatesResolvers {
		if resolver.ACME != nil {
			names = append(names, name)
			continue
		}

		resolvers[name] = resolver
	}

	if len(names) == 0 {
		return conf, nil
	}

	slices.Sort(names)

	log.Warn().Strs("resolvers", names).
		Msg("This instance is a satellite of the mesh: its ACME certificate resolvers do not start, the instances it links to ask for the certificates")

	withoutACME := *conf
	withoutACME.CertificatesResolvers = resolvers

	return &withoutACME, names
}

// acmeCertificates gives the mesh the certificates of the ACME resolvers of this instance, for the satellites that serve
// the domains of the routers they announce.
func acmeCertificates(providers []*acme.Provider) mesh.CertificateSource {
	byName := make(map[string]*acme.Provider, len(providers))
	for _, provider := range providers {
		byName[provider.ResolverName] = provider
	}

	return func(resolver string) ([]mesh.Certificate, error) {
		provider, ok := byName[resolver]
		if !ok || provider.Store == nil {
			return nil, nil
		}

		stored, err := provider.Store.GetCertificates(resolver)
		if err != nil {
			return nil, err
		}

		certificates := make([]mesh.Certificate, 0, len(stored))

		for _, certificate := range stored {
			if certificate == nil || len(certificate.Certificate.Certificate) == 0 || len(certificate.Key) == 0 {
				continue
			}

			certificates = append(certificates, mesh.Certificate{
				Domains:     certificate.Domain.ToStrArray(),
				Certificate: string(certificate.Certificate.Certificate),
				Key:         string(certificate.Key),
				NotAfter:    notAfter(certificate.Certificate.Certificate),
			})
		}

		return certificates, nil
	}
}

// notAfter returns the end of the validity of the first certificate of a PEM chain, or the zero time.
func notAfter(chain []byte) time.Time {
	block, _ := pem.Decode(chain)
	if block == nil {
		return time.Time{}
	}

	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}
	}

	return certificate.NotAfter
}
