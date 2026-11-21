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

// countRequest counts a request this instance forwards, or could not forward, to another one.
func (m *Mesh) countRequest(peer, kind, result string) {
	if registry := m.registry(); registry != nil {
		registry.MeshRequestsCounter().With("peer", peer, "kind", kind, "result", result).Add(1)
	}
}

// countReceived counts a request another instance forwarded to this one.
func (m *Mesh) countReceived(peer, kind, result string) {
	if registry := m.registry(); registry != nil {
		registry.MeshReceivedCounter().With("peer", peer, "kind", kind, "result", result).Add(1)
	}
}

func (m *Mesh) setConfigMismatch(peer string, differences int) {
	if registry := m.registry(); registry != nil {
		registry.MeshConfigMismatchGauge().With("peer", peer).Set(float64(differences))
	}
}

// setRoutes follows how many routers of each kind another instance announces.
func (m *Mesh) setRoutes(peer string, routes []Route) {
	registry := m.registry()
	if registry == nil {
		return
	}

	counts := countByKind(routes)

	for _, kind := range []string{KindHTTP, KindTCP, KindUDP} {
		registry.MeshRoutesGauge().With("peer", peer, "kind", kind).Set(float64(counts[kind]))
	}
}

// setExportedRoutes follows how many routers of each kind this instance announces.
func (m *Mesh) setExportedRoutes(routes []Route) {
	registry := m.registry()
	if registry == nil {
		return
	}

	counts := countByKind(routes)

	for _, kind := range []string{KindHTTP, KindTCP, KindUDP} {
		registry.MeshExportedRoutesGauge().With("kind", kind).Set(float64(counts[kind]))
	}
}

func countByKind(routes []Route) map[string]int {
	counts := map[string]int{}
	for _, route := range routes {
		counts[route.Kind]++
	}

	return counts
}

// countPackets counts the packets of a flow that ended.
func (m *Mesh) countPackets(peer string, flow *Flow) {
	registry := m.registry()
	if registry == nil {
		return
	}

	counts := flow.counts()

	for _, count := range []struct {
		path, direction string
		value           uint64
	}{
		{pathDatagram, directionSent, counts.sentDatagrams},
		{pathStream, directionSent, counts.sentInStream},
		{pathDatagram, directionReceived, counts.receivedDatagrams},
		{pathStream, directionReceived, counts.receivedInStream},
	} {
		if count.value > 0 {
			registry.MeshUDPPacketsCounter().With("peer", peer, "path", count.path, "direction", count.direction).Add(float64(count.value))
		}
	}
}

const (
	pathDatagram      = "datagram"
	pathStream        = "stream"
	directionSent     = "sent"
	directionReceived = "received"
)
