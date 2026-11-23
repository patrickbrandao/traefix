package mesh

import (
	"cmp"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/cluster"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
)

// TestSwarm runs many satellites in this process, with the timers of the real thing, for
// scripts/traefix/e2e-satellite.sh to measure what they cost a central that runs the real binary.
// It runs only when TRAEFIX_MESH_SWARM names the mesh address of the central:
//
//	TRAEFIX_MESH_SWARM        address of the central
//	TRAEFIX_MESH_SWARM_TOKEN  token of the cluster
//	TRAEFIX_MESH_SWARM_HOST   address the satellites listen on, one port each from 20000 (default 127.0.0.1)
//	TRAEFIX_MESH_SWARM_COUNT  how many satellites (default 200)
//	TRAEFIX_MESH_SWARM_HOLD   how long they stay (default 2m)
//
// Satellite n announces the router sat<n>@docker, Host(`sat<n>.example`) on the entry point web, which answers sat<n>.
func TestSwarm(t *testing.T) {
	central := os.Getenv("TRAEFIX_MESH_SWARM")
	if central == "" {
		t.Skip("TRAEFIX_MESH_SWARM is not set")
	}

	count := envInt(t, "TRAEFIX_MESH_SWARM_COUNT", 200)
	host := cmp.Or(os.Getenv("TRAEFIX_MESH_SWARM_HOST"), "127.0.0.1")

	hold := 2 * time.Minute
	if value := os.Getenv("TRAEFIX_MESH_SWARM_HOLD"); value != "" {
		var err error
		hold, err = time.ParseDuration(value)
		require.NoError(t, err)
	}

	// The central derives its authority as the real thing does, and so must the satellites.
	authorityIterations = defaultAuthorityIterations
	t.Cleanup(func() { authorityIterations = 1000 })

	// The derivation takes a while: the satellites are prepared a few at a time, on every CPU.
	satellites := make([]*Mesh, count)
	failures := make(chan error, count)
	slots := make(chan struct{}, runtime.NumCPU())

	var wg sync.WaitGroup

	for n := range count {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()

			m, err := newSwarmSatellite(t, n, central, host)
			if err != nil {
				failures <- err
				return
			}

			satellites[n] = m
		})
	}

	wg.Wait()
	close(failures)

	for err := range failures {
		require.NoError(t, err)
	}

	for _, m := range satellites {
		go m.Run(t.Context())
	}

	t.Logf("%d satellites link to %s", count, central)

	time.Sleep(hold)
}

func envInt(t *testing.T, name string, fallback int) int {
	t.Helper()

	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	n, err := strconv.Atoi(value)
	require.NoError(t, err)

	return n
}

func newSwarmSatellite(t *testing.T, n int, central, host string) (*Mesh, error) {
	t.Helper()

	name := "sat" + strconv.Itoa(n)

	conf := &cluster.Configuration{Mesh: &cluster.Mesh{}}
	conf.SetDefaults()
	conf.Mesh.SetDefaults()
	conf.NodeID = name
	conf.Token = os.Getenv("TRAEFIX_MESH_SWARM_TOKEN")
	conf.Mesh.Satellite = true
	conf.Mesh.Address = net.JoinHostPort(host, strconv.Itoa(20000+n))
	conf.Mesh.Peers = []string{central}

	member, err := cluster.New(t.Context(), conf)
	if err != nil {
		return nil, err
	}

	m, err := New(conf, member)
	if err != nil {
		return nil, err
	}

	m.SetEntryPoints([]string{"web"}, nil, []string{"web"})

	configurations := make(chan dynamic.Message, 10)
	if err := m.Provider().Provide(configurations, nil); err != nil {
		return nil, err
	}

	go func() {
		for range configurations {
		}
	}()

	m.SwitchHandlers(Handlers{HTTP: map[string]http.Handler{name + "@docker": named(name)}}, nil)
	m.Transform(t.Context(), dynamic.Configurations{
		"docker": {HTTP: &dynamic.HTTPConfiguration{Routers: map[string]*dynamic.Router{
			name: {Rule: fmt.Sprintf("Host(`%s.example`)", name), EntryPoints: []string{"web"}, Service: name},
		}}},
	})

	return m, nil
}
