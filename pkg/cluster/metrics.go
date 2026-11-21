package cluster

import (
	"sync/atomic"

	"github.com/traefik/traefik/v3/pkg/observability/metrics"
)

// The instance joins the cluster before the metrics registries are created, which reach it afterwards.
var metricsRegistry atomic.Pointer[metrics.Registry]

// SetMetricsRegistry sets the registry of the metrics of the cluster.
func SetMetricsRegistry(registry metrics.Registry) {
	if registry != nil {
		metricsRegistry.Store(&registry)
	}
}

func setRedisUp(up bool) {
	registry := metricsRegistry.Load()
	if registry == nil {
		return
	}

	value := 0.0
	if up {
		value = 1
	}

	(*registry).ClusterRedisUpGauge().Set(value)
}

func setNodes(count int) {
	if registry := metricsRegistry.Load(); registry != nil {
		(*registry).ClusterNodesGauge().Set(float64(count))
	}
}
