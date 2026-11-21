package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/config/static"
	"github.com/traefik/traefik/v3/pkg/provider/acme"
)

func TestSatelliteResolvers(t *testing.T) {
	resolvers := map[string]static.CertificateResolver{
		"le":      {ACME: &acme.Configuration{Email: "ops@example.com"}},
		"staging": {ACME: &acme.Configuration{}},
		"ts":      {Tailscale: &struct{}{}},
	}

	member := &static.Configuration{
		CertificatesResolvers: resolvers,
		Cluster:               &cluster.Configuration{Mesh: &cluster.Mesh{}},
	}

	conf, names := satelliteResolvers(member)
	assert.Same(t, member, conf)
	assert.Empty(t, names)

	satellite := &static.Configuration{
		CertificatesResolvers: resolvers,
		Cluster:               &cluster.Configuration{Mesh: &cluster.Mesh{Satellite: true}},
	}

	// A satellite keeps its other resolvers, and the names of the ACME ones for its routers.
	conf, names = satelliteResolvers(satellite)
	assert.Equal(t, []string{"le", "staging"}, names)
	assert.Equal(t, map[string]static.CertificateResolver{"ts": resolvers["ts"]}, conf.CertificatesResolvers)
	assert.Len(t, satellite.CertificatesResolvers, 3)

	withoutCluster := &static.Configuration{CertificatesResolvers: resolvers}

	conf, names = satelliteResolvers(withoutCluster)
	assert.Same(t, withoutCluster, conf)
	assert.Empty(t, names)
}
