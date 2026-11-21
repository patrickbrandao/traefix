---
title: "Traefik Cluster Mesh Documentation"
description: "Link the Traefix instances of a cluster with the mesh option: the cluster token gives each instance an identity that the other instances verify."
---

The `cluster.mesh` option links the Traefix instances of a [cluster](./cluster.md) to each other, so that an instance can reach the services that another instance discovered on its own host. The option is specific to Traefix, the fork of Traefik Proxy.

!!! warning "Work in progress"

    In this version, an instance checks the options of this page, prepares its identity from the cluster token, and tells the cluster its address. The instance opens no link and forwards no request yet.

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

The mesh doesn't need the `cluster.redis` option. With a Redis server, the instances also find each other through it.

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

Traefix doesn't start when the `cluster.mesh` option is set without a token, when the token has less than 32 characters or its file can't be read, or when an address has no port.

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
