package static

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/provider/acme"
)

func TestValidateConfiguration_cluster(t *testing.T) {
	valid := &cluster.Configuration{Redis: &cluster.Redis{}}
	valid.SetDefaults()
	valid.Redis.SetDefaults()

	withoutRedis := &cluster.Configuration{}
	withoutRedis.SetDefaults()

	meshOnly := &cluster.Configuration{Mesh: &cluster.Mesh{}}
	meshOnly.SetDefaults()
	meshOnly.Mesh.SetDefaults()
	meshOnly.Token = "0123456789abcdef0123456789abcdef"

	require.NoError(t, (&Configuration{}).ValidateConfiguration())
	require.NoError(t, (&Configuration{Cluster: valid}).ValidateConfiguration())
	require.NoError(t, (&Configuration{Cluster: meshOnly}).ValidateConfiguration())
	require.Error(t, (&Configuration{Cluster: withoutRedis}).ValidateConfiguration())

	// The certificates in the Redis server need the Redis server, which the mesh alone does not bring.
	resolvers := map[string]CertificateResolver{
		"le": {ACME: &acme.Configuration{Storage: "acme.json", CertStore: &acme.CertStore{Primary: acme.StoreRedis}}},
	}

	require.Error(t, (&Configuration{Cluster: meshOnly, CertificatesResolvers: resolvers}).ValidateConfiguration())
}
