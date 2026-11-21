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
	meshPeersName    = MetricNamePrefix + "mesh_peers"
	meshLinkUpName   = MetricNamePrefix + "mesh_link_up"
	meshLinkRTTName  = MetricNamePrefix + "mesh_link_rtt_seconds"
	meshRequestsName = MetricNamePrefix + "mesh_requests_total"
	meshReceivedName = MetricNamePrefix + "mesh_received_total"

	// The StatsD and Datadog clients add the "traefik." prefix themselves.
	statsdMeshPeersName    = "mesh.peers"
	statsdMeshLinkUpName   = "mesh.link.up"
	statsdMeshLinkRTTName  = "mesh.link.rtt"
	statsdMeshRequestsName = "mesh.requests.total"
	statsdMeshReceivedName = "mesh.received.total"

	meshPeersHelp    = "How many instances of the mesh this instance opens a link to, partitioned by the state of the link."
	meshLinkUpHelp   = "Whether the link this instance opens to another instance of the mesh is up, partitioned by peer and transport."
	meshLinkRTTHelp  = "How long the other instance took to answer the last ping on a link of the mesh, partitioned by peer."
	meshRequestsHelp = "How many requests the instance forwarded to another instance of the mesh, partitioned by peer, kind and result."
	meshReceivedHelp = "How many requests the instance received from another instance of the mesh, partitioned by peer, kind and result."

	labelPeer = "peer"
)

// meshRegistry is the part of Registry holding the metrics of the Traefix mesh.
type meshRegistry interface {
	// MeshPeersGauge is the count of instances this instance opens a link to, by state.
	MeshPeersGauge() metrics.Gauge
	// MeshLinkUpGauge is 1 for the transport a link is up with, by peer and transport.
	MeshLinkUpGauge() metrics.Gauge
	// MeshLinkRTTGauge is the time the last ping took on a link, in seconds, by peer.
	MeshLinkRTTGauge() metrics.Gauge
	// MeshRequestsCounter counts the requests forwarded to another instance, by peer, kind and result.
	MeshRequestsCounter() metrics.Counter
	// MeshReceivedCounter counts the requests received from another instance, by peer, kind and result.
	MeshReceivedCounter() metrics.Counter
}

type meshMetrics struct {
	peers    metrics.Gauge
	linkUp   metrics.Gauge
	linkRTT  metrics.Gauge
	requests metrics.Counter
	received metrics.Counter
}

func (r *standardRegistry) MeshRequestsCounter() metrics.Counter {
	return r.traefix.mesh.requests
}

func (r *standardRegistry) MeshReceivedCounter() metrics.Counter {
	return r.traefix.mesh.received
}

func (r *standardRegistry) MeshPeersGauge() metrics.Gauge {
	return r.traefix.mesh.peers
}

func (r *standardRegistry) MeshLinkUpGauge() metrics.Gauge {
	return r.traefix.mesh.linkUp
}

func (r *standardRegistry) MeshLinkRTTGauge() metrics.Gauge {
	return r.traefix.mesh.linkRTT
}

func newMultiMeshMetrics(registries []Registry) meshMetrics {
	var peers, linkUp, linkRTT []metrics.Gauge
	var requests, received []metrics.Counter

	for _, r := range registries {
		if c := r.MeshRequestsCounter(); c != nil {
			requests = append(requests, c)
		}
		if c := r.MeshReceivedCounter(); c != nil {
			received = append(received, c)
		}
		if g := r.MeshPeersGauge(); g != nil {
			peers = append(peers, g)
		}
		if g := r.MeshLinkUpGauge(); g != nil {
			linkUp = append(linkUp, g)
		}
		if g := r.MeshLinkRTTGauge(); g != nil {
			linkRTT = append(linkRTT, g)
		}
	}

	return meshMetrics{
		peers:    multi.NewGauge(peers...),
		linkUp:   multi.NewGauge(linkUp...),
		linkRTT:  multi.NewGauge(linkRTT...),
		requests: multi.NewCounter(requests...),
		received: multi.NewCounter(received...),
	}
}

func newPrometheusMeshMetrics() (meshMetrics, []vector) {
	peers := newGaugeFrom(stdprometheus.GaugeOpts{Name: meshPeersName, Help: meshPeersHelp}, []string{"state"})
	linkUp := newGaugeFrom(stdprometheus.GaugeOpts{Name: meshLinkUpName, Help: meshLinkUpHelp}, []string{labelPeer, "transport"})
	linkRTT := newGaugeFrom(stdprometheus.GaugeOpts{Name: meshLinkRTTName, Help: meshLinkRTTHelp}, []string{labelPeer})
	requests := newCounterFrom(stdprometheus.CounterOpts{Name: meshRequestsName, Help: meshRequestsHelp}, []string{labelPeer, "kind", labelResult})
	received := newCounterFrom(stdprometheus.CounterOpts{Name: meshReceivedName, Help: meshReceivedHelp}, []string{labelPeer, "kind", labelResult})

	m := meshMetrics{peers: peers, linkUp: linkUp, linkRTT: linkRTT, requests: requests, received: received}

	return m, []vector{peers.gv, linkUp.gv, linkRTT.gv, requests.cv, received.cv}
}

func newDatadogMeshMetrics(client *dogstatsd.Dogstatsd) meshMetrics {
	return meshMetrics{
		peers:   client.NewGauge(statsdMeshPeersName),
		linkUp:  client.NewGauge(statsdMeshLinkUpName),
		linkRTT: client.NewGauge(statsdMeshLinkRTTName),

		requests: client.NewCounter(statsdMeshRequestsName, 1.0),
		received: client.NewCounter(statsdMeshReceivedName, 1.0),
	}
}

func newStatsdMeshMetrics(client *statsd.Statsd) meshMetrics {
	return meshMetrics{
		peers:   client.NewGauge(statsdMeshPeersName),
		linkUp:  client.NewGauge(statsdMeshLinkUpName),
		linkRTT: client.NewGauge(statsdMeshLinkRTTName),

		requests: client.NewCounter(statsdMeshRequestsName, 1.0),
		received: client.NewCounter(statsdMeshReceivedName, 1.0),
	}
}

func newInfluxDB2MeshMetrics(store *influx.Influx) meshMetrics {
	return meshMetrics{
		peers:   store.NewGauge(influxDBFingerprintPrefix + statsdMeshPeersName),
		linkUp:  store.NewGauge(influxDBFingerprintPrefix + statsdMeshLinkUpName),
		linkRTT: store.NewGauge(influxDBFingerprintPrefix + statsdMeshLinkRTTName),

		requests: store.NewCounter(influxDBFingerprintPrefix + statsdMeshRequestsName),
		received: store.NewCounter(influxDBFingerprintPrefix + statsdMeshReceivedName),
	}
}

func newOTLPMeshMetrics(meter metric.Meter) meshMetrics {
	return meshMetrics{
		peers:   newOTLPGaugeFrom(meter, meshPeersName, meshPeersHelp, "1"),
		linkUp:  newOTLPGaugeFrom(meter, meshLinkUpName, meshLinkUpHelp, "1"),
		linkRTT: newOTLPGaugeFrom(meter, meshLinkRTTName, meshLinkRTTHelp, "s"),

		requests: newOTLPCounterFrom(meter, meshRequestsName, meshRequestsHelp),
		received: newOTLPCounterFrom(meter, meshReceivedName, meshReceivedHelp),
	}
}
