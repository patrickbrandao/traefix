---
title: "Traefik ACME Certificate Store Documentation"
description: "Choose where a Traefix ACME certificate resolver keeps its account and certificates: the acme.json storage file, one directory of PEM files per domain, or both."
---

The `certStore` option of an ACME certificate resolver sets where the resolver keeps its account and certificates. You can keep them in the `storage` file, in one directory of PEM files per domain, or in both. The option is specific to Traefix, the fork of Traefik Proxy.

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
| <a id="opt-certStore-primary" href="#opt-certStore-primary" title="#opt-certStore-primary">`certStore.primary`</a> | Store the resolver reads its account and certificates from: `file` for the `storage` file, or `dir` for the directory of `certStore.dir`. | `file` | No |
| <a id="opt-certStore-dir" href="#opt-certStore-dir" title="#opt-certStore-dir">`certStore.dir`</a> | Directory that holds one sub-directory of [certificate files](#certificate-files) per domain. Traefix creates the directory on the first write. | "" | Yes, when `dir` is the primary store or an export |
| <a id="opt-certStore-export" href="#opt-certStore-export" title="#opt-certStore-export">`certStore.export`</a> | Stores that receive a copy of the primary store: `dir`, `file`. The primary store cannot be an export. | [] | No |

Traefix doesn't start when a `certStore` option names an unknown store, lists a store twice, or uses `dir` without `certStore.dir`.

### Store Combinations

| Result | `primary` | `export` |
|:-------|:----------|:---------|
| <a id="opt-The-storage-file-only-as-in-Traefik" href="#opt-The-storage-file-only-as-in-Traefik" title="#opt-The-storage-file-only-as-in-Traefik">The `storage` file only, as in Traefik</a> | `file` | |
| <a id="opt-The-storage-file-copied-to-certificate-files" href="#opt-The-storage-file-copied-to-certificate-files" title="#opt-The-storage-file-copied-to-certificate-files">The `storage` file, copied to certificate files</a> | `file` | `dir` |
| <a id="opt-Certificate-files-only" href="#opt-Certificate-files-only" title="#opt-Certificate-files-only">Certificate files only</a> | `dir` | |
| <a id="opt-Certificate-files-copied-to-the-storage-file" href="#opt-Certificate-files-copied-to-the-storage-file" title="#opt-Certificate-files-copied-to-the-storage-file">Certificate files, copied to the `storage` file</a> | `dir` | `file` |

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

## Exports

Traefix brings each export in line with the primary store when the resolver starts, and after each certificate the resolver obtains or renews.

- The `file` export writes the account and the certificates to the `storage` file, in the format Traefik reads. You can use the file as a backup, or to go back to the `storage` file as the primary store.
- The `dir` export writes the certificate files. The account stays in the primary store.
- A failed write to an export is logged and doesn't fail the certificate request: the primary store holds the certificate.
