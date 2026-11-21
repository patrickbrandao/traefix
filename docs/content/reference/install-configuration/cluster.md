---
title: "Traefik Cluster Documentation"
description: "Join the Traefix instances that share a Valkey or Redis server into a cluster: each instance announces itself to the others, with an identifier and a priority."
---

The `cluster` option joins the Traefix instances that use the same Valkey or Redis server and the same `prefix` into a cluster. Each instance announces itself to the other instances with its identifier, its priority and its version. The option is specific to Traefix, the fork of Traefik Proxy.

The instances of a cluster can share their ACME account and certificates, with the [`redis` certificate store](./tls/certificate-resolvers/acme-certstore.md#redis-as-the-primary-store) of a certificate resolver.

The `priority` option sets which instance requests the certificates of such a resolver: see [Certificate Requests in a Cluster](./tls/certificate-resolvers/acme-certstore.md#certificate-requests-in-a-cluster).

The instances of a cluster also answer the HTTP-01 challenges that the other instances started: see [HTTP-01 Challenge in a Cluster](./tls/certificate-resolvers/acme-certstore.md#http-01-challenge-in-a-cluster).

Without the `cluster` option, the instance connects to no Redis server and behaves as in Traefik.

## Configuration Examples

```yaml tab="File (YAML)"
cluster:
  # Unique among the instances. Defaults to the host name.
  nodeID: srv1
  priority: 100
  redis:
    endpoints:
      - "valkey.example.com:6379"
    username: traefix
    password: my-password
```

```toml tab="File (TOML)"
[cluster]
  # Unique among the instances. Defaults to the host name.
  nodeID = "srv1"
  priority = 100
  [cluster.redis]
    endpoints = ["valkey.example.com:6379"]
    username = "traefix"
    password = "my-password"
```

```bash tab="CLI"
--cluster.nodeID=srv1
--cluster.priority=100
--cluster.redis.endpoints=valkey.example.com:6379
--cluster.redis.username=traefix
--cluster.redis.password=my-password
```

### Redis Sentinel

Set `masterName` to the name of the master set, and list the Sentinels in `endpoints`. The instance follows the master that the Sentinels elect.

```yaml tab="File (YAML)"
cluster:
  redis:
    endpoints:
      - "sentinel1.example.com:26379"
      - "sentinel2.example.com:26379"
      - "sentinel3.example.com:26379"
    masterName: mymaster
```

```toml tab="File (TOML)"
[cluster.redis]
  endpoints = ["sentinel1.example.com:26379", "sentinel2.example.com:26379", "sentinel3.example.com:26379"]
  masterName = "mymaster"
```

```bash tab="CLI"
--cluster.redis.endpoints=sentinel1.example.com:26379,sentinel2.example.com:26379,sentinel3.example.com:26379
--cluster.redis.masterName=mymaster
```

### Redis Cluster

List several nodes in `endpoints` and leave `masterName` empty. The instance treats the endpoints as the nodes of a Redis Cluster.

```yaml tab="File (YAML)"
cluster:
  redis:
    endpoints:
      - "node1.example.com:6379"
      - "node2.example.com:6379"
      - "node3.example.com:6379"
```

```toml tab="File (TOML)"
[cluster.redis]
  endpoints = ["node1.example.com:6379", "node2.example.com:6379", "node3.example.com:6379"]
```

```bash tab="CLI"
--cluster.redis.endpoints=node1.example.com:6379,node2.example.com:6379,node3.example.com:6379
```

## Configuration Options

| Field | Description | Default | Required |
|:------|:------------|:--------|:---------|
| <a id="opt-cluster-nodeID" href="#opt-cluster-nodeID" title="#opt-cluster-nodeID">`cluster.nodeID`</a> | Identifier of the instance, unique among the instances of the cluster. Starts with a letter or a digit, followed by letters, digits, dots, underscores or hyphens. | Host name | No |
| <a id="opt-cluster-priority" href="#opt-cluster-priority" title="#opt-cluster-priority">`cluster.priority`</a> | Priority of the instance to request the ACME certificates of the cluster, zero or more. The instance with the highest priority requests them, and an instance with the priority 0 requests none. | 100 | No |
| <a id="opt-cluster-prefix" href="#opt-cluster-prefix" title="#opt-cluster-prefix">`cluster.prefix`</a> | Prefix of the keys and channels of the cluster in the Redis server. The instances with the same server and the same prefix form one cluster. Same characters as `cluster.nodeID`. | `traefix` | No |
| <a id="opt-cluster-heartbeatInterval" href="#opt-cluster-heartbeatInterval" title="#opt-cluster-heartbeatInterval">`cluster.heartbeatInterval`</a> | Interval between two announcements of the instance. | 5s | No |
| <a id="opt-cluster-nodeTimeout" href="#opt-cluster-nodeTimeout" title="#opt-cluster-nodeTimeout">`cluster.nodeTimeout`</a> | Time without announcement after which the other instances consider the instance gone. At least twice `cluster.heartbeatInterval`. | 15s | No |
| <a id="opt-cluster-redis-endpoints" href="#opt-cluster-redis-endpoints" title="#opt-cluster-redis-endpoints">`cluster.redis.endpoints`</a> | Address of the server, addresses of the nodes of a Redis Cluster, or addresses of the Sentinels when `cluster.redis.masterName` is set. | `localhost:6379` | Yes |
| <a id="opt-cluster-redis-username" href="#opt-cluster-redis-username" title="#opt-cluster-redis-username">`cluster.redis.username`</a> | Username to connect to the server. | "" | No |
| <a id="opt-cluster-redis-password" href="#opt-cluster-redis-password" title="#opt-cluster-redis-password">`cluster.redis.password`</a> | Password to connect to the server. | "" | No |
| <a id="opt-cluster-redis-db" href="#opt-cluster-redis-db" title="#opt-cluster-redis-db">`cluster.redis.db`</a> | Database selected after connecting to the server. A Redis Cluster only has the database 0. | 0 | No |
| <a id="opt-cluster-redis-masterName" href="#opt-cluster-redis-masterName" title="#opt-cluster-redis-masterName">`cluster.redis.masterName`</a> | Name of the Sentinel master set. | "" | No |
| <a id="opt-cluster-redis-sentinelUsername" href="#opt-cluster-redis-sentinelUsername" title="#opt-cluster-redis-sentinelUsername">`cluster.redis.sentinelUsername`</a> | Username to connect to the Sentinels. | "" | No |
| <a id="opt-cluster-redis-sentinelPassword" href="#opt-cluster-redis-sentinelPassword" title="#opt-cluster-redis-sentinelPassword">`cluster.redis.sentinelPassword`</a> | Password to connect to the Sentinels. | "" | No |
| <a id="opt-cluster-redis-tls" href="#opt-cluster-redis-tls" title="#opt-cluster-redis-tls">`cluster.redis.tls`</a> | Connects to the server with TLS. The `ca`, `cert`, `key` and `insecureSkipVerify` options set the certificate authority, the client certificate and the verification of the server certificate. | | No |
| <a id="opt-cluster-redis-dialTimeout" href="#opt-cluster-redis-dialTimeout" title="#opt-cluster-redis-dialTimeout">`cluster.redis.dialTimeout`</a> | Timeout to establish a connection. | 5s | No |
| <a id="opt-cluster-redis-readTimeout" href="#opt-cluster-redis-readTimeout" title="#opt-cluster-redis-readTimeout">`cluster.redis.readTimeout`</a> | Timeout of a read on a connection. | 3s | No |
| <a id="opt-cluster-redis-writeTimeout" href="#opt-cluster-redis-writeTimeout" title="#opt-cluster-redis-writeTimeout">`cluster.redis.writeTimeout`</a> | Timeout of a write on a connection. | 3s | No |

Traefix doesn't start when the `cluster` option has no `redis` endpoints, a negative `priority`, or a `nodeID` or `prefix` with other characters than the ones listed.

## Behavior

- **Each instance needs its own `nodeID`.** When two running instances announce the same `nodeID`, each instance logs an error at each announcement.
- **An instance that stops leaves the cluster at once.** An instance that crashes, or loses the Redis server, is gone for the other instances after `nodeTimeout`. The Redis server counts this time, so the clocks of the instances don't have to agree.
- **An instance that cannot reach the Redis server keeps serving requests.** The instance logs the loss once, tries again at each `heartbeatInterval`, and logs when it joins the cluster again.

## Keys in the Redis Server

The names below use the default `traefix` prefix.

| Key | Contains |
|:----|:---------|
| <a id="opt-traefixnodenodeID" href="#opt-traefixnodenodeID" title="#opt-traefixnodenodeID">`traefix:node:<nodeID>`</a> | Announcement of an instance, in JSON: identifier, priority, version and start time. Expires after `nodeTimeout`. |
| <a id="opt-traefixnodes" href="#opt-traefixnodes" title="#opt-traefixnodes">`traefix:nodes`</a> | Set of the identifiers of the instances. |
| <a id="opt-traefixlockname" href="#opt-traefixlockname" title="#opt-traefixlockname">`traefix:lock:<name>`</a> | Lock held by one instance at a time. Expires when its holder stops extending it. |
| <a id="opt-traefixevents" href="#opt-traefixevents" title="#opt-traefixevents">`traefix:events`</a> | Channel of the messages the instances send to each other. |
| <a id="opt-traefixacmehttptokendomain" href="#opt-traefixacmehttptokendomain" title="#opt-traefixacmehttptokendomain">`traefix:acme:http:<token>:<domain>`</a> | Answer of an HTTP-01 challenge in progress. Expires after 10 minutes. |

The `redis` certificate store adds the [keys of each certificate resolver](./tls/certificate-resolvers/acme-certstore.md#keys-in-the-redis-server).

When the Redis server uses access control lists, give the user of the instances access to the keys and channels that start with the prefix, for example `~traefix:* &traefix:*`. With the `redis` certificate store and its copy for traefik-certs, add `~tcerts:* &tcerts:*`.
