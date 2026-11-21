package label

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
)

func TestDecodeConfiguration_routerMesh(t *testing.T) {
	conf, err := DecodeConfiguration(map[string]string{
		"traefik.http.routers.app.rule":            "Host(`app.example.com`)",
		"traefik.http.routers.app.mesh.priority":   "10",
		"traefik.http.routers.private.rule":        "Host(`private.example.com`)",
		"traefik.http.routers.private.mesh.export": "false",
		"traefik.tcp.routers.pg.rule":              "HostSNI(`*`)",
		"traefik.tcp.routers.pg.mesh.priority":     "20",
		"traefik.tcp.routers.pg.mesh.export":       "true",
		"traefik.udp.routers.dns.entrypoints":      "dns",
		"traefik.udp.routers.dns.mesh.priority":    "30",
	})
	require.NoError(t, err)

	assert.Equal(t, &dynamic.RouterMesh{Priority: 10}, conf.HTTP.Routers["app"].Mesh)
	assert.Equal(t, &dynamic.RouterMesh{Export: new(false)}, conf.HTTP.Routers["private"].Mesh)
	assert.Equal(t, &dynamic.RouterMesh{Priority: 20, Export: new(true)}, conf.TCP.Routers["pg"].Mesh)
	assert.Equal(t, &dynamic.RouterMesh{Priority: 30}, conf.UDP.Routers["dns"].Mesh)

	// A router without the labels of the mesh has nothing of it.
	plain, err := DecodeConfiguration(map[string]string{"traefik.http.routers.app.rule": "Host(`app.example.com`)"})
	require.NoError(t, err)
	assert.Nil(t, plain.HTTP.Routers["app"].Mesh)
}

func TestEncodeConfiguration_routerMesh(t *testing.T) {
	labels, err := EncodeConfiguration(&dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Routers: map[string]*dynamic.Router{
				"app": {Rule: "Host(`app.example.com`)", Mesh: &dynamic.RouterMesh{Priority: 10, Export: new(false)}},
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "10", labels["traefik.HTTP.Routers.app.Mesh.Priority"])
	assert.Equal(t, "false", labels["traefik.HTTP.Routers.app.Mesh.Export"])
}
