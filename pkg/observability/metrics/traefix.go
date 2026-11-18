package metrics

import (
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/dogstatsd"
	"github.com/go-kit/kit/metrics/influx"
	"github.com/go-kit/kit/metrics/multi"
	"github.com/go-kit/kit/metrics/statsd"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/metric"
)

// The Traefix metrics are kept apart from the upstream ones, so that the merges of the upstream registries only meet
// the single line wiring them in each backend.

const (
	fingerprintExportRecordsName      = MetricNamePrefix + "fingerprint_export_records_total"
	fingerprintExportDroppedName      = MetricNamePrefix + "fingerprint_export_dropped_total"
	fingerprintExportBatchesName      = MetricNamePrefix + "fingerprint_export_batches_total"
	fingerprintExportQueueRecordsName = MetricNamePrefix + "fingerprint_export_queue_records"
	fingerprintExportSendName         = MetricNamePrefix + "fingerprint_export_send_seconds"
	fingerprintCaptureTruncatedName   = MetricNamePrefix + "fingerprint_capture_truncated_total"
	forwardAuthFailOpenName           = MetricNamePrefix + "forwardauth_failopen_total"

	// The StatsD and Datadog clients add the "traefik." prefix themselves.
	statsdFingerprintExportRecordsName      = "fingerprint.export.records.total"
	statsdFingerprintExportDroppedName      = "fingerprint.export.dropped.total"
	statsdFingerprintExportBatchesName      = "fingerprint.export.batches.total"
	statsdFingerprintExportQueueRecordsName = "fingerprint.export.queue.records"
	statsdFingerprintExportSendName         = "fingerprint.export.send.duration"
	statsdFingerprintCaptureTruncatedName   = "fingerprint.capture.truncated.total"
	statsdForwardAuthFailOpenName           = "forwardauth.failopen.total"

	influxDBFingerprintPrefix = "traefik."

	labelEntryPoint = "entrypoint"
	labelCode       = "code"
)

// otlpSecondsUnit is a variable, not a constant, so that the upstream newOTLPHistogramFrom keeps receiving a unit
// unparam does not see as always the same.
var otlpSecondsUnit = "s"

// traefixRegistry is the part of Registry holding the Traefix metrics.
type traefixRegistry interface {
	// FingerprintExportRecordsCounter counts the records queued for export, by entrypoint and type.
	FingerprintExportRecordsCounter() metrics.Counter
	// FingerprintExportDroppedCounter counts the records dropped, by entrypoint and reason.
	FingerprintExportDroppedCounter() metrics.Counter
	// FingerprintExportBatchesCounter counts the batches sent, by entrypoint and code.
	FingerprintExportBatchesCounter() metrics.Counter
	// FingerprintExportQueueRecordsGauge is the count of records waiting to be sent, by entrypoint.
	FingerprintExportQueueRecordsGauge() metrics.Gauge
	// FingerprintExportSendHistogram is the duration of the batch sending, by entrypoint.
	FingerprintExportSendHistogram() ScalableHistogram
	// FingerprintCaptureTruncatedCounter counts the captures cut at their size limit, by entrypoint and part.
	FingerprintCaptureTruncatedCounter() metrics.Counter
	// ForwardAuthFailOpenCounter counts the requests let through after an authentication server failure,
	// by middleware and reason.
	ForwardAuthFailOpenCounter() metrics.Counter
}

type traefixMetrics struct {
	fingerprintExportRecords      metrics.Counter
	fingerprintExportDropped      metrics.Counter
	fingerprintExportBatches      metrics.Counter
	fingerprintExportQueueRecords metrics.Gauge
	fingerprintExportSend         ScalableHistogram
	fingerprintCaptureTruncated   metrics.Counter
	forwardAuthFailOpen           metrics.Counter
}

func (r *standardRegistry) FingerprintExportRecordsCounter() metrics.Counter {
	return r.traefix.fingerprintExportRecords
}

func (r *standardRegistry) FingerprintExportDroppedCounter() metrics.Counter {
	return r.traefix.fingerprintExportDropped
}

func (r *standardRegistry) FingerprintExportBatchesCounter() metrics.Counter {
	return r.traefix.fingerprintExportBatches
}

func (r *standardRegistry) FingerprintExportQueueRecordsGauge() metrics.Gauge {
	return r.traefix.fingerprintExportQueueRecords
}

func (r *standardRegistry) FingerprintExportSendHistogram() ScalableHistogram {
	return r.traefix.fingerprintExportSend
}

func (r *standardRegistry) FingerprintCaptureTruncatedCounter() metrics.Counter {
	return r.traefix.fingerprintCaptureTruncated
}

func (r *standardRegistry) ForwardAuthFailOpenCounter() metrics.Counter {
	return r.traefix.forwardAuthFailOpen
}

func newMultiTraefixMetrics(registries []Registry) traefixMetrics {
	var records, dropped, batches, truncated, failOpen []metrics.Counter
	var queue []metrics.Gauge
	var send []ScalableHistogram

	for _, r := range registries {
		if c := r.FingerprintExportRecordsCounter(); c != nil {
			records = append(records, c)
		}
		if c := r.FingerprintExportDroppedCounter(); c != nil {
			dropped = append(dropped, c)
		}
		if c := r.FingerprintExportBatchesCounter(); c != nil {
			batches = append(batches, c)
		}
		if g := r.FingerprintExportQueueRecordsGauge(); g != nil {
			queue = append(queue, g)
		}
		if h := r.FingerprintExportSendHistogram(); h != nil {
			send = append(send, h)
		}
		if c := r.FingerprintCaptureTruncatedCounter(); c != nil {
			truncated = append(truncated, c)
		}
		if c := r.ForwardAuthFailOpenCounter(); c != nil {
			failOpen = append(failOpen, c)
		}
	}

	return traefixMetrics{
		fingerprintExportRecords:      multi.NewCounter(records...),
		fingerprintExportDropped:      multi.NewCounter(dropped...),
		fingerprintExportBatches:      multi.NewCounter(batches...),
		fingerprintExportQueueRecords: multi.NewGauge(queue...),
		fingerprintExportSend:         MultiHistogram(send),
		fingerprintCaptureTruncated:   multi.NewCounter(truncated...),
		forwardAuthFailOpen:           multi.NewCounter(failOpen...),
	}
}

func newPrometheusTraefixMetrics(buckets []float64) (traefixMetrics, []vector) {
	records := newCounterFrom(stdprometheus.CounterOpts{
		Name: fingerprintExportRecordsName,
		Help: "How many connection fingerprint records were queued for export, partitioned by entrypoint and type.",
	}, []string{labelEntryPoint, "type"})
	dropped := newCounterFrom(stdprometheus.CounterOpts{
		Name: fingerprintExportDroppedName,
		Help: "How many connection fingerprint records were dropped, partitioned by entrypoint and reason.",
	}, []string{labelEntryPoint, "reason"})
	batches := newCounterFrom(stdprometheus.CounterOpts{
		Name: fingerprintExportBatchesName,
		Help: "How many batches of connection fingerprint records were sent, partitioned by entrypoint and response code.",
	}, []string{labelEntryPoint, labelCode})
	queue := newGaugeFrom(stdprometheus.GaugeOpts{
		Name: fingerprintExportQueueRecordsName,
		Help: "How many connection fingerprint records are waiting to be sent, partitioned by entrypoint.",
	}, []string{labelEntryPoint})
	send := newHistogramFrom(stdprometheus.HistogramOpts{
		Name:    fingerprintExportSendName,
		Help:    "How long it took to send a batch of connection fingerprint records, partitioned by entrypoint.",
		Buckets: buckets,
	}, []string{labelEntryPoint})
	truncated := newCounterFrom(stdprometheus.CounterOpts{
		Name: fingerprintCaptureTruncatedName,
		Help: "How many connection captures were cut at their size limit, partitioned by entrypoint and part.",
	}, []string{labelEntryPoint, "part"})
	failOpen := newCounterFrom(stdprometheus.CounterOpts{
		Name: forwardAuthFailOpenName,
		Help: "How many requests a forwardAuth middleware let through after an authentication server failure, partitioned by middleware and reason.",
	}, []string{"middleware", "reason"})

	m := traefixMetrics{
		fingerprintExportRecords:      records,
		fingerprintExportDropped:      dropped,
		fingerprintExportBatches:      batches,
		fingerprintExportQueueRecords: queue,
		fingerprintCaptureTruncated:   truncated,
		forwardAuthFailOpen:           failOpen,
	}
	m.fingerprintExportSend, _ = NewHistogramWithScale(send, time.Second)

	return m, []vector{records.cv, dropped.cv, batches.cv, queue.gv, send.hv, truncated.cv, failOpen.cv}
}

func newDatadogTraefixMetrics(client *dogstatsd.Dogstatsd) traefixMetrics {
	m := traefixMetrics{
		fingerprintExportRecords:      client.NewCounter(statsdFingerprintExportRecordsName, 1.0),
		fingerprintExportDropped:      client.NewCounter(statsdFingerprintExportDroppedName, 1.0),
		fingerprintExportBatches:      client.NewCounter(statsdFingerprintExportBatchesName, 1.0),
		fingerprintExportQueueRecords: client.NewGauge(statsdFingerprintExportQueueRecordsName),
		fingerprintCaptureTruncated:   client.NewCounter(statsdFingerprintCaptureTruncatedName, 1.0),
		forwardAuthFailOpen:           client.NewCounter(statsdForwardAuthFailOpenName, 1.0),
	}
	m.fingerprintExportSend, _ = NewHistogramWithScale(client.NewHistogram(statsdFingerprintExportSendName, 1.0), time.Second)

	return m
}

func newStatsdTraefixMetrics(client *statsd.Statsd) traefixMetrics {
	m := traefixMetrics{
		fingerprintExportRecords:      client.NewCounter(statsdFingerprintExportRecordsName, 1.0),
		fingerprintExportDropped:      client.NewCounter(statsdFingerprintExportDroppedName, 1.0),
		fingerprintExportBatches:      client.NewCounter(statsdFingerprintExportBatchesName, 1.0),
		fingerprintExportQueueRecords: client.NewGauge(statsdFingerprintExportQueueRecordsName),
		fingerprintCaptureTruncated:   client.NewCounter(statsdFingerprintCaptureTruncatedName, 1.0),
		forwardAuthFailOpen:           client.NewCounter(statsdForwardAuthFailOpenName, 1.0),
	}
	m.fingerprintExportSend, _ = NewHistogramWithScale(client.NewTiming(statsdFingerprintExportSendName, 1.0), time.Millisecond)

	return m
}

func newInfluxDB2TraefixMetrics(store *influx.Influx) traefixMetrics {
	m := traefixMetrics{
		fingerprintExportRecords:      store.NewCounter(influxDBFingerprintPrefix + statsdFingerprintExportRecordsName),
		fingerprintExportDropped:      store.NewCounter(influxDBFingerprintPrefix + statsdFingerprintExportDroppedName),
		fingerprintExportBatches:      store.NewCounter(influxDBFingerprintPrefix + statsdFingerprintExportBatchesName),
		fingerprintExportQueueRecords: store.NewGauge(influxDBFingerprintPrefix + statsdFingerprintExportQueueRecordsName),
		fingerprintCaptureTruncated:   store.NewCounter(influxDBFingerprintPrefix + statsdFingerprintCaptureTruncatedName),
		forwardAuthFailOpen:           store.NewCounter(influxDBFingerprintPrefix + statsdForwardAuthFailOpenName),
	}
	m.fingerprintExportSend, _ = NewHistogramWithScale(store.NewHistogram(influxDBFingerprintPrefix+statsdFingerprintExportSendName), time.Second)

	return m
}

func newOTLPTraefixMetrics(meter metric.Meter) traefixMetrics {
	m := traefixMetrics{
		fingerprintExportRecords: newOTLPCounterFrom(meter, fingerprintExportRecordsName,
			"How many connection fingerprint records were queued for export, partitioned by entrypoint and type."),
		fingerprintExportDropped: newOTLPCounterFrom(meter, fingerprintExportDroppedName,
			"How many connection fingerprint records were dropped, partitioned by entrypoint and reason."),
		fingerprintExportBatches: newOTLPCounterFrom(meter, fingerprintExportBatchesName,
			"How many batches of connection fingerprint records were sent, partitioned by entrypoint and response code."),
		fingerprintExportQueueRecords: newOTLPGaugeFrom(meter, fingerprintExportQueueRecordsName,
			"How many connection fingerprint records are waiting to be sent, partitioned by entrypoint.", "1"),
		fingerprintCaptureTruncated: newOTLPCounterFrom(meter, fingerprintCaptureTruncatedName,
			"How many connection captures were cut at their size limit, partitioned by entrypoint and part."),
		forwardAuthFailOpen: newOTLPCounterFrom(meter, forwardAuthFailOpenName,
			"How many requests a forwardAuth middleware let through after an authentication server failure, partitioned by middleware and reason."),
	}
	m.fingerprintExportSend, _ = NewHistogramWithScale(newOTLPHistogramFrom(meter, fingerprintExportSendName,
		"How long it took to send a batch of connection fingerprint records, partitioned by entrypoint.", otlpSecondsUnit), time.Second)

	return m
}
