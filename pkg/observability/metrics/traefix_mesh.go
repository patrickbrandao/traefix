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
	meshPeersName   = MetricNamePrefix + "mesh_peers"
	meshLinkUpName  = MetricNamePrefix + "mesh_link_up"
	meshLinkRTTName = MetricNamePrefix + "mesh_link_rtt_seconds"

	// The StatsD and Datadog clients add the "traefik." prefix themselves.
	statsdMeshPeersName   = "mesh.peers"
	statsdMeshLinkUpName  = "mesh.link.up"
	statsdMeshLinkRTTName = "mesh.link.rtt"

	meshPeersHelp   = "How many instances of the mesh this instance opens a link to, partitioned by the state of the link."
	meshLinkUpHelp  = "Whether the link this instance opens to another instance of the mesh is up, partitioned by peer and transport."
	meshLinkRTTHelp = "How long the other instance took to answer the last ping on a link of the mesh, partitioned by peer."

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
}

type meshMetrics struct {
	peers   metrics.Gauge
	linkUp  metrics.Gauge
	linkRTT metrics.Gauge
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

	for _, r := range registries {
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
		peers:   multi.NewGauge(peers...),
		linkUp:  multi.NewGauge(linkUp...),
		linkRTT: multi.NewGauge(linkRTT...),
	}
}

func newPrometheusMeshMetrics() (meshMetrics, []vector) {
	peers := newGaugeFrom(stdprometheus.GaugeOpts{Name: meshPeersName, Help: meshPeersHelp}, []string{"state"})
	linkUp := newGaugeFrom(stdprometheus.GaugeOpts{Name: meshLinkUpName, Help: meshLinkUpHelp}, []string{labelPeer, "transport"})
	linkRTT := newGaugeFrom(stdprometheus.GaugeOpts{Name: meshLinkRTTName, Help: meshLinkRTTHelp}, []string{labelPeer})

	return meshMetrics{peers: peers, linkUp: linkUp, linkRTT: linkRTT}, []vector{peers.gv, linkUp.gv, linkRTT.gv}
}

func newDatadogMeshMetrics(client *dogstatsd.Dogstatsd) meshMetrics {
	return meshMetrics{
		peers:   client.NewGauge(statsdMeshPeersName),
		linkUp:  client.NewGauge(statsdMeshLinkUpName),
		linkRTT: client.NewGauge(statsdMeshLinkRTTName),
	}
}

func newStatsdMeshMetrics(client *statsd.Statsd) meshMetrics {
	return meshMetrics{
		peers:   client.NewGauge(statsdMeshPeersName),
		linkUp:  client.NewGauge(statsdMeshLinkUpName),
		linkRTT: client.NewGauge(statsdMeshLinkRTTName),
	}
}

func newInfluxDB2MeshMetrics(store *influx.Influx) meshMetrics {
	return meshMetrics{
		peers:   store.NewGauge(influxDBFingerprintPrefix + statsdMeshPeersName),
		linkUp:  store.NewGauge(influxDBFingerprintPrefix + statsdMeshLinkUpName),
		linkRTT: store.NewGauge(influxDBFingerprintPrefix + statsdMeshLinkRTTName),
	}
}

func newOTLPMeshMetrics(meter metric.Meter) meshMetrics {
	return meshMetrics{
		peers:   newOTLPGaugeFrom(meter, meshPeersName, meshPeersHelp, "1"),
		linkUp:  newOTLPGaugeFrom(meter, meshLinkUpName, meshLinkUpHelp, "1"),
		linkRTT: newOTLPGaugeFrom(meter, meshLinkRTTName, meshLinkRTTHelp, "s"),
	}
}
