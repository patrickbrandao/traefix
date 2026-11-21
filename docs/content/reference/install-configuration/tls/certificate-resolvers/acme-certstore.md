---
title: "Traefik ACME Certificate Store Documentation"
description: "Choose where a Traefix ACME certificate resolver keeps its account and certificates: the acme.json storage file, one directory of PEM files per domain, or the Valkey or Redis server that the instances of a cluster share."
---

The `certStore` option of an ACME certificate resolver sets where the resolver keeps its account and certificates. You can keep them in the `storage` file, in one directory of PEM files per domain, or in the Valkey or Redis server of the [`cluster`](../../cluster.md) option, which several instances share. The option is specific to Traefix, the fork of Traefik Proxy.

One store is the primary store, which the resolver reads when Traefix starts. The other stores are exports: Traefix copies the primary store to them and doesn't read them.

Without the `certStore` option, the resolver keeps everything in the `storage` file, as in Traefik.

## Configuration Examples

```yaml tab="File (YAML)"
certificatesResolvers:
  myresolver:
    acme:
      email: your-email@example.com
      storage: /data/acme.json
      httpChallenge:
        entryPoint: web
      certStore:
        # The directory is the source of truth.
        primary: dir
        dir: /data/certs
        # The storage file receives a copy.
        export:
          - file
```

```toml tab="File (TOML)"
[certificatesResolvers.myresolver.acme]
  email = "your-email@example.com"
  storage = "/data/acme.json"
  [certificatesResolvers.myresolver.acme.httpChallenge]
    entryPoint = "web"
  [certificatesResolvers.myresolver.acme.certStore]
    # The directory is the source of truth.
    primary = "dir"
    dir = "/data/certs"
    # The storage file receives a copy.
    export = ["file"]
```

```bash tab="CLI"
--certificatesresolvers.myresolver.acme.email=your-email@example.com
--certificatesresolvers.myresolver.acme.storage=/data/acme.json
--certificatesresolvers.myresolver.acme.httpchallenge.entrypoint=web
--certificatesresolvers.myresolver.acme.certstore.primary=dir
--certificatesresolvers.myresolver.acme.certstore.dir=/data/certs
--certificatesresolvers.myresolver.acme.certstore.export=file
```

## Configuration Options

| Field | Description | Default | Required |
|:------|:------------|:--------|:---------|
| <a id="opt-certStore-primary" href="#opt-certStore-primary" title="#opt-certStore-primary">`certStore.primary`</a> | Store the resolver reads its account and certificates from: `file` for the `storage` file, `dir` for the directory of `certStore.dir`, or `redis` for the [server of the cluster](#redis-as-the-primary-store). | `file` | No |
| <a id="opt-certStore-dir" href="#opt-certStore-dir" title="#opt-certStore-dir">`certStore.dir`</a> | Directory that holds one sub-directory of [certificate files](#certificate-files) per domain. Traefix creates the directory on the first write. | "" | Yes, when `dir` is the primary store or an export |
| <a id="opt-certStore-export" href="#opt-certStore-export" title="#opt-certStore-export">`certStore.export`</a> | Stores that receive a copy of the primary store: `dir`, `file`. The primary store cannot be an export. | [] | No |
| <a id="opt-certStore-tcerts-enabled" href="#opt-certStore-tcerts-enabled" title="#opt-certStore-tcerts-enabled">`certStore.tcerts.enabled`</a> | With the `redis` primary store, writes a [copy of each certificate](#copy-for-traefik-certs) for the tools of the traefik-certs project. | true | No |
| <a id="opt-certStore-tcerts-prefix" href="#opt-certStore-tcerts-prefix" title="#opt-certStore-tcerts-prefix">`certStore.tcerts.prefix`</a> | Prefix of the keys and of the channel of the copy. | `tcerts` | No |
| <a id="opt-certStore-election-step" href="#opt-certStore-election-step" title="#opt-certStore-election-step">`certStore.election.step`</a> | With the `redis` primary store, time an instance waits for each level of priority above its own before it tries to take a [certificate request](#certificate-requests-in-a-cluster). | 3s | No |
| <a id="opt-certStore-election-lockTTL" href="#opt-certStore-election-lockTTL" title="#opt-certStore-election-lockTTL">`certStore.election.lockTTL`</a> | With the `redis` primary store, time after which the request of an instance that stopped is free for another instance. At least 10s. | 60s | No |

Traefix doesn't start when a `certStore` option names an unknown store, lists a store twice, or uses `dir` without `certStore.dir`.

### Store Combinations

| Result | `primary` | `export` |
|:-------|:----------|:---------|
| <a id="opt-The-storage-file-only-as-in-Traefik" href="#opt-The-storage-file-only-as-in-Traefik" title="#opt-The-storage-file-only-as-in-Traefik">The `storage` file only, as in Traefik</a> | `file` | |
| <a id="opt-The-storage-file-copied-to-certificate-files" href="#opt-The-storage-file-copied-to-certificate-files" title="#opt-The-storage-file-copied-to-certificate-files">The `storage` file, copied to certificate files</a> | `file` | `dir` |
| <a id="opt-Certificate-files-only" href="#opt-Certificate-files-only" title="#opt-Certificate-files-only">Certificate files only</a> | `dir` | |
| <a id="opt-Certificate-files-copied-to-the-storage-file" href="#opt-Certificate-files-copied-to-the-storage-file" title="#opt-Certificate-files-copied-to-the-storage-file">Certificate files, copied to the `storage` file</a> | `dir` | `file` |
| <a id="opt-The-server-of-the-cluster-copied-to-certificate-files-on-each-instance" href="#opt-The-server-of-the-cluster-copied-to-certificate-files-on-each-instance" title="#opt-The-server-of-the-cluster-copied-to-certificate-files-on-each-instance">The server of the cluster, copied to certificate files on each instance</a> | `redis` | `dir` |
| <a id="opt-The-server-of-the-cluster-copied-to-certificate-files-and-to-the-storage-file" href="#opt-The-server-of-the-cluster-copied-to-certificate-files-and-to-the-storage-file" title="#opt-The-server-of-the-cluster-copied-to-certificate-files-and-to-the-storage-file">The server of the cluster, copied to certificate files and to the `storage` file</a> | `redis` | `dir`, `file` |

The resolvers that set the same `storage` path share the file, and the resolvers that set the same `certStore.dir` share the directory.

## Certificate Files

For each domain of a certificate, Traefix writes these files to `<certStore.dir>/<domain>/`. The layout is the layout of the traefik-certs project, so that its `cert-get` tool reads the directory.

| File | Contains | Mode |
|:-----|:---------|:-----|
| <a id="opt-fullchain-pem" href="#opt-fullchain-pem" title="#opt-fullchain-pem">`fullchain.pem`</a> | Certificate of the domain, followed by the intermediate certificates. | `0644` |
| <a id="opt-cert-pem" href="#opt-cert-pem" title="#opt-cert-pem">`cert.pem`</a> | Certificate of the domain. | `0644` |
| <a id="opt-chain-pem" href="#opt-chain-pem" title="#opt-chain-pem">`chain.pem`</a> | Intermediate certificates. | `0644` |
| <a id="opt-privkey-pem" href="#opt-privkey-pem" title="#opt-privkey-pem">`privkey.pem`</a> | Private key. | `0600` |
| <a id="opt-cert-md5" href="#opt-cert-md5" title="#opt-cert-md5">`cert.md5`</a> | MD5 checksum of `cert.pem`, in hexadecimal. | `0644` |
| <a id="opt-cert-json" href="#opt-cert-json" title="#opt-cert-json">`cert.json`</a> | Every value of the other files, the dates and names of the certificate, and the private key. | `0600` |

- A certificate with several domains is written to the directory of each domain.
- The certificate of `*.example.com` is written to the `example.com` directory.
- When one certificate holds `example.com` and another certificate holds only `*.example.com`, the certificate of `example.com` keeps the `example.com` directory and the wildcard certificate is written to `_wildcard.example.com`.
- Traefix writes each file to a temporary file and renames it, and writes `cert.json` last. A program that reads the directory never reads a partial file.
- Traefix writes a directory only when its certificate changes, and doesn't replace a certificate with an older certificate for the same names.
- Traefix doesn't delete the directory of a certificate, including an expired certificate.

The `cert.json` file carries a `traefix` object with the resolver name, the main domain, the other domains and the TLS store of the certificate. Traefix uses the object to load the certificate when the directory is the primary store.

!!! warning "Private keys"

    `privkey.pem` and `cert.json` contain the private key of the certificate. Give access to the directory only to the programs that serve the certificates.

## Directory as the Primary Store

With `primary: dir`, the resolver reads the certificates from the `cert.json` file of each directory, and keeps its ACME account in `<certStore.dir>/_traefix/<resolver>/account.json`, with the `0600` mode.

- **Traefix imports the `storage` file once.** When the directory holds no account and no certificate for the resolver, and the `storage` file holds some, Traefix copies them to the directory. Traefix requests no certificate from the ACME server for the domains of the imported certificates. Traefix doesn't start the resolver when the `storage` file exists and cannot be read.
- **A `cert.json` file without the `traefix` object belongs to the resolver named in its `acme_resolver` field.** The `cert-watcher` tool of traefik-certs writes such files.
- **Traefix ignores the certificates of a resolver that the install configuration doesn't define**, and the directories without a readable `cert.json`. Traefix logs both at startup. To serve such a certificate, declare its files in the [TLS certificates](../../../routing-configuration/http/tls/tls-certificates.md) of the routing configuration.

Don't point `certStore.dir` to a directory that another program writes to, such as the `TCERTS_SAVEDIR` of a running `cert-watcher`.

## Redis as the Primary Store

With `primary: redis`, the resolver keeps its account and certificates in the Valkey or Redis server of the [`cluster`](../../cluster.md) option. Every instance of the cluster with the same resolver name reads the same account and the same certificates.

```yaml tab="File (YAML)"
cluster:
  redis:
    endpoints:
      - "valkey.example.com:6379"

certificatesResolvers:
  myresolver:
    acme:
      email: your-email@example.com
      httpChallenge:
        entryPoint: web
      certStore:
        primary: redis
        # Each instance keeps a copy on its own disk.
        dir: /data/certs
        export:
          - dir
```

```toml tab="File (TOML)"
[cluster.redis]
  endpoints = ["valkey.example.com:6379"]

[certificatesResolvers.myresolver.acme]
  email = "your-email@example.com"
  [certificatesResolvers.myresolver.acme.httpChallenge]
    entryPoint = "web"
  [certificatesResolvers.myresolver.acme.certStore]
    primary = "redis"
    # Each instance keeps a copy on its own disk.
    dir = "/data/certs"
    export = ["dir"]
```

```bash tab="CLI"
--cluster.redis.endpoints=valkey.example.com:6379
--certificatesresolvers.myresolver.acme.email=your-email@example.com
--certificatesresolvers.myresolver.acme.httpchallenge.entrypoint=web
--certificatesresolvers.myresolver.acme.certstore.primary=redis
--certificatesresolvers.myresolver.acme.certstore.dir=/data/certs
--certificatesresolvers.myresolver.acme.certstore.export=dir
```

- **Traefix doesn't start with `primary: redis` and no `cluster` option, or with `tlsChallenge`.** Use `httpChallenge` or `dnsChallenge`.
- **A certificate that an instance stores reaches the other instances within seconds.** Each instance serves the certificate and writes its exports. An instance also compares itself with the server every 15 seconds, to catch up on what it missed.
- **The server keeps the most recent certificate for a set of domains.** A certificate with an earlier expiration date doesn't replace the certificate of the server.
- **Set a `dir` or `file` export on each instance.** An instance that starts while the server is unreachable serves the certificates of its exports. Without an export, the instance has no certificate until the server is back.
- **An instance requests and renews no certificate while the server is unreachable.** The instance keeps serving the certificates it holds, and logs the loss and the return of the server.
- **Each instance offers its local certificates to the server when it starts.** The instance sends the certificates of its exports and of an existing `storage` file. The server takes the certificates it doesn't have and the ones that are more recent than its own, and keeps its own account when it has one. This is how the `storage` file of an instance that joins the cluster reaches the other instances, and how a server that lost its data gets the certificates back.

The instances share the account of the first instance that reaches the Redis server. The certificates that another account obtained are [renewed as new certificates](#renewal-of-imported-certificates).

With the `redis` primary store, the `dir` export also writes the ACME account to `<certStore.dir>/_traefix/<resolver>/account.json`, so that an instance can start from its exports alone.

### Certificate Requests in a Cluster

The instances of a cluster agree on the one that requests each certificate from the ACME server. The other instances receive the certificate from the Redis server.

- **The instance with the highest [`cluster.priority`](../../cluster.md) requests the certificate.** An instance waits `election.step` for each distinct priority above its own, among the running instances that have the resolver, and then tries to take the request. An instance with a lower priority only takes the request when the instances above did not.
- **The instances with the same priority race for the request.** The first one to take the request in the Redis server requests the certificate.
- **An instance with `cluster.priority: 0` requests and renews no certificate.** Such an instance still tells the cluster which certificates its routers need, and serves the certificates that the other instances obtain.
- **Any instance can ask for a certificate.** When an instance finds a router without a certificate, the instance tells the cluster. The elected instance requests the certificate, even when the elected instance doesn't have the router. This covers the hosts where only one instance sees a container.
- **An instance that just started waits `cluster.heartbeatInterval` before its first request.** The instances that start together use this time to see each other.
- **When the elected instance stops during a request, another instance takes the request** after `election.lockTTL` at most.
- **The instances share the failures.** After a failed request, no instance tries again before a delay of 2 minutes, then 15 minutes, 30 minutes, 1 hour, 2 hours, and 6 hours for the following failures. The instance that failed lets the other instances try first at the next attempt. After 14 days of failures, the instances stop requesting the certificate until a change of the routing configuration asks for the certificate again.
- **Renewals follow the same election.** An instance that finds certificates to renew tells the instances with a higher priority, which take the renewals of the certificates they hold in the same version. The instance renews what the instances above it left. A failed renewal is tried again with the same delays, and is not abandoned.
- **The instances log a warning when every instance has the priority 0.** A certificate that a router needs then waits until an instance with a priority above zero starts.
- **No instance takes a request while the Redis server is unreachable.** The certificates that the routers need meanwhile are requested when the Redis server is back.
- **The instances share one ACME account.** The first instance that needs the account registers it, and the other instances wait for it instead of registering their own.

The instances log a warning when they configure the same resolver with a different `caServer`, `email`, `keyType`, challenge, `profile` or `preferredChain`: the certificate then depends on the instance that requests it.

For `dnsChallenge`, set the credentials of the DNS provider on every instance with a priority above zero.

### HTTP-01 Challenge in a Cluster

The ACME server sends its validation request to any of the instances behind the name of a domain. The instance that requests the certificate writes the answer of the challenge to the Redis server, and the instance that receives the validation request reads the answer from there.

- **Configure `httpChallenge` on every instance that receives the requests for the domain.** This includes the instances with `cluster.priority: 0`. An instance without `httpChallenge` doesn't answer the requests for `/.well-known/acme-challenge/`.
- **The instances share the challenges of all their resolvers.** The sharing starts with the [`cluster`](../../cluster.md) option, whatever the `primary` store of a resolver.
- **The answer stays in the Redis server during the validation.** The instance removes the answer when the validation ends. An answer that is not removed expires after 10 minutes.
- **An instance asks the Redis server for the challenges it didn't start.** The instance waits 500 milliseconds at most for the Redis server, and remembers for 5 seconds that the Redis server doesn't know a token.
- **A request without a valid token gets a `404` response without a request to the Redis server.** An ACME token has 22 to 128 letters, digits, hyphens or underscores.
- **Without the Redis server, an instance only answers the challenges it started.** The validation then succeeds when the ACME server reaches this instance.

### Keys in the Redis Server

The names use the default `traefix` prefix of the `cluster` option and a resolver named `myresolver`. The braces keep the keys of a resolver on one node of a Redis Cluster.

| Key | Contains |
|:----|:---------|
| <a id="opt-traefixacmemyresolveraccount" href="#opt-traefixacmemyresolveraccount" title="#opt-traefixacmemyresolveraccount">`traefix:acme:{myresolver}:account`</a> | ACME account of the resolver, with its private key. |
| <a id="opt-traefixacmemyresolvercertid" href="#opt-traefixacmemyresolvercertid" title="#opt-traefixacmemyresolvercertid">`traefix:acme:{myresolver}:cert:<id>`</a> | A certificate, in JSON: domains, TLS store, certificate chain and private key in PEM, and expiration date. The identifier is the SHA-256 hash of the sorted domains. |
| <a id="opt-traefixacmemyresolvercerts" href="#opt-traefixacmemyresolvercerts" title="#opt-traefixacmemyresolvercerts">`traefix:acme:{myresolver}:certs`</a> | Set of the identifiers of the certificates. |
| <a id="opt-traefixacmemyresolverversion" href="#opt-traefixacmemyresolverversion" title="#opt-traefixacmemyresolverversion">`traefix:acme:{myresolver}:version`</a> | Counter that changes with each certificate written. |
| <a id="opt-traefixacmemyresolverwantid" href="#opt-traefixacmemyresolverwantid" title="#opt-traefixacmemyresolverwantid">`traefix:acme:{myresolver}:want:<id>`</a> | A certificate that an instance needs and that no instance stored yet, with its domains. Expires after 24 hours without a new attempt. |
| <a id="opt-traefixacmemyresolverwants" href="#opt-traefixacmemyresolverwants" title="#opt-traefixacmemyresolverwants">`traefix:acme:{myresolver}:wants`</a> | Set of the identifiers of the needed certificates. |
| <a id="opt-traefixacmemyresolverfailid" href="#opt-traefixacmemyresolverfailid" title="#opt-traefixacmemyresolverfailid">`traefix:acme:{myresolver}:fail:<id>`</a> | Last failure of a request or of a renewal: error, instance, number of attempts and time of the next attempt. |
| <a id="opt-traefixlockacmemyresolverid" href="#opt-traefixlockacmemyresolverid" title="#opt-traefixlockacmemyresolverid">`traefix:lock:acme:myresolver:<id>`</a> | Lock of the instance that requests or renews a certificate. Expires after `election.lockTTL` without the instance. |
| <a id="opt-traefixlockacmemyresolveraccount" href="#opt-traefixlockacmemyresolveraccount" title="#opt-traefixlockacmemyresolveraccount">`traefix:lock:acme:myresolver:account`</a> | Lock of the instance that registers the ACME account. |

!!! warning "Private keys"

    The Redis server holds the private keys of the certificates and of the ACME account in clear text. Protect the server with a password or access control lists and with TLS, and don't expose it outside the network of the cluster.

### Copy for traefik-certs

With the `redis` primary store, Traefix also writes each certificate the way the traefik-certs project stores it: the `cert.json` document in the `tcerts:cert:<domain>` key of each domain, and the domain on the `tcerts:events` channel. The `cert-get` and `cert-watcher` tools of traefik-certs read these keys. The names of the keys follow the rules of the [certificate files](#certificate-files), including the `_wildcard.` prefix.

Set `certStore.tcerts.enabled` to `false` when no program reads the copy: the private key of each certificate is then stored once instead of once per domain.

The `cert-watcher` tool connects to a single Redis instance. It cannot read the copy from a Redis Cluster.

## Exports

Traefix brings each export in line with the primary store when the resolver starts, and after each certificate the resolver obtains or renews.

- The `file` export writes the account and the certificates to the `storage` file, in the format Traefik reads. You can use the file as a backup, or to go back to the `storage` file as the primary store.
- The `dir` export writes the certificate files. The account stays in the primary store.
- A failed write to an export is logged and doesn't fail the certificate request: the primary store holds the certificate.

## Renewal of Imported Certificates

A resolver can hold certificates that another ACME account obtained: the certificates of the `storage` file of another instance, or of a directory that `cert-watcher` filled.

When Traefix renews a certificate, it tells the ACME server which certificate the new one replaces. The ACME server only accepts this from the account that obtained the certificate. When the ACME server refuses the request for this reason, Traefix sends the request again as a request for a new certificate, and logs the refusal at the `INFO` level.

Such a renewal doesn't get the rate limit exemption that some ACME servers give to the renewals that name the certificate they replace. The next renewals of the certificate are regular ones, as the account of the resolver obtained it.

## Metrics

| Metric | Labels | Contains |
|:-------|:-------|:---------|
| <a id="opt-traefik-acme-store-writes-total" href="#opt-traefik-acme-store-writes-total" title="#opt-traefik-acme-store-writes-total">`traefik_acme_store_writes_total`</a> | `resolver`, `target`, `result` | Writes of the certificates to the primary store and to the exports. The `target` is `file`, `dir`, `redis`, or `tcerts` for the [copy for traefik-certs](#copy-for-traefik-certs). The `result` is `success` or `failure`. |
| <a id="opt-traefik-acme-cluster-elections-total" href="#opt-traefik-acme-cluster-elections-total" title="#opt-traefik-acme-cluster-elections-total">`traefik_acme_cluster_elections_total`</a> | `resolver`, `result` | Elections of a certificate request the instance took part in: `won`, `lost` (another instance took the request), `skipped` (a request in progress or a recent failure). |
| <a id="opt-traefik-acme-cluster-issuances-total" href="#opt-traefik-acme-cluster-issuances-total" title="#opt-traefik-acme-cluster-issuances-total">`traefik_acme_cluster_issuances_total`</a> | `resolver`, `kind`, `result` | Certificates the instance requested from the ACME server after an election. The `kind` is `obtain` or `renew`, and the `result` is `success` or `failure`. |
| <a id="opt-traefik-acme-cluster-challenge-lookups-total" href="#opt-traefik-acme-cluster-challenge-lookups-total" title="#opt-traefik-acme-cluster-challenge-lookups-total">`traefik_acme_cluster_challenge_lookups_total`</a> | `result` | Requests for an HTTP-01 challenge: `local` (the instance started the challenge), `redis` (another instance started it), `miss` (unknown token), `invalid` (no valid token), `error` (the Redis server didn't answer). |
| <a id="opt-traefik-acme-cluster-config-mismatch" href="#opt-traefik-acme-cluster-config-mismatch" title="#opt-traefik-acme-cluster-config-mismatch">`traefik_acme_cluster_config_mismatch`</a> | `resolver` | `1` when another instance of the cluster configures the resolver differently, `0` otherwise. |

The metrics with `cluster` in their name only have values with the `redis` primary store, or with the [`cluster`](../../cluster.md) option for the challenge requests. Datadog, StatsD and InfluxDB use the same names with dots, for example `traefik.acme.store.writes.total`.
