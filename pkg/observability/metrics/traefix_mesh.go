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
	meshPeersName      = MetricNamePrefix + "mesh_peers"
	meshLinkUpName     = MetricNamePrefix + "mesh_link_up"
	meshLinkRTTName    = MetricNamePrefix + "mesh_link_rtt_seconds"
	meshRequestsName   = MetricNamePrefix + "mesh_requests_total"
	meshReceivedName   = MetricNamePrefix + "mesh_received_total"
	meshRoutesName     = MetricNamePrefix + "mesh_routes"
	meshExportedName   = MetricNamePrefix + "mesh_exported_routes"
	meshPacketsName    = MetricNamePrefix + "mesh_udp_packets_total"
	meshMismatchName   = MetricNamePrefix + "mesh_config_mismatch"
	meshSatellitesName = MetricNamePrefix + "mesh_satellites"
	meshDeliveredName  = MetricNamePrefix + "mesh_certificates_delivered"

	// The StatsD and Datadog clients add the "traefik." prefix themselves.
	statsdMeshPeersName      = "mesh.peers"
	statsdMeshLinkUpName     = "mesh.link.up"
	statsdMeshLinkRTTName    = "mesh.link.rtt"
	statsdMeshRequestsName   = "mesh.requests.total"
	statsdMeshReceivedName   = "mesh.received.total"
	statsdMeshRoutesName     = "mesh.routes"
	statsdMeshExportedName   = "mesh.exported.routes"
	statsdMeshPacketsName    = "mesh.udp.packets.total"
	statsdMeshMismatchName   = "mesh.config.mismatch"
	statsdMeshSatellitesName = "mesh.satellites"
	statsdMeshDeliveredName  = "mesh.certificates.delivered"

	meshPeersHelp      = "How many instances of the mesh this instance opens a link to, partitioned by the state of the link."
	meshLinkUpHelp     = "Whether the link this instance opens to another instance of the mesh is up, partitioned by peer and transport."
	meshLinkRTTHelp    = "How long the other instance took to answer the last ping on a link of the mesh, partitioned by peer."
	meshRequestsHelp   = "How many requests the instance forwarded to another instance of the mesh, partitioned by peer, kind and result."
	meshReceivedHelp   = "How many requests the instance received from another instance of the mesh, partitioned by peer, kind and result."
	meshRoutesHelp     = "How many routers another instance of the mesh announces to this one, partitioned by peer and kind."
	meshExportedHelp   = "How many routers this instance announces to the other instances of the mesh, partitioned by kind."
	meshPacketsHelp    = "How many UDP packets the instance carried on the links of the mesh, partitioned by peer, path and direction."
	meshMismatchHelp   = "How many entry points another instance of the mesh has with the same name and another configuration, partitioned by peer."
	meshSatellitesHelp = "How many satellites of the mesh this instance reaches back, partitioned by the state of the link back."
	meshDeliveredHelp  = "How many certificates this instance gave a satellite of the mesh in its last answer, partitioned by peer."

	labelPeer = "peer"
	labelKind = "kind"
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
	// MeshRoutesGauge is the count of routers another instance announces, by peer and kind.
	MeshRoutesGauge() metrics.Gauge
	// MeshExportedRoutesGauge is the count of routers this instance announces, by kind.
	MeshExportedRoutesGauge() metrics.Gauge
	// MeshUDPPacketsCounter counts the UDP packets carried on the links, by peer, path and direction.
	MeshUDPPacketsCounter() metrics.Counter
	// MeshConfigMismatchGauge is the count of entry points another instance has with another configuration, by peer.
	MeshConfigMismatchGauge() metrics.Gauge
}

// meshSatelliteRegistry is the part of Registry holding the metrics of the satellites of the Traefix mesh.
type meshSatelliteRegistry interface {
	// MeshSatellitesGauge is the count of satellites this instance reaches back, by state.
	MeshSatellitesGauge() metrics.Gauge
	// MeshCertificatesDeliveredGauge is the count of certificates given to a satellite, by peer.
	MeshCertificatesDeliveredGauge() metrics.Gauge
}

type meshMetrics struct {
	peers    metrics.Gauge
	linkUp   metrics.Gauge
	linkRTT  metrics.Gauge
	requests metrics.Counter
	received metrics.Counter
	routes   metrics.Gauge
	exported metrics.Gauge
	packets  metrics.Counter
	mismatch metrics.Gauge

	satellites metrics.Gauge
	delivered  metrics.Gauge
}

func (r *standardRegistry) MeshRoutesGauge() metrics.Gauge {
	return r.traefix.mesh.routes
}

func (r *standardRegistry) MeshExportedRoutesGauge() metrics.Gauge {
	return r.traefix.mesh.exported
}

func (r *standardRegistry) MeshUDPPacketsCounter() metrics.Counter {
	return r.traefix.mesh.packets
}

func (r *standardRegistry) MeshConfigMismatchGauge() metrics.Gauge {
	return r.traefix.mesh.mismatch
}

func (r *standardRegistry) MeshSatellitesGauge() metrics.Gauge {
	return r.traefix.mesh.satellites
}

func (r *standardRegistry) MeshCertificatesDeliveredGauge() metrics.Gauge {
	return r.traefix.mesh.delivered
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
	var peers, linkUp, linkRTT, routes, exported, mismatch, satellites, delivered []metrics.Gauge
	var requests, received, packets []metrics.Counter

	for _, r := range registries {
		if g := r.MeshRoutesGauge(); g != nil {
			routes = append(routes, g)
		}
		if g := r.MeshExportedRoutesGauge(); g != nil {
			exported = append(exported, g)
		}
		if g := r.MeshConfigMismatchGauge(); g != nil {
			mismatch = append(mismatch, g)
		}
		if g := r.MeshSatellitesGauge(); g != nil {
			satellites = append(satellites, g)
		}
		if g := r.MeshCertificatesDeliveredGauge(); g != nil {
			delivered = append(delivered, g)
		}
		if c := r.MeshUDPPacketsCounter(); c != nil {
			packets = append(packets, c)
		}
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
		routes:   multi.NewGauge(routes...),
		exported: multi.NewGauge(exported...),
		packets:  multi.NewCounter(packets...),
		mismatch: multi.NewGauge(mismatch...),

		satellites: multi.NewGauge(satellites...),
		delivered:  multi.NewGauge(delivered...),
	}
}

func newPrometheusMeshMetrics() (meshMetrics, []vector) {
	peers := newGaugeFrom(stdprometheus.GaugeOpts{Name: meshPeersName, Help: meshPeersHelp}, []string{"state"})
	linkUp := newGaugeFrom(stdprometheus.GaugeOpts{Name: meshLinkUpName, Help: meshLinkUpHelp}, []string{labelPeer, "transport"})
	linkRTT := newGaugeFrom(stdprometheus.GaugeOpts{Name: meshLinkRTTName, Help: meshLinkRTTHelp}, []string{labelPeer})
	requests := newCounterFrom(stdprometheus.CounterOpts{Name: meshRequestsName, Help: meshRequestsHelp}, []string{labelPeer, labelKind, labelResult})
	received := newCounterFrom(stdprometheus.CounterOpts{Name: meshReceivedName, Help: meshReceivedHelp}, []string{labelPeer, labelKind, labelResult})

	routes := newGaugeFrom(stdprometheus.GaugeOpts{Name: meshRoutesName, Help: meshRoutesHelp}, []string{labelPeer, labelKind})
	exported := newGaugeFrom(stdprometheus.GaugeOpts{Name: meshExportedName, Help: meshExportedHelp}, []string{labelKind})
	packets := newCounterFrom(stdprometheus.CounterOpts{Name: meshPacketsName, Help: meshPacketsHelp}, []string{labelPeer, "path", "direction"})
	mismatch := newGaugeFrom(stdprometheus.GaugeOpts{Name: meshMismatchName, Help: meshMismatchHelp}, []string{labelPeer})
	satellites := newGaugeFrom(stdprometheus.GaugeOpts{Name: meshSatellitesName, Help: meshSatellitesHelp}, []string{"state"})
	delivered := newGaugeFrom(stdprometheus.GaugeOpts{Name: meshDeliveredName, Help: meshDeliveredHelp}, []string{labelPeer})

	m := meshMetrics{
		peers: peers, linkUp: linkUp, linkRTT: linkRTT, requests: requests, received: received,
		routes: routes, exported: exported, packets: packets, mismatch: mismatch,
		satellites: satellites, delivered: delivered,
	}

	return m, []vector{
		peers.gv, linkUp.gv, linkRTT.gv, requests.cv, received.cv, routes.gv, exported.gv, packets.cv, mismatch.gv,
		satellites.gv, delivered.gv,
	}
}

func newDatadogMeshMetrics(client *dogstatsd.Dogstatsd) meshMetrics {
	return meshMetrics{
		peers:   client.NewGauge(statsdMeshPeersName),
		linkUp:  client.NewGauge(statsdMeshLinkUpName),
		linkRTT: client.NewGauge(statsdMeshLinkRTTName),

		requests: client.NewCounter(statsdMeshRequestsName, 1.0),
		received: client.NewCounter(statsdMeshReceivedName, 1.0),
		routes:   client.NewGauge(statsdMeshRoutesName),
		exported: client.NewGauge(statsdMeshExportedName),
		packets:  client.NewCounter(statsdMeshPacketsName, 1.0),
		mismatch: client.NewGauge(statsdMeshMismatchName),

		satellites: client.NewGauge(statsdMeshSatellitesName),
		delivered:  client.NewGauge(statsdMeshDeliveredName),
	}
}

func newStatsdMeshMetrics(client *statsd.Statsd) meshMetrics {
	return meshMetrics{
		peers:   client.NewGauge(statsdMeshPeersName),
		linkUp:  client.NewGauge(statsdMeshLinkUpName),
		linkRTT: client.NewGauge(statsdMeshLinkRTTName),

		requests: client.NewCounter(statsdMeshRequestsName, 1.0),
		received: client.NewCounter(statsdMeshReceivedName, 1.0),
		routes:   client.NewGauge(statsdMeshRoutesName),
		exported: client.NewGauge(statsdMeshExportedName),
		packets:  client.NewCounter(statsdMeshPacketsName, 1.0),
		mismatch: client.NewGauge(statsdMeshMismatchName),

		satellites: client.NewGauge(statsdMeshSatellitesName),
		delivered:  client.NewGauge(statsdMeshDeliveredName),
	}
}

func newInfluxDB2MeshMetrics(store *influx.Influx) meshMetrics {
	return meshMetrics{
		peers:   store.NewGauge(influxDBFingerprintPrefix + statsdMeshPeersName),
		linkUp:  store.NewGauge(influxDBFingerprintPrefix + statsdMeshLinkUpName),
		linkRTT: store.NewGauge(influxDBFingerprintPrefix + statsdMeshLinkRTTName),

		requests: store.NewCounter(influxDBFingerprintPrefix + statsdMeshRequestsName),
		received: store.NewCounter(influxDBFingerprintPrefix + statsdMeshReceivedName),
		routes:   store.NewGauge(influxDBFingerprintPrefix + statsdMeshRoutesName),
		exported: store.NewGauge(influxDBFingerprintPrefix + statsdMeshExportedName),
		packets:  store.NewCounter(influxDBFingerprintPrefix + statsdMeshPacketsName),
		mismatch: store.NewGauge(influxDBFingerprintPrefix + statsdMeshMismatchName),

		satellites: store.NewGauge(influxDBFingerprintPrefix + statsdMeshSatellitesName),
		delivered:  store.NewGauge(influxDBFingerprintPrefix + statsdMeshDeliveredName),
	}
}

func newOTLPMeshMetrics(meter metric.Meter) meshMetrics {
	return meshMetrics{
		peers:   newOTLPGaugeFrom(meter, meshPeersName, meshPeersHelp, "1"),
		linkUp:  newOTLPGaugeFrom(meter, meshLinkUpName, meshLinkUpHelp, "1"),
		linkRTT: newOTLPGaugeFrom(meter, meshLinkRTTName, meshLinkRTTHelp, "s"),

		requests: newOTLPCounterFrom(meter, meshRequestsName, meshRequestsHelp),
		received: newOTLPCounterFrom(meter, meshReceivedName, meshReceivedHelp),
		routes:   newOTLPGaugeFrom(meter, meshRoutesName, meshRoutesHelp, "1"),
		exported: newOTLPGaugeFrom(meter, meshExportedName, meshExportedHelp, "1"),
		packets:  newOTLPCounterFrom(meter, meshPacketsName, meshPacketsHelp),
		mismatch: newOTLPGaugeFrom(meter, meshMismatchName, meshMismatchHelp, "1"),

		satellites: newOTLPGaugeFrom(meter, meshSatellitesName, meshSatellitesHelp, "1"),
		delivered:  newOTLPGaugeFrom(meter, meshDeliveredName, meshDeliveredHelp, "1"),
	}
}
