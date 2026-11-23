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
	wafRequestsName                   = MetricNamePrefix + "waf_requests_total"
	wafDurationName                   = MetricNamePrefix + "waf_duration_seconds"
	wafRuleMatchesName                = MetricNamePrefix + "waf_rule_matches_total"

	// The StatsD and Datadog clients add the "traefik." prefix themselves.
	statsdFingerprintExportRecordsName      = "fingerprint.export.records.total"
	statsdFingerprintExportDroppedName      = "fingerprint.export.dropped.total"
	statsdFingerprintExportBatchesName      = "fingerprint.export.batches.total"
	statsdFingerprintExportQueueRecordsName = "fingerprint.export.queue.records"
	statsdFingerprintExportSendName         = "fingerprint.export.send.duration"
	statsdFingerprintCaptureTruncatedName   = "fingerprint.capture.truncated.total"
	statsdForwardAuthFailOpenName           = "forwardauth.failopen.total"
	statsdWAFRequestsName                   = "waf.requests.total"
	statsdWAFDurationName                   = "waf.duration"
	statsdWAFRuleMatchesName                = "waf.rule.matches.total"

	influxDBFingerprintPrefix = "traefik."

	labelEntryPoint = "entrypoint"
	labelCode       = "code"
	labelMiddleware = "middleware"
)

// otlpSecondsUnit is a variable, not a constant, so that the upstream newOTLPHistogramFrom keeps receiving a unit
// unparam does not see as always the same.
var otlpSecondsUnit = "s"

// traefixRegistry is the part of Registry holding the Traefix metrics.
//
//nolint:interfacebloat // It gathers the metrics of every Traefix feature behind the single line the upstream Registry embeds.
type traefixRegistry interface {
	clusterRegistry
	meshRegistry
	meshSatelliteRegistry

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
	// WAFRequestsCounter counts the requests the Coraza middleware processed, by middleware and result.
	WAFRequestsCounter() metrics.Counter
	// WAFDurationHistogram is the processing time of the Coraza middleware, without the next handlers, by middleware.
	WAFDurationHistogram() ScalableHistogram
	// WAFRuleMatchesCounter counts the rules with a message that matched, by middleware and rule_id.
	WAFRuleMatchesCounter() metrics.Counter
}

type traefixMetrics struct {
	fingerprintExportRecords      metrics.Counter
	fingerprintExportDropped      metrics.Counter
	fingerprintExportBatches      metrics.Counter
	fingerprintExportQueueRecords metrics.Gauge
	fingerprintExportSend         ScalableHistogram
	fingerprintCaptureTruncated   metrics.Counter
	forwardAuthFailOpen           metrics.Counter
	wafRequests                   metrics.Counter
	wafDuration                   ScalableHistogram
	wafRuleMatches                metrics.Counter

	cluster clusterMetrics
	mesh    meshMetrics
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

func (r *standardRegistry) WAFRequestsCounter() metrics.Counter {
	return r.traefix.wafRequests
}

func (r *standardRegistry) WAFDurationHistogram() ScalableHistogram {
	return r.traefix.wafDuration
}

func (r *standardRegistry) WAFRuleMatchesCounter() metrics.Counter {
	return r.traefix.wafRuleMatches
}

func newMultiTraefixMetrics(registries []Registry) traefixMetrics {
	var records, dropped, batches, truncated, failOpen, wafRequests, wafRuleMatches []metrics.Counter
	var queue []metrics.Gauge
	var send, wafDuration []ScalableHistogram

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
		if c := r.WAFRequestsCounter(); c != nil {
			wafRequests = append(wafRequests, c)
		}
		if h := r.WAFDurationHistogram(); h != nil {
			wafDuration = append(wafDuration, h)
		}
		if c := r.WAFRuleMatchesCounter(); c != nil {
			wafRuleMatches = append(wafRuleMatches, c)
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
		wafRequests:                   multi.NewCounter(wafRequests...),
		wafDuration:                   MultiHistogram(wafDuration),
		wafRuleMatches:                multi.NewCounter(wafRuleMatches...),
		cluster:                       newMultiClusterMetrics(registries),
		mesh:                          newMultiMeshMetrics(registries),
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
	}, []string{labelMiddleware, "reason"})
	wafRequests := newCounterFrom(stdprometheus.CounterOpts{
		Name: wafRequestsName,
		Help: "How many requests a Coraza middleware processed, partitioned by middleware and result.",
	}, []string{labelMiddleware, "result"})
	wafDuration := newHistogramFrom(stdprometheus.HistogramOpts{
		Name:    wafDurationName,
		Help:    "How long a Coraza middleware took to process a request, without the next handlers, partitioned by middleware.",
		Buckets: []float64{0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
	}, []string{labelMiddleware})
	wafRuleMatches := newCounterFrom(stdprometheus.CounterOpts{
		Name: wafRuleMatchesName,
		Help: "How many times the rules with a message of a Coraza middleware matched, partitioned by middleware and rule identifier.",
	}, []string{labelMiddleware, "rule_id"})

	m := traefixMetrics{
		fingerprintExportRecords:      records,
		fingerprintExportDropped:      dropped,
		fingerprintExportBatches:      batches,
		fingerprintExportQueueRecords: queue,
		fingerprintCaptureTruncated:   truncated,
		forwardAuthFailOpen:           failOpen,
		wafRequests:                   wafRequests,
		wafRuleMatches:                wafRuleMatches,
	}
	m.fingerprintExportSend, _ = NewHistogramWithScale(send, time.Second)
	m.wafDuration, _ = NewHistogramWithScale(wafDuration, time.Second)

	vectors := []vector{records.cv, dropped.cv, batches.cv, queue.gv, send.hv, truncated.cv, failOpen.cv, wafRequests.cv, wafDuration.hv, wafRuleMatches.cv}

	var clusterVectors, meshVectors []vector
	m.cluster, clusterVectors = newPrometheusClusterMetrics()
	m.mesh, meshVectors = newPrometheusMeshMetrics()

	return m, append(append(vectors, clusterVectors...), meshVectors...)
}

func newDatadogTraefixMetrics(client *dogstatsd.Dogstatsd) traefixMetrics {
	m := traefixMetrics{
		fingerprintExportRecords:      client.NewCounter(statsdFingerprintExportRecordsName, 1.0),
		fingerprintExportDropped:      client.NewCounter(statsdFingerprintExportDroppedName, 1.0),
		fingerprintExportBatches:      client.NewCounter(statsdFingerprintExportBatchesName, 1.0),
		fingerprintExportQueueRecords: client.NewGauge(statsdFingerprintExportQueueRecordsName),
		fingerprintCaptureTruncated:   client.NewCounter(statsdFingerprintCaptureTruncatedName, 1.0),
		forwardAuthFailOpen:           client.NewCounter(statsdForwardAuthFailOpenName, 1.0),
		wafRequests:                   client.NewCounter(statsdWAFRequestsName, 1.0),
		wafRuleMatches:                client.NewCounter(statsdWAFRuleMatchesName, 1.0),
	}
	m.fingerprintExportSend, _ = NewHistogramWithScale(client.NewHistogram(statsdFingerprintExportSendName, 1.0), time.Second)
	m.wafDuration, _ = NewHistogramWithScale(client.NewHistogram(statsdWAFDurationName, 1.0), time.Second)
	m.cluster = newDatadogClusterMetrics(client)
	m.mesh = newDatadogMeshMetrics(client)

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
		wafRequests:                   client.NewCounter(statsdWAFRequestsName, 1.0),
		wafRuleMatches:                client.NewCounter(statsdWAFRuleMatchesName, 1.0),
	}
	m.fingerprintExportSend, _ = NewHistogramWithScale(client.NewTiming(statsdFingerprintExportSendName, 1.0), time.Millisecond)
	m.wafDuration, _ = NewHistogramWithScale(client.NewTiming(statsdWAFDurationName, 1.0), time.Millisecond)
	m.cluster = newStatsdClusterMetrics(client)
	m.mesh = newStatsdMeshMetrics(client)

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
		wafRequests:                   store.NewCounter(influxDBFingerprintPrefix + statsdWAFRequestsName),
		wafRuleMatches:                store.NewCounter(influxDBFingerprintPrefix + statsdWAFRuleMatchesName),
	}
	m.fingerprintExportSend, _ = NewHistogramWithScale(store.NewHistogram(influxDBFingerprintPrefix+statsdFingerprintExportSendName), time.Second)
	m.wafDuration, _ = NewHistogramWithScale(store.NewHistogram(influxDBFingerprintPrefix+statsdWAFDurationName), time.Second)
	m.cluster = newInfluxDB2ClusterMetrics(store)
	m.mesh = newInfluxDB2MeshMetrics(store)

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
		wafRequests: newOTLPCounterFrom(meter, wafRequestsName,
			"How many requests a Coraza middleware processed, partitioned by middleware and result."),
		wafRuleMatches: newOTLPCounterFrom(meter, wafRuleMatchesName,
			"How many times the rules with a message of a Coraza middleware matched, partitioned by middleware and rule identifier."),
	}
	m.fingerprintExportSend, _ = NewHistogramWithScale(newOTLPHistogramFrom(meter, fingerprintExportSendName,
		"How long it took to send a batch of connection fingerprint records, partitioned by entrypoint.", otlpSecondsUnit), time.Second)
	m.wafDuration, _ = NewHistogramWithScale(newOTLPHistogramFrom(meter, wafDurationName,
		"How long a Coraza middleware took to process a request, without the next handlers, partitioned by middleware.", otlpSecondsUnit), time.Second)
	m.cluster = newOTLPClusterMetrics(meter)
	m.mesh = newOTLPMeshMetrics(meter)

	return m
}
