package tcp

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	tcpmiddleware "github.com/traefik/traefik/v3/pkg/server/middleware/tcp"
	"github.com/traefik/traefik/v3/pkg/server/service/tcp"
	tcp2 "github.com/traefik/traefik/v3/pkg/tcp"
	traefiktls "github.com/traefik/traefik/v3/pkg/tls"
)

// fakeMesh announces some routers, marks the connections it decides for, and serves the services of the mesh.
type fakeMesh struct {
	local map[string]bool
}

func (m *fakeMesh) LocalTCPRoute(routerName string) bool { return m.local[routerName] }

func (m *fakeMesh) DecideTCP(name string, local tcp2.Handler) tcp2.Handler {
	return tcp2.HandlerFunc(func(conn tcp2.WriteCloser) {
		_, _ = io.WriteString(conn, "decided for "+name+": ")

		local.ServeTCP(conn)
	})
}

func (m *fakeMesh) BuildTCP(_ context.Context, serviceName string) (tcp2.Handler, error) {
	if !strings.HasSuffix(serviceName, "@mesh") {
		return nil, nil
	}

	return tcp2.HandlerFunc(func(conn tcp2.WriteCloser) {
		defer func() { _ = conn.Close() }()

		_, _ = io.WriteString(conn, "forwarded by "+serviceName)
	}), nil
}

// tcpServer starts a server that tells its name to each client, and returns its address.
func tcpServer(t *testing.T, name string) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			_, _ = io.WriteString(conn, name)
			_ = conn.Close()
		}
	}()

	return listener.Addr().String()
}

// askTCP gives the handler the connection of a client, and returns what the client reads before the connection ends.
func askTCP(t *testing.T, handler tcp2.Handler) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	defer func() { _ = listener.Close() }()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		handler.ServeTCP(conn.(*net.TCPConn))
	}()

	client, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)

	defer func() { _ = client.Close() }()

	require.NoError(t, client.SetDeadline(time.Now().Add(10*time.Second)))

	answer, err := io.ReadAll(client)
	require.NoError(t, err)

	return string(answer)
}

func newMeshTestManager(t *testing.T, conf *runtime.Configuration, mesh *fakeMesh) *Manager {
	t.Helper()

	dialerManager := tcp2.NewDialerManager(nil)
	dialerManager.Update(map[string]*dynamic.TCPServersTransport{"default@internal": {}})

	serviceManager := tcp.NewManager(conf, dialerManager)

	tlsManager := traefiktls.NewManager(nil)
	tlsManager.UpdateConfigs(t.Context(), map[string]traefiktls.Store{}, map[string]traefiktls.Options{"default": {}}, []*traefiktls.CertAndStores{})

	manager := NewManager(conf, serviceManager, tcpmiddleware.NewBuilder(conf.TCPMiddlewares), nil, nil, tlsManager, nil)

	if mesh != nil {
		manager.SetMeshDecider(mesh)
		serviceManager.SetTraefixBuilder(mesh)
	}

	return manager
}

func loadBalancer(address string) *runtime.TCPServiceInfo {
	return &runtime.TCPServiceInfo{TCPService: &dynamic.TCPService{
		LoadBalancer: &dynamic.TCPServersLoadBalancer{Servers: []dynamic.TCPServer{{Address: address}}},
	}}
}

func TestManager_mesh(t *testing.T) {
	pg := &dynamic.TCPRouter{EntryPoints: []string{"postgres"}, Rule: "HostSNI(`*`)", Service: "pg"}
	offsite := &dynamic.TCPRouter{EntryPoints: []string{"elsewhere"}, Rule: "HostSNI(`*`)", Service: "offsite"}
	broken := &dynamic.TCPRouter{EntryPoints: []string{"elsewhere"}, Rule: "HostSNI(`*`)", Service: "missing"}

	conf := &runtime.Configuration{
		TCPServices: map[string]*runtime.TCPServiceInfo{
			"pg@docker":      loadBalancer(tcpServer(t, "pg")),
			"offsite@docker": loadBalancer(tcpServer(t, "offsite")),
			"plain@docker":   loadBalancer(tcpServer(t, "plain")),
			// The service of a router learned from the mesh has no configuration.
			"mqtt-docker@mesh": {TCPService: &dynamic.TCPService{}},
		},
		TCPRouters: map[string]*runtime.TCPRouterInfo{
			"pg@docker":        {TCPRouter: pg.DeepCopy(), Status: runtime.StatusEnabled},
			"plain@docker":     {TCPRouter: &dynamic.TCPRouter{EntryPoints: []string{"plain"}, Rule: "HostSNI(`*`)", Service: "plain"}, Status: runtime.StatusEnabled},
			"mqtt-docker@mesh": {TCPRouter: &dynamic.TCPRouter{EntryPoints: []string{"mqtt"}, Rule: "HostSNI(`*`)", Service: "mqtt-docker"}, Status: runtime.StatusEnabled},
		},
	}

	mesh := &fakeMesh{local: map[string]bool{"pg@docker": true}}
	manager := newMeshTestManager(t, conf, mesh)

	entryPoints := manager.BuildHandlers(t.Context(), []string{"postgres", "plain", "mqtt"})
	require.Len(t, entryPoints, 3)

	for name, router := range conf.TCPRouters {
		assert.Empty(t, router.Err, name)
	}

	// A router that the mesh announces lets the mesh choose where its connections are served.
	assert.Equal(t, "decided for pg@docker: pg", askTCP(t, entryPoints["postgres"]))
	// The other routers are served as they are.
	assert.Equal(t, "plain", askTCP(t, entryPoints["plain"]))
	// The mesh builds the service of a router that came from it.
	assert.Equal(t, "forwarded by mqtt-docker@mesh", askTCP(t, entryPoints["mqtt"]))

	named := manager.BuildNamedHandlers(t.Context(), map[string]*dynamic.TCPRouter{
		"pg@docker":      pg,
		"offsite@docker": offsite,
		"broken@docker":  broken,
	})

	require.Len(t, named, 2)

	// A connection that came through the mesh is served here: the mesh does not choose again.
	assert.Equal(t, "pg", askTCP(t, named["pg@docker"]))
	// A router without an entry point on this instance still serves what the other instances receive for it.
	assert.Equal(t, "offsite", askTCP(t, named["offsite@docker"]))
}

func TestManager_withoutMesh(t *testing.T) {
	conf := &runtime.Configuration{
		TCPServices: map[string]*runtime.TCPServiceInfo{"pg@docker": loadBalancer(tcpServer(t, "pg"))},
		TCPRouters: map[string]*runtime.TCPRouterInfo{
			"pg@docker": {TCPRouter: &dynamic.TCPRouter{EntryPoints: []string{"postgres"}, Rule: "HostSNI(`*`)", Service: "pg"}, Status: runtime.StatusEnabled},
		},
	}

	manager := newMeshTestManager(t, conf, nil)

	entryPoints := manager.BuildHandlers(t.Context(), []string{"postgres"})

	assert.Equal(t, "pg", askTCP(t, entryPoints["postgres"]))
}
