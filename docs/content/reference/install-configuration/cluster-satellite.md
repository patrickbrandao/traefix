---
title: "Traefik Cluster Satellite Documentation"
description: "Run a Traefix instance behind NAT as a satellite of the cluster mesh: it dials the other instances, and serves its routers through the links it opens."
---

A satellite is a Traefix instance on a host that accepts no connection from outside, such as a host behind a home router, behind carrier-grade NAT, or behind a firewall that only lets connections out. The satellite dials the other instances of the [mesh](./cluster-mesh.md), and announces its routers to them. The option is specific to Traefix, the fork of Traefik Proxy.

!!! warning "Work in progress"

    In this version, a satellite links to the other instances and introduces itself as a satellite. The other instances don't reach it back yet, so they don't serve its routers.

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
- **The other instances don't dial a satellite.** They don't tell each other about it, and an address of a satellite in a list of peers is dropped with a warning.
- **A satellite doesn't link to another satellite.**
- **A satellite announces its routers, and imports none.** The routers of the other instances don't become routers of the satellite.
- **A satellite doesn't ask a CA for a certificate.** Its ACME certificate resolvers don't start, and Traefix logs a warning. The routers of the satellite still name the resolvers, which are the ones with the same names on the other instances.
- **A satellite is a full member of the cluster.** It holds the token of the cluster. Anyone who takes over the host of a satellite can join the mesh and announce any router: to remove a satellite, [change the token](./cluster-mesh.md#change-the-token).

In the [API](./cluster-mesh.md#api), `GET /api/mesh` returns `role`: `satellite` or `member`. Each link and each inbound instance has the role that the other instance told.
