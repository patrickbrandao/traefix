---
title: "Traefik Cluster Satellite Documentation"
description: "Run a Traefix instance behind NAT as a satellite of the cluster mesh: it dials the other instances, and serves its routers through the links it opens."
---

A satellite is a Traefix instance on a host that accepts no connection from outside, such as a host behind a home router, behind carrier-grade NAT, or behind a firewall that only lets connections out. The satellite dials the other instances of the [mesh](./cluster-mesh.md), and announces its routers to them. The option is specific to Traefix, the fork of Traefik Proxy.

!!! warning "Work in progress"

    In this version, a satellite doesn't get the certificates of its routers yet.

## Configuration Examples

```yaml tab="File (YAML)"
cluster:
  nodeID: home1
  tokenFile: /run/secrets/cluster-token
  mesh:
    satellite: true
    peers:
      - "edge1.example.com:7443"
```

```toml tab="File (TOML)"
[cluster]
  nodeID = "home1"
  tokenFile = "/run/secrets/cluster-token"
  [cluster.mesh]
    satellite = true
    peers = ["edge1.example.com:7443"]
```

```bash tab="CLI"
--cluster.nodeID=home1
--cluster.tokenFile=/run/secrets/cluster-token
--cluster.mesh.satellite=true
--cluster.mesh.peers=edge1.example.com:7443
```

The satellite uses the token of the cluster, and the options of [`cluster.mesh`](./cluster-mesh.md#configuration-options).

## Configuration Options

| Field | Description | Default | Required |
|:------|:------------|:--------|:---------|
| <a id="opt-cluster-mesh-satellite" href="#opt-cluster-mesh-satellite" title="#opt-cluster-mesh-satellite">`cluster.mesh.satellite`</a> | Makes the instance a satellite of the mesh. | false | No |

Traefix doesn't start a satellite:

- without `cluster.mesh.peers` or `cluster.mesh.dns`, because the other instances never reach a satellite first.
- with `cluster.mesh.advertise`, because a satellite has no address that the other instances reach.
- with `cluster.redis`, because a satellite doesn't use the Redis server.

## How a Satellite Behaves

- **A satellite links to every instance it learns about.** It learns them from its list of peers, its DNS names and its file of peers, and from the instances it links to. An instance that joins the mesh after the satellite is learned within 30 seconds.
- **The other instances only reach a satellite back.** They don't dial it at an address of their own, don't tell each other about it, and drop an address of a satellite in a list of peers with a warning. See [How the Other Instances Reach a Satellite](#how-the-other-instances-reach-a-satellite).
- **A satellite doesn't link to another satellite.**
- **A satellite announces its routers, and imports none.** The routers of the other instances don't become routers of the satellite.
- **A satellite doesn't ask a CA for a certificate.** Its ACME certificate resolvers don't start, and Traefix logs a warning. The routers of the satellite still name the resolvers, which are the ones with the same names on the other instances.
- **A satellite is a full member of the cluster.** It holds the token of the cluster. Anyone who takes over the host of a satellite can join the mesh and announce any router: to remove a satellite, [change the token](./cluster-mesh.md#change-the-token).

In the [API](./cluster-mesh.md#api), `GET /api/mesh` returns `role`: `satellite` or `member`. Each link and each inbound instance has the role that the other instance told.

## How the Other Instances Reach a Satellite

A satellite links to an instance, and the instance dials it back at the address that link comes from, from its own mesh port. The NAT in front of the satellite already maps that address to the satellite, for the packets that come from that port, so the packets of the link back reach the satellite. The instance then uses the link back as a link of the [mesh](./cluster-mesh.md): it follows the routers of the satellite and sends it the requests, connections and UDP sessions for them.

- **The satellite needs no open port.** It only sends UDP to the mesh port of the other instances. That works behind a home router, behind carrier-grade NAT, and behind a NAT that maps each destination apart.
- **The other instances need their mesh port reachable in UDP, from the same public address that they send from.** A load balancer that changes the source port of the replies, in front of an instance, keeps the link back from reaching the satellite.
- **The instance follows the NAT.** When the NAT maps the satellite to another port, the link of the satellite moves with it, and the instance dials the satellite back at the new address. A few requests can fail while the link back opens again.
- **A satellite that stops linking is forgotten.** The instance stops dialing the satellite back after `cluster.mesh.linkTimeout` without a sign of the link of the satellite, and its routers leave the instance.
- **Without QUIC, the satellite opens the link back itself, over TCP.** When the instance doesn't reach the satellite back over QUIC within 3 seconds of its link, or when the link of the satellite is over TCP, the satellite opens a second connection to the mesh port of the instance, in TCP, and the instance sends its requests through it. The instance switches to a link back over QUIC as soon as it has one, and the satellite stops opening the connection over TCP while it is reached over QUIC.
- **Over TCP, UDP travels in a stream.** The packets of the UDP routers of the satellite arrive whole and in order, and a lost packet delays the ones after it.
- **With `transport: quic`, the satellite never opens the connection over TCP.** An instance that doesn't reach it back over QUIC doesn't serve its routers, and the satellite logs a warning.

In the [API](./cluster-mesh.md#api) of an instance, the link to a satellite has the address the satellite came from, `satellite` among its sources, and the transport `quic`, or `tcp-reverse` for the connection over TCP that the satellite opened. The `traefik_mesh_link_up` metric has the same transport.

