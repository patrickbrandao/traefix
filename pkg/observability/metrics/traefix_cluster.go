package metrics

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/dogstatsd"
	"github.com/go-kit/kit/metrics/influx"
	"github.com/go-kit/kit/metrics/multi"
	"github.com/go-kit/kit/metrics/statsd"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/metric"
)

const (
	clusterRedisUpName              = MetricNamePrefix + "cluster_redis_up"
	clusterNodesName                = MetricNamePrefix + "cluster_nodes"
	acmeClusterElectionsName        = MetricNamePrefix + "acme_cluster_elections_total"
	acmeClusterIssuancesName        = MetricNamePrefix + "acme_cluster_issuances_total"
	acmeClusterChallengeLookupsName = MetricNamePrefix + "acme_cluster_challenge_lookups_total"
	acmeClusterConfigMismatchName   = MetricNamePrefix + "acme_cluster_config_mismatch"
	acmeStoreWritesName             = MetricNamePrefix + "acme_store_writes_total"

	// The StatsD and Datadog clients add the "traefik." prefix themselves.
	statsdClusterRedisUpName              = "cluster.redis.up"
	statsdClusterNodesName                = "cluster.nodes"
	statsdACMEClusterElectionsName        = "acme.cluster.elections.total"
	statsdACMEClusterIssuancesName        = "acme.cluster.issuances.total"
	statsdACMEClusterChallengeLookupsName = "acme.cluster.challenge.lookups.total"
	statsdACMEClusterConfigMismatchName   = "acme.cluster.config.mismatch"
	statsdACMEStoreWritesName             = "acme.store.writes.total"

	clusterRedisUpHelp              = "Whether the last announcement of the instance reached the Redis server of the cluster."
	clusterNodesHelp                = "How many instances of the cluster are running, as seen by this instance."
	acmeClusterElectionsHelp        = "How many times the instance took part in the election of a certificate request, partitioned by resolver and result."
	acmeClusterIssuancesHelp        = "How many certificates the instance requested from the ACME server for the cluster, partitioned by resolver, kind and result."
	acmeClusterChallengeLookupsHelp = "How many HTTP-01 challenge requests the instance answered, partitioned by result."
	acmeClusterConfigMismatchHelp   = "Whether another instance of the cluster configures the certificate resolver differently, partitioned by resolver."
	acmeStoreWritesHelp             = "How many times the instance wrote certificates to a certificate store, partitioned by resolver, target and result."

	labelResolver = "resolver"
	labelResult   = "result"
)

// clusterRegistry is the part of Registry holding the metrics of the Traefix cluster.
type clusterRegistry interface {
	// ClusterRedisUpGauge is 1 when the last announcement of the instance reached the Redis server of the cluster.
	ClusterRedisUpGauge() metrics.Gauge
	// ClusterNodesGauge is the count of running instances of the cluster.
	ClusterNodesGauge() metrics.Gauge
	// ACMEClusterElectionsCounter counts the elections of a certificate request, by resolver and result.
	ACMEClusterElectionsCounter() metrics.Counter
	// ACMEClusterIssuancesCounter counts the certificates requested for the cluster, by resolver, kind and result.
	ACMEClusterIssuancesCounter() metrics.Counter
	// ACMEClusterChallengeLookupsCounter counts the HTTP-01 challenge requests, by result.
	ACMEClusterChallengeLookupsCounter() metrics.Counter
	// ACMEClusterConfigMismatchGauge is 1 when another instance configures the resolver differently, by resolver.
	ACMEClusterConfigMismatchGauge() metrics.Gauge
	// ACMEStoreWritesCounter counts the writes to a certificate store, by resolver, target and result.
	ACMEStoreWritesCounter() metrics.Counter
}

type clusterMetrics struct {
	redisUp          metrics.Gauge
	nodes            metrics.Gauge
	elections        metrics.Counter
	issuances        metrics.Counter
	challengeLookups metrics.Counter
	configMismatch   metrics.Gauge
	storeWrites      metrics.Counter
}

func (r *standardRegistry) ClusterRedisUpGauge() metrics.Gauge {
	return r.traefix.cluster.redisUp
}

func (r *standardRegistry) ClusterNodesGauge() metrics.Gauge {
	return r.traefix.cluster.nodes
}

func (r *standardRegistry) ACMEClusterElectionsCounter() metrics.Counter {
	return r.traefix.cluster.elections
}

func (r *standardRegistry) ACMEClusterIssuancesCounter() metrics.Counter {
	return r.traefix.cluster.issuances
}

func (r *standardRegistry) ACMEClusterChallengeLookupsCounter() metrics.Counter {
	return r.traefix.cluster.challengeLookups
}

func (r *standardRegistry) ACMEClusterConfigMismatchGauge() metrics.Gauge {
	return r.traefix.cluster.configMismatch
}

func (r *standardRegistry) ACMEStoreWritesCounter() metrics.Counter {
	return r.traefix.cluster.storeWrites
}

func newMultiClusterMetrics(registries []Registry) clusterMetrics {
	var redisUp, nodes, configMismatch []metrics.Gauge
	var elections, issuances, challengeLookups, storeWrites []metrics.Counter

	for _, r := range registries {
		if g := r.ClusterRedisUpGauge(); g != nil {
			redisUp = append(redisUp, g)
		}
		if g := r.ClusterNodesGauge(); g != nil {
			nodes = append(nodes, g)
		}
		if c := r.ACMEClusterElectionsCounter(); c != nil {
			elections = append(elections, c)
		}
		if c := r.ACMEClusterIssuancesCounter(); c != nil {
			issuances = append(issuances, c)
		}
		if c := r.ACMEClusterChallengeLookupsCounter(); c != nil {
			challengeLookups = append(challengeLookups, c)
		}
		if g := r.ACMEClusterConfigMismatchGauge(); g != nil {
			configMismatch = append(configMismatch, g)
		}
		if c := r.ACMEStoreWritesCounter(); c != nil {
			storeWrites = append(storeWrites, c)
		}
	}

	return clusterMetrics{
		redisUp:          multi.NewGauge(redisUp...),
		nodes:            multi.NewGauge(nodes...),
		elections:        multi.NewCounter(elections...),
		issuances:        multi.NewCounter(issuances...),
		challengeLookups: multi.NewCounter(challengeLookups...),
		configMismatch:   multi.NewGauge(configMismatch...),
		storeWrites:      multi.NewCounter(storeWrites...),
	}
}

func newPrometheusClusterMetrics() (clusterMetrics, []vector) {
	redisUp := newGaugeFrom(stdprometheus.GaugeOpts{Name: clusterRedisUpName, Help: clusterRedisUpHelp}, []string{})
	nodes := newGaugeFrom(stdprometheus.GaugeOpts{Name: clusterNodesName, Help: clusterNodesHelp}, []string{})
	elections := newCounterFrom(stdprometheus.CounterOpts{Name: acmeClusterElectionsName, Help: acmeClusterElectionsHelp},
		[]string{labelResolver, labelResult})
	issuances := newCounterFrom(stdprometheus.CounterOpts{Name: acmeClusterIssuancesName, Help: acmeClusterIssuancesHelp},
		[]string{labelResolver, labelKind, labelResult})
	challengeLookups := newCounterFrom(stdprometheus.CounterOpts{Name: acmeClusterChallengeLookupsName, Help: acmeClusterChallengeLookupsHelp},
		[]string{labelResult})
	configMismatch := newGaugeFrom(stdprometheus.GaugeOpts{Name: acmeClusterConfigMismatchName, Help: acmeClusterConfigMismatchHelp},
		[]string{labelResolver})
	storeWrites := newCounterFrom(stdprometheus.CounterOpts{Name: acmeStoreWritesName, Help: acmeStoreWritesHelp},
		[]string{labelResolver, "target", labelResult})

	m := clusterMetrics{
		redisUp:          redisUp,
		nodes:            nodes,
		elections:        elections,
		issuances:        issuances,
		challengeLookups: challengeLookups,
		configMismatch:   configMismatch,
		storeWrites:      storeWrites,
	}

	return m, []vector{redisUp.gv, nodes.gv, elections.cv, issuances.cv, challengeLookups.cv, configMismatch.gv, storeWrites.cv}
}

func newDatadogClusterMetrics(client *dogstatsd.Dogstatsd) clusterMetrics {
	return clusterMetrics{
		redisUp:          client.NewGauge(statsdClusterRedisUpName),
		nodes:            client.NewGauge(statsdClusterNodesName),
		elections:        client.NewCounter(statsdACMEClusterElectionsName, 1.0),
		issuances:        client.NewCounter(statsdACMEClusterIssuancesName, 1.0),
		challengeLookups: client.NewCounter(statsdACMEClusterChallengeLookupsName, 1.0),
		configMismatch:   client.NewGauge(statsdACMEClusterConfigMismatchName),
		storeWrites:      client.NewCounter(statsdACMEStoreWritesName, 1.0),
	}
}

func newStatsdClusterMetrics(client *statsd.Statsd) clusterMetrics {
	return clusterMetrics{
		redisUp:          client.NewGauge(statsdClusterRedisUpName),
		nodes:            client.NewGauge(statsdClusterNodesName),
		elections:        client.NewCounter(statsdACMEClusterElectionsName, 1.0),
		issuances:        client.NewCounter(statsdACMEClusterIssuancesName, 1.0),
		challengeLookups: client.NewCounter(statsdACMEClusterChallengeLookupsName, 1.0),
		configMismatch:   client.NewGauge(statsdACMEClusterConfigMismatchName),
		storeWrites:      client.NewCounter(statsdACMEStoreWritesName, 1.0),
	}
}

func newInfluxDB2ClusterMetrics(store *influx.Influx) clusterMetrics {
	return clusterMetrics{
		redisUp:          store.NewGauge(influxDBFingerprintPrefix + statsdClusterRedisUpName),
		nodes:            store.NewGauge(influxDBFingerprintPrefix + statsdClusterNodesName),
		elections:        store.NewCounter(influxDBFingerprintPrefix + statsdACMEClusterElectionsName),
		issuances:        store.NewCounter(influxDBFingerprintPrefix + statsdACMEClusterIssuancesName),
		challengeLookups: store.NewCounter(influxDBFingerprintPrefix + statsdACMEClusterChallengeLookupsName),
		configMismatch:   store.NewGauge(influxDBFingerprintPrefix + statsdACMEClusterConfigMismatchName),
		storeWrites:      store.NewCounter(influxDBFingerprintPrefix + statsdACMEStoreWritesName),
	}
}

func newOTLPClusterMetrics(meter metric.Meter) clusterMetrics {
	return clusterMetrics{
		redisUp:          newOTLPGaugeFrom(meter, clusterRedisUpName, clusterRedisUpHelp, "1"),
		nodes:            newOTLPGaugeFrom(meter, clusterNodesName, clusterNodesHelp, "1"),
		elections:        newOTLPCounterFrom(meter, acmeClusterElectionsName, acmeClusterElectionsHelp),
		issuances:        newOTLPCounterFrom(meter, acmeClusterIssuancesName, acmeClusterIssuancesHelp),
		challengeLookups: newOTLPCounterFrom(meter, acmeClusterChallengeLookupsName, acmeClusterChallengeLookupsHelp),
		configMismatch:   newOTLPGaugeFrom(meter, acmeClusterConfigMismatchName, acmeClusterConfigMismatchHelp, "1"),
		storeWrites:      newOTLPCounterFrom(meter, acmeStoreWritesName, acmeStoreWritesHelp),
	}
}
