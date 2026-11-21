package static

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
)

func TestValidateConfiguration_cluster(t *testing.T) {
	valid := &cluster.Configuration{Redis: &cluster.Redis{}}
	valid.SetDefaults()
	valid.Redis.SetDefaults()

	withoutRedis := &cluster.Configuration{}
	withoutRedis.SetDefaults()

	require.NoError(t, (&Configuration{}).ValidateConfiguration())
	require.NoError(t, (&Configuration{Cluster: valid}).ValidateConfiguration())
	require.Error(t, (&Configuration{Cluster: withoutRedis}).ValidateConfiguration())
}
