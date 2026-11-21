---
title: "Traefik Cluster Mesh Documentation"
description: "Link the Traefix instances of a cluster with the mesh option: the cluster token gives each instance an identity that the other instances verify."
---

The `cluster.mesh` option links the Traefix instances of a [cluster](./cluster.md) to each other, so that an instance can reach the services that another instance discovered on its own host. The option is specific to Traefix, the fork of Traefik Proxy.

!!! warning "Work in progress"

    In this version, the mesh forwards the HTTP requests. The TCP and UDP routers are announced and not forwarded yet.

## Configuration Examples

```yaml tab="File (YAML)"
cluster:
  nodeID: srv1
  tokenFile: /run/secrets/cluster-token
  mesh:
    address: ":7443"
    advertise:
      - "203.0.113.10:7443"
    peers:
      - "srv2.example.com:7443"
```

```toml tab="File (TOML)"
[cluster]
  nodeID = "srv1"
  tokenFile = "/run/secrets/cluster-token"
  [cluster.mesh]
    address = ":7443"
    advertise = ["203.0.113.10:7443"]
    peers = ["srv2.example.com:7443"]
```

```bash tab="CLI"
--cluster.nodeID=srv1
--cluster.tokenFile=/run/secrets/cluster-token
--cluster.mesh.address=:7443
--cluster.mesh.advertise=203.0.113.10:7443
--cluster.mesh.peers=srv2.example.com:7443
```

The mesh doesn't need the `cluster.redis` option. With a Redis server, the instances also [find each other](#find-the-other-instances) through it.

## Configuration Options

| Field | Description | Default | Required |
|:------|:------------|:--------|:---------|
| <a id="opt-cluster-token" href="#opt-cluster-token" title="#opt-cluster-token">`cluster.token`</a> | Secret shared by the instances of the cluster, of 32 characters or more. Can't be set with `cluster.tokenFile`. | "" | Yes, or `cluster.tokenFile` |
| <a id="opt-cluster-tokenFile" href="#opt-cluster-tokenFile" title="#opt-cluster-tokenFile">`cluster.tokenFile`</a> | File that holds the token. Traefix reads the file at startup and ignores the spaces and line breaks around the token. | "" | Yes, or `cluster.token` |
| <a id="opt-cluster-additionalTokens" href="#opt-cluster-additionalTokens" title="#opt-cluster-additionalTokens">`cluster.additionalTokens`</a> | Tokens that the instance accepts from the other instances, in addition to its own. See [Change the Token](#change-the-token). | [] | No |
| <a id="opt-cluster-mesh-address" href="#opt-cluster-mesh-address" title="#opt-cluster-mesh-address">`cluster.mesh.address`</a> | Address the instance listens on for the other instances, in UDP and in TCP. | `:7443` | No |
| <a id="opt-cluster-mesh-advertise" href="#opt-cluster-mesh-advertise" title="#opt-cluster-mesh-advertise">`cluster.mesh.advertise`</a> | Addresses that the other instances use to reach this instance, each with a host and a port. See [Advertised Address](#advertised-address). | Guessed | No |
| <a id="opt-cluster-mesh-transport" href="#opt-cluster-mesh-transport" title="#opt-cluster-mesh-transport">`cluster.mesh.transport`</a> | Transport of the links: `auto` (QUIC, then TCP when QUIC doesn't get through), `quic` or `tcp`. | `auto` | No |
| <a id="opt-cluster-mesh-peers" href="#opt-cluster-mesh-peers" title="#opt-cluster-mesh-peers">`cluster.mesh.peers`</a> | Addresses of instances to link to at startup, each with a host and a port. | [] | No |
| <a id="opt-cluster-mesh-dns" href="#opt-cluster-mesh-dns" title="#opt-cluster-mesh-dns">`cluster.mesh.dns`</a> | DNS names whose SRV or TXT records list the addresses of the instances. | [] | No |
| <a id="opt-cluster-mesh-peersFile" href="#opt-cluster-mesh-peersFile" title="#opt-cluster-mesh-peersFile">`cluster.mesh.peersFile`</a> | File that keeps the instances learned from the other instances, to link to them at the next startup. | "" | No |
| <a id="opt-cluster-mesh-linkTimeout" href="#opt-cluster-mesh-linkTimeout" title="#opt-cluster-mesh-linkTimeout">`cluster.mesh.linkTimeout`</a> | Time without an answer after which the instance considers a link lost. At least 1 second. | 10s | No |
| <a id="opt-cluster-mesh-peerRetention" href="#opt-cluster-mesh-peerRetention" title="#opt-cluster-mesh-peerRetention">`cluster.mesh.peerRetention`</a> | Time without a link after which the instance forgets a learned instance. At least `cluster.mesh.linkTimeout`. | 168h | No |
| <a id="opt-cluster-mesh-export" href="#opt-cluster-mesh-export" title="#opt-cluster-mesh-export">`cluster.mesh.export`</a> | Routers that the instance announces to the other instances: `all`, or `labeled` for the routers that opt in. | `all` | No |
| <a id="opt-cluster-mesh-balance" href="#opt-cluster-mesh-balance" title="#opt-cluster-mesh-balance">`cluster.mesh.balance`</a> | Choice between the instances with the same priority for a router: `clientIP` sends a client to the same instance whichever instance the client reaches, `roundRobin` spreads the requests evenly. | `clientIP` | No |

Traefix doesn't start when the `cluster.mesh` option is set without a token, when the token has less than 32 characters or its file can't be read, when an address has no port, or when the port of `cluster.mesh.address` is in use.

## Links

Each instance listens on `cluster.mesh.address` and opens a link to every instance it [finds](#find-the-other-instances). A link serves one direction: the instance that opens the link is the one that sends requests through it. Two instances that reach each other have two links.

- **A link uses QUIC, or TLS over TCP.** With `transport: auto`, the instance tries QUIC for 3 seconds, then TCP. An instance linked over TCP tries QUIC again every 5 minutes, and moves the link when QUIC gets through.
- **Open the port in UDP and in TCP.** With `transport: quic` the instance only listens in UDP, and with `transport: tcp` only in TCP.
- **Both ends verify each other.** A link needs TLS 1.3 and a certificate of the [cluster token](#cluster-token) on both sides. An instance with another token gets a warning in its logs that names the cause.
- **A link is lost after `linkTimeout` without an answer.** The instance pings the other end three times per `linkTimeout`, and logs the loss. It then dials again after 1 second, and doubles the wait up to 30 seconds while the other instance is away.
- **The list of peers can name the instance itself.** The instance notices that it reached itself and stops dialing that address, so that every instance can get the same list.
- **Two instances can't share a `nodeID`.** An instance that reaches another one with its own `nodeID` doesn't link to it, and logs an error.
- **An instance has one link to another instance, whatever the number of its addresses.** The instance tries the addresses in order, the one that worked last first, then the ones the other instance advertises.

## Find the Other Instances

An instance links to every instance it learns about, from five sources:

| Source | Option | Learns |
|:-------|:-------|:-------|
| <a id="opt-List" href="#opt-List" title="#opt-List">List</a> | `cluster.mesh.peers` | The addresses of the list. |
| <a id="opt-Redis-server" href="#opt-Redis-server" title="#opt-Redis-server">Redis server</a> | `cluster.redis` | The instances that announce themselves in the Redis server, with the addresses they advertise. |
| <a id="opt-DNS" href="#opt-DNS" title="#opt-DNS">DNS</a> | `cluster.mesh.dns` | The addresses of the SRV records of each name, and of its TXT records that start with `traefix-mesh=`. The instance looks the names up every minute. |
| <a id="opt-Other-instances" href="#opt-Other-instances" title="#opt-Other-instances">Other instances</a> | | An instance that opens a link tells its addresses and the instances it is in contact with. Each instance asks the same of the instances it links to, every 30 seconds. |
| <a id="opt-File" href="#opt-File" title="#opt-File">File</a> | `cluster.mesh.peersFile` | The instances that the instance was in contact with before it stopped. |

- **A new instance only needs to know one instance.** The instance it links to links back to it, and tells it about the others. Every instance ends up with a link to every other instance.
- **The instances don't relay.** An instance only reaches the instances it has a link to. Open the port of the mesh between every pair of instances.
- **An instance of the cluster with another token is left alone.** The instance logs a warning once, and doesn't dial it.
- **The file speeds the startup up.** The instance writes the file when the instances it knows change, every 5 minutes, and when it stops. At startup, it links to the instances of the file before the other sources answer. The instance still starts when the file is missing or can't be read.
- **A learned instance is forgotten after `peerRetention` without contact.** The time goes on counting across the restarts. The addresses of `cluster.mesh.peers` are never forgotten.

DNS records for two instances, as SRV records or as one TXT record:

```text
_traefix-mesh._udp.example.com. 300 IN SRV 0 0 7443 srv1.example.com.
_traefix-mesh._udp.example.com. 300 IN SRV 0 0 7443 srv2.example.com.

mesh.example.com. 300 IN TXT "traefix-mesh=srv1.example.com:7443,srv2.example.com:7443"
```

The priority and the weight of the SRV records are not used.

## Routers of the Other Instances

Each instance announces its routers to the instances that link to it, and creates a router for each router that another instance announces.

- **An instance announces the routers of its providers.** It leaves out the routers of the dashboard and of the API, and the routers it learned from the mesh. With `export: labeled`, it only announces the routers with `mesh.export=true`.
- **An announcement carries what matches a request.** That is the rule, the priority, the entry points and the TLS options, certificate resolver and domains of the router. The middlewares, the service and the servers stay with the instance that owns the router.
- **The router that an instance creates is named after the announced router and its provider.** The `app@docker` router of another instance becomes `app-docker@mesh`.
- **The router uses the entry points that the instance has with the same names.** Use the same entry point names on every instance. A router with no entry point in common is not created.
- **The TLS options and the certificate resolver of the router must exist on the instance, with the same names.** The instance requests the certificate of the router as it does for its own routers.
- **The middlewares of the entry point apply to the router.** Declare them with their provider, such as `waf@file`: a name without a provider is looked up among the middlewares of the mesh, which has none.
- **An instance keeps its own router.** When the instance has a router with the name of an announced router, it creates no router for the announcement. With the same rule, entry points and TLS, the two routers are one route, and the [priorities](#where-a-request-is-served) choose the instance that serves each request.
- **A change reaches the other instances at once.** They follow the routers of an instance for as long as their link to it is up, and remove them when the link is lost.
- **Routers with the same name and another rule are served apart.** While a router is being changed on its instances one at a time, the other instances create a second router, with a suffix in its name, and log a warning.

The options of a router, in the routing configuration:

| Field | Description | Default |
|:------|:------------|:--------|
| <a id="opt-mesh-export" href="#opt-mesh-export" title="#opt-mesh-export">`mesh.export`</a> | Announces the router to the other instances, or not. When unset, `cluster.mesh.export` decides. | |
| <a id="opt-mesh-priority" href="#opt-mesh-priority" title="#opt-mesh-priority">`mesh.priority`</a> | Chooses between the instances that have the router, from 0 to 1000. See [Where a Request Is Served](#where-a-request-is-served). | 0 |

```yaml tab="Labels"
labels:
  - "traefik.http.routers.app.rule=Host(`app.example.com`)"
  - "traefik.http.routers.app.mesh.priority=10"
  - "traefik.http.routers.internal.rule=Host(`internal.example.com`)"
  - "traefik.http.routers.internal.mesh.export=false"
```

```yaml tab="File (YAML)"
http:
  routers:
    app:
      rule: "Host(`app.example.com`)"
      service: app
      mesh:
        priority: 10
    internal:
      rule: "Host(`internal.example.com`)"
      service: internal
      mesh:
        export: false
```

The TCP and UDP routers have the same options, and are announced. The instances create no router for them yet.

## Requests Through the Mesh

A request for a router that the instance learned from the mesh goes to an instance that owns the router, through the link to it. The request makes one hop at most.

| Step | Instance that received the request | Instance that owns the router |
|:-----|:-----------------------------------|:------------------------------|
| <a id="opt-TLS-and-fingerprint" href="#opt-TLS-and-fingerprint" title="#opt-TLS-and-fingerprint">TLS and fingerprint</a> | Yes | No |
| <a id="opt-Middlewares-of-the-entry-point" href="#opt-Middlewares-of-the-entry-point" title="#opt-Middlewares-of-the-entry-point">Middlewares of the entry point</a> | Yes | No |
| <a id="opt-Rule-of-the-router" href="#opt-Rule-of-the-router" title="#opt-Rule-of-the-router">Rule of the router</a> | Yes | No |
| <a id="opt-Middlewares-of-the-router" href="#opt-Middlewares-of-the-router" title="#opt-Middlewares-of-the-router">Middlewares of the router</a> | No | Yes |
| <a id="opt-Service-load-balancing-health-check-sticky-session" href="#opt-Service-load-balancing-health-check-sticky-session" title="#opt-Service-load-balancing-health-check-sticky-session">Service: load balancing, health check, sticky session</a> | No | Yes |

- **The instance that owns the router sees the client.** The request has the address and the port of the client, its TLS version, cipher and server name, and the `X-Forwarded-*` and `X-Real-Ip` headers that the entry point set. The `ipAllowList` and `rateLimit` middlewares of the router, and the `X-Forwarded-For` header sent to the server, are the same as for a request received directly.
- **The fingerprints follow the request.** The `X-Traefix-*` headers reach the server, and the [`fingerprintList`](../routing-configuration/http/middlewares/fingerprintlist.md) middleware, the access log and the traces of the instance that owns the router use the fingerprints that the entry point measured. Enable the `fingerprint` option on the entry points of every instance that receives requests.
- **The middlewares of the entry points only run where the request arrives.** A request that reaches the instance that owns the router through the mesh doesn't go through its entry points. Declare the same entry point middlewares on every instance.
- **WebSocket and gRPC work.** A request with an `Upgrade` header travels in a stream of its own, and streaming requests and responses flow in both directions at the same time.
- **The client certificate stops at the entry point.** The [`passTLSClientCert`](../routing-configuration/http/middlewares/passtlsclientcert.md) middleware of a router has nothing to pass for a request that came through the mesh.
- **An answer of the mesh carries a `Proxy-Status` header.** The instance answers `503` when it has no link to an instance that owns the router, and `502` when the request could not be delivered. An error of the service itself comes back as the service sent it, without the header.

In the access log, the instance that received the request records its router, with `mesh://<nodeID>` as the service URL. The instance that owns the router records the request with the `traefix-mesh` entry point, the address of the client and its own router.

### Where a Request Is Served

The instances that own a router are the candidates for its requests: the instance that received the request, when it has the router, and the other instances that announce it.

1. The instance takes the candidates that can serve the request now, and the highest `mesh.priority` among them.
2. With a priority above 0, the candidates with that priority get the requests. The [`balance`](#configuration-options) option chooses between several of them.
3. With the default priority, 0, the instance serves the request itself when it has the router. Otherwise, the `balance` option chooses between the other instances.

| Instance that received the request | Other instances | Where the request goes |
|:-----------------------------------|:----------------|:-----------------------|
| Default | `srv2`: default | The instance itself. `srv2` only when the instance can't serve it. |
| No router | `srv2` and `srv3`: default | `srv2` or `srv3`. |
| Default | `srv2`: 10 | `srv2`. The instance itself when `srv2` can't serve it. |
| 10 | `srv2`: 10, `srv3`: default | The instance itself or `srv2`. `srv3` when neither can. |
| 5 | `srv2`: 10 | `srv2`. The instance itself when `srv2` can't serve it. |

Different priorities make a primary and its backups. The same priority, above 0, shares the requests.

- **With `balance: clientIP`, a client keeps its instance.** The choice only depends on the address of the client and on the candidates, so every instance that the client reaches sends it to the same one. With `balance: roundRobin`, the requests are spread evenly, and each instance counts on its own.
- **An instance tells the others when its router has no server.** The servers of the service are all down, by its health check, or gone. The other instances stop sending it requests, and an instance in that state sends the requests it receives to another candidate.
- **An instance without any other candidate serves the request itself.** Its answer is the one it gives without the mesh, such as `503` when its servers are down.
- **An instance that fails three requests in a row is left alone for the router.** The instance comes back after 5 seconds, and after twice as long each time it fails again, up to 1 minute.
- **A request that didn't leave the instance goes to the next candidate.** That is the case when the link was just lost, or when the other instance answers that it no longer has the router. A request that was sent is not sent a second time: the server may have received it.
- **A request in transit to an instance that dies is lost.** It fails after `linkTimeout` at most, when the link is considered lost. The requests that follow go to the other candidates.

## API

With the [API](./api-dashboard.md) enabled, `GET /api/mesh` returns the state of the mesh on the instance:

| Field | Contains |
|:------|:---------|
| <a id="opt-links" href="#opt-links" title="#opt-links">`links`</a> | The links that the instance opens: the other instance, its addresses, where it was learned from, the transport, the time of the last ping, and the number of routers it announces. |
| <a id="opt-inbound" href="#opt-inbound" title="#opt-inbound">`inbound`</a> | The instances that opened a link to this one. |
| <a id="opt-exported" href="#opt-exported" title="#opt-exported">`exported`</a> | The routers that the instance announces. |
| <a id="opt-imported" href="#opt-imported" title="#opt-imported">`imported`</a> | The routers that the other instances announce, with the instances that announce each one and what this instance made of it: `imported` with the name of its router, `local` when it has its own router, or `skipped` with the reason. |

The rules of the announced routers appear in this answer. Protect the API as you protect the dashboard.

## Metrics

| Metric | Labels | Contains |
|:-------|:-------|:---------|
| <a id="opt-traefik-mesh-peers" href="#opt-traefik-mesh-peers" title="#opt-traefik-mesh-peers">`traefik_mesh_peers`</a> | `state` | Number of instances that this instance opens a link to, with the link `up` or `down`. An address that no instance answers at counts as `down`. |
| <a id="opt-traefik-mesh-link-up" href="#opt-traefik-mesh-link-up" title="#opt-traefik-mesh-link-up">`traefik_mesh_link_up`</a> | `peer`, `transport` | `1` for the transport, `quic` or `tcp`, that the link to the instance is up with, `0` otherwise. |
| <a id="opt-traefik-mesh-link-rtt-seconds" href="#opt-traefik-mesh-link-rtt-seconds" title="#opt-traefik-mesh-link-rtt-seconds">`traefik_mesh_link_rtt_seconds`</a> | `peer` | Time the other instance took to answer the last ping, `0` while the link is down. |
| <a id="opt-traefik-mesh-requests-total" href="#opt-traefik-mesh-requests-total" title="#opt-traefik-mesh-requests-total">`traefik_mesh_requests_total`</a> | `peer`, `kind`, `result` | Requests that the instance sent to another instance. The `kind` is `http` or `upgrade`. The `result` is `forwarded`, `error` when the request could not be delivered, or `unavailable`, with an empty `peer`, when no instance could take it. |
| <a id="opt-traefik-mesh-received-total" href="#opt-traefik-mesh-received-total" title="#opt-traefik-mesh-received-total">`traefik_mesh_received_total`</a> | `peer`, `kind`, `result` | Requests that the instance received from another instance: `served`, or `unknown` when it no longer has the router. |

The `peer` label is the `nodeID` of the other instance. An address of the list or of the DNS that no instance answered at yet has no `nodeID`: it only counts in `traefik_mesh_peers`. Datadog, StatsD and InfluxDB use the same names with dots, for example `traefik.mesh.link.up`.

## Cluster Token

The token proves to an instance that another instance belongs to its cluster. Every instance of the cluster needs the same token and the same [`cluster.prefix`](./cluster.md).

- **Generate a random token.** For example with `openssl rand -base64 32`. The instances derive a signing key from the token, so a short or guessable token would let a stranger join the cluster.
- **The token never leaves the instance.** Each instance derives a certificate authority from the token and uses it to sign its own certificate. The instances with the same token derive the same authority, and verify each other with it.
- **Compare the tokens of two instances with their logs.** At startup, each instance logs `The mesh identity of this instance is ready` with an `authority` value. Two instances with the same value hold the same token. The value doesn't reveal the token.
- **The clocks of the instances don't have to agree.** The certificates of the mesh don't expire: changing the token is what removes an instance from the cluster.

Traefix replaces the token with `xxxx` in its logs. Prefer `cluster.tokenFile` with a secret of your orchestrator to a token in the install configuration.

### Change the Token

An instance shows the certificate of its own token, and accepts the certificates of its own token and of the `cluster.additionalTokens`. To change the token without unlinking the instances, restart them one at a time at each step:

1. Add the new token to `cluster.additionalTokens` on every instance.
2. On every instance, set the new token as `cluster.token` and move the old token to `cluster.additionalTokens`.
3. Remove the old token from `cluster.additionalTokens` on every instance.

## Advertised Address

The other instances need an address to reach the instance at. Without the `cluster.mesh.advertise` option, the instance tells them:

1. The `cluster.mesh.address`, when it names one interface, such as `192.0.2.1:7443`.
2. Otherwise, the local address that the instance uses to reach the Redis server or its first peer, with the port of `cluster.mesh.address`.
3. Otherwise, the first address of the host that is neither a loopback nor a link-local address.

The instance logs the address it guessed. Set `cluster.mesh.advertise` when the instance runs in a container, behind address translation, or on a host with several addresses.

With the `cluster.redis` option, the address and the `authority` value of each instance are in its [announcement in the Redis server](./cluster.md#keys-in-the-redis-server).
