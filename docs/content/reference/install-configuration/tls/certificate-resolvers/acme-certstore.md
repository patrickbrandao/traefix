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

!!! warning "The instances don't coordinate their certificate requests yet"

    In this version, the instances share the certificates they hold, and each instance still requests a certificate on its own for a router that has none. Several instances that see the same router request or renew the same certificate, and an instance cannot answer the HTTP-01 challenge that another instance started. Until the instances coordinate their requests, don't use the `redis` primary store with several instances in production.

- **Traefix doesn't start with `primary: redis` and no `cluster` option, or with `tlsChallenge`.** Use `httpChallenge` or `dnsChallenge`.
- **A certificate that an instance stores reaches the other instances within seconds.** Each instance serves the certificate and writes its exports. An instance also compares itself with the server every 15 seconds, to catch up on what it missed.
- **The server keeps the most recent certificate for a set of domains.** A certificate with an earlier expiration date doesn't replace the certificate of the server.
- **Set a `dir` or `file` export on each instance.** An instance that starts while the server is unreachable serves the certificates of its exports. Without an export, the instance has no certificate until the server is back.
- **An instance requests and renews no certificate while the server is unreachable.** The instance keeps serving the certificates it holds, and logs the loss and the return of the server.
- **Each instance offers its local certificates to the server when it starts.** The instance sends the certificates of its exports and of an existing `storage` file. The server takes the certificates it doesn't have and the ones that are more recent than its own, and keeps its own account when it has one. This is how the `storage` file of an instance that joins the cluster reaches the other instances, and how a server that lost its data gets the certificates back.

With the `redis` primary store, the `dir` export also writes the ACME account to `<certStore.dir>/_traefix/<resolver>/account.json`, so that an instance can start from its exports alone.

### Keys in the Redis Server

The names use the default `traefix` prefix of the `cluster` option and a resolver named `myresolver`. The braces keep the keys of a resolver on one node of a Redis Cluster.

| Key | Contains |
|:----|:---------|
| <a id="opt-traefixacmemyresolveraccount" href="#opt-traefixacmemyresolveraccount" title="#opt-traefixacmemyresolveraccount">`traefix:acme:{myresolver}:account`</a> | ACME account of the resolver, with its private key. |
| <a id="opt-traefixacmemyresolvercertid" href="#opt-traefixacmemyresolvercertid" title="#opt-traefixacmemyresolvercertid">`traefix:acme:{myresolver}:cert:<id>`</a> | A certificate, in JSON: domains, TLS store, certificate chain and private key in PEM, and expiration date. The identifier is the SHA-256 hash of the sorted domains. |
| <a id="opt-traefixacmemyresolvercerts" href="#opt-traefixacmemyresolvercerts" title="#opt-traefixacmemyresolvercerts">`traefix:acme:{myresolver}:certs`</a> | Set of the identifiers of the certificates. |
| <a id="opt-traefixacmemyresolverversion" href="#opt-traefixacmemyresolverversion" title="#opt-traefixacmemyresolverversion">`traefix:acme:{myresolver}:version`</a> | Counter that changes with each certificate written. |

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
