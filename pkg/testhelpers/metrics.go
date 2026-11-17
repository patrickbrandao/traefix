package testhelpers

import (
	"sync"

	"github.com/go-kit/kit/metrics"
)

// CollectingCounter is a metrics.Counter implementation that enables access to the CounterValue and LastLabelValues.
type CollectingCounter struct {
	CounterValue    float64
	LastLabelValues []string
}

// With is there to satisfy the metrics.Counter interface.
func (c *CollectingCounter) With(labelValues ...string) metrics.Counter {
	c.LastLabelValues = labelValues
	return c
}

// Add is there to satisfy the metrics.Counter interface.
func (c *CollectingCounter) Add(delta float64) {
	c.CounterValue += delta
}

// CollectingGauge is a metrics.Gauge implementation that enables access to the GaugeValue and LastLabelValues.
// The fields must only be read once the writers are done.
type CollectingGauge struct {
	GaugeValue      float64
	LastLabelValues []string

	// mu serializes the writers, as health checkers update the gauge from several goroutines.
	mu sync.Mutex
}

// With is there to satisfy the metrics.Gauge interface.
func (g *CollectingGauge) With(labelValues ...string) metrics.Gauge {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.LastLabelValues = labelValues
	return g
}

// Set is there to satisfy the metrics.Gauge interface.
func (g *CollectingGauge) Set(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.GaugeValue = value
}

// Add is there to satisfy the metrics.Gauge interface.
func (g *CollectingGauge) Add(delta float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.GaugeValue = delta
}

// CollectingHealthCheckMetrics can be used for testing the Metrics instrumentation of the HealthCheck package.
type CollectingHealthCheckMetrics struct {
	Gauge *CollectingGauge
}

// NewCollectingHealthCheckMetrics creates a new CollectingHealthCheckMetrics instance.
func NewCollectingHealthCheckMetrics() *CollectingHealthCheckMetrics {
	return &CollectingHealthCheckMetrics{&CollectingGauge{}}
}

// BackendServerUpGauge is there to satisfy the healthcheck.metricsRegistry interface.
func (m *CollectingHealthCheckMetrics) BackendServerUpGauge() metrics.Gauge {
	return m.Gauge
}
