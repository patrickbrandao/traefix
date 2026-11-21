package mesh

import (
	"sync/atomic"
	"time"

	"github.com/traefik/traefik/v3/pkg/observability/metrics"
)

// The mesh is prepared before the metrics registries are created, which reach it afterwards.
var metricsRegistry atomic.Pointer[metrics.Registry]

// SetMetricsRegistry sets the registry of the metrics of the mesh.
func SetMetricsRegistry(registry metrics.Registry) {
	if registry != nil {
		metricsRegistry.Store(&registry)
	}
}

// registry returns the registry of this instance: the one of the process, unless a test gave the instance its own.
func (m *Mesh) registry() metrics.Registry {
	if m.metrics != nil {
		return m.metrics
	}

	if registry := metricsRegistry.Load(); registry != nil {
		return *registry
	}

	return nil
}

func (m *Mesh) setLinkUp(peer, transport string, up bool) {
	registry := m.registry()
	if registry == nil {
		return
	}

	value := 0.0
	if up {
		value = 1
	}

	registry.MeshLinkUpGauge().With("peer", peer, "transport", transport).Set(value)
}

func (m *Mesh) setLinkRTT(peer string, rtt time.Duration) {
	if registry := m.registry(); registry != nil {
		registry.MeshLinkRTTGauge().With("peer", peer).Set(rtt.Seconds())
	}
}

func (m *Mesh) setPeers(up, down int) {
	registry := m.registry()
	if registry == nil {
		return
	}

	registry.MeshPeersGauge().With("state", "up").Set(float64(up))
	registry.MeshPeersGauge().With("state", "down").Set(float64(down))
}
