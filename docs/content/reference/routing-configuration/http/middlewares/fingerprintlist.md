---
title: "Traefik FingerprintList Documentation"
description: "Deny or allow requests by their JA3, JA4, JA4H or Akamai HTTP/2 fingerprint with the Traefix fingerprintList middleware."
---

The `fingerprintList` middleware denies or allows requests by the fingerprints that the [entry point fingerprint](../../../install-configuration/entrypoints-fingerprint.md) computes. The middleware is specific to Traefix, the fork of Traefik Proxy.

## Configuration Examples

```yaml tab="Structured (YAML)"
# Rejects the curl clients with a 429.
http:
  middlewares:
    no-curl:
      fingerprintList:
        mode: deny
        ja4:
          - "t13d4907h2_*"
        ja4h:
          - "ge11nn03*"
        rejectStatusCode: 429
```

```toml tab="Structured (TOML)"
# Rejects the curl clients with a 429.
[http.middlewares.no-curl.fingerprintList]
  mode = "deny"
  ja4 = ["t13d4907h2_*"]
  ja4h = ["ge11nn03*"]
  rejectStatusCode = 429
```

```yaml tab="Labels"
# Rejects the curl clients with a 429.
labels:
  - "traefik.http.middlewares.no-curl.fingerprintlist.mode=deny"
  - "traefik.http.middlewares.no-curl.fingerprintlist.ja4=t13d4907h2_*"
  - "traefik.http.middlewares.no-curl.fingerprintlist.ja4h=ge11nn03*"
  - "traefik.http.middlewares.no-curl.fingerprintlist.rejectStatusCode=429"
```

The Kubernetes `Middleware` resource doesn't support this middleware yet.

## Configuration Options

| Field | Description | Default | Required |
|:------|:------------|:--------|:---------|
| <a id="opt-mode" href="#opt-mode" title="#opt-mode">`mode`</a> | `deny` rejects the requests with a listed fingerprint. `allow` rejects the requests without a listed fingerprint, including the requests whose fingerprint wasn't computed. Any other value fails the middleware at load. | | Yes |
| <a id="opt-ja3" href="#opt-ja3" title="#opt-ja3">`ja3`</a> | MD5 hashes of JA3 fingerprints. | | No |
| <a id="opt-ja4" href="#opt-ja4" title="#opt-ja4">`ja4`</a> | JA4 fingerprints, for example `t13d1516h2_8daaf6152771_e5627efa2ab1`. | | No |
| <a id="opt-ja4h" href="#opt-ja4h" title="#opt-ja4h">`ja4h`</a> | JA4H fingerprints. | | No |
| <a id="opt-http2" href="#opt-http2" title="#opt-http2">`http2`</a> | Akamai HTTP/2 fingerprints, checked on HTTP/2 requests only. | | No |
| <a id="opt-rejectStatusCode" href="#opt-rejectStatusCode" title="#opt-rejectStatusCode">`rejectStatusCode`</a> | HTTP status code of the rejected requests. | `403` | No |

In each list, a trailing `*` matches the fingerprints starting with the given prefix, for example `t13d1516h2_*`. A lone `*` fails the middleware at load, and so does a configuration without any listed fingerprint.

A request is listed when one of its fingerprints matches its list. A fingerprint the entry point didn't compute matches nothing: JA3 and JA4 over plain HTTP, JA4H over HTTP/3 without the `http3HeaderOrder` option, and every fingerprint on an entry point without the `fingerprint` option. In `allow` mode, such requests are rejected.
