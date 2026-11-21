package udp

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/server/service/udp"
	udp2 "github.com/traefik/traefik/v3/pkg/udp"
)

// fakeMesh announces some routers, marks the sessions it decides for, and serves the services of the mesh.
type fakeMesh struct {
	local map[string]bool
}

func (m *fakeMesh) LocalUDPRoute(routerName string) bool { return m.local[routerName] }

func (m *fakeMesh) DecideUDP(name string, local udp2.Handler) udp2.Handler {
	return udp2.HandlerFunc(func(conn *udp2.Conn) {
		_, _ = conn.Write([]byte("decided for " + name))
		local.ServeUDP(conn)
	})
}

func (m *fakeMesh) BuildUDP(_ context.Context, serviceName string) (udp2.Handler, error) {
	if !strings.HasSuffix(serviceName, "@mesh") {
		return nil, nil
	}

	return udp2.HandlerFunc(func(conn *udp2.Conn) {
		defer func() { _ = conn.Close() }()

		_, _ = conn.Write([]byte("forwarded by " + serviceName))
	}), nil
}

// udpServer starts a server that answers each packet with its name, and returns its address.
func udpServer(t *testing.T, name string) string {
	t.Helper()

	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() { _ = conn.Close() })

	go func() {
		buf := make([]byte, 1500)

		for {
			_, addr, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}

			_, _ = conn.WriteTo([]byte(name), addr)
		}
	}()

	return conn.LocalAddr().String()
}

// askUDP sends a packet to a session the handler serves, and returns the packets that come back.
func askUDP(t *testing.T, handler udp2.Handler, answers int) []string {
	t.Helper()

	listener, err := udp2.Listen(net.ListenConfig{}, "udp", "127.0.0.1:0", time.Second)
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		handler.ServeUDP(conn)
	}()

	client, err := net.DialUDP("udp", nil, listener.Addr().(*net.UDPAddr))
	require.NoError(t, err)

	defer func() { _ = client.Close() }()

	_, err = client.Write([]byte("hello"))
	require.NoError(t, err)

	require.NoError(t, client.SetReadDeadline(time.Now().Add(5*time.Second)))

	var got []string

	buf := make([]byte, 1500)

	for range answers {
		n, err := client.Read(buf)
		require.NoError(t, err)

		got = append(got, string(buf[:n]))
	}

	return got
}

func loadBalancer(address string) *runtime.UDPServiceInfo {
	return &runtime.UDPServiceInfo{UDPService: &dynamic.UDPService{
		LoadBalancer: &dynamic.UDPServersLoadBalancer{Servers: []dynamic.UDPServer{{Address: address}}},
	}}
}

func TestManager_mesh(t *testing.T) {
	conf := &runtime.Configuration{
		UDPServices: map[string]*runtime.UDPServiceInfo{
			"dns@docker":     loadBalancer(udpServer(t, "dns")),
			"syslog@docker":  loadBalancer(udpServer(t, "syslog")),
			"offsite@docker": loadBalancer(udpServer(t, "offsite")),
			// The service of a router learned from the mesh has no configuration.
			"ntp-docker@mesh": {UDPService: &dynamic.UDPService{}},
		},
		UDPRouters: map[string]*runtime.UDPRouterInfo{
			"dns@docker":      {UDPRouter: &dynamic.UDPRouter{EntryPoints: []string{"dns"}, Service: "dns"}, Status: runtime.StatusEnabled},
			"syslog@docker":   {UDPRouter: &dynamic.UDPRouter{EntryPoints: []string{"syslog"}, Service: "syslog"}, Status: runtime.StatusEnabled},
			"ntp-docker@mesh": {UDPRouter: &dynamic.UDPRouter{EntryPoints: []string{"ntp"}, Service: "ntp-docker"}, Status: runtime.StatusEnabled},
		},
	}

	mesh := &fakeMesh{local: map[string]bool{"dns@docker": true}}

	serviceManager := udp.NewManager(conf)
	serviceManager.SetTraefixBuilder(mesh)

	manager := NewManager(conf, serviceManager)
	manager.SetMeshDecider(mesh)

	entryPoints := manager.BuildHandlers(t.Context(), []string{"dns", "syslog", "ntp"})
	require.Len(t, entryPoints, 3)

	for name, router := range conf.UDPRouters {
		assert.Empty(t, router.Err, name)
	}

	// A router that the mesh announces lets the mesh choose where its sessions are served.
	assert.Equal(t, []string{"decided for dns@docker", "dns"}, askUDP(t, entryPoints["dns"], 2))
	// The other routers are served as they are.
	assert.Equal(t, []string{"syslog"}, askUDP(t, entryPoints["syslog"], 1))
	// The mesh builds the service of a router that came from it.
	assert.Equal(t, []string{"forwarded by ntp-docker@mesh"}, askUDP(t, entryPoints["ntp"], 1))

	named := manager.BuildNamedHandlers(t.Context(), map[string]*dynamic.UDPRouter{
		"dns@docker":     {EntryPoints: []string{"dns"}, Service: "dns"},
		"offsite@docker": {EntryPoints: []string{"elsewhere"}, Service: "offsite"},
		"broken@docker":  {EntryPoints: []string{"elsewhere"}, Service: "missing"},
	})

	require.Len(t, named, 2)

	// A session that came through the mesh is served here: the mesh does not choose again.
	assert.Equal(t, []string{"dns"}, askUDP(t, named["dns@docker"], 1))
	assert.Equal(t, []string{"offsite"}, askUDP(t, named["offsite@docker"], 1))
}
