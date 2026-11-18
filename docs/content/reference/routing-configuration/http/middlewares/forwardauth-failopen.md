---
title: "Traefik ForwardAuth Fail-Open Documentation"
description: "Let the requests reach the backend when the forwardAuth authentication server fails, and send fixed secret headers to the authentication server, with the Traefix options of the forwardAuth middleware."
---

The `failOpen` and `requestHeaders` options of the [`forwardAuth`](./forwardauth.md) middleware let you query a verdict server, such as a bot detection service, without its failures taking the site down. The options are specific to Traefix, the fork of Traefik Proxy.

Without `failOpen`, the middleware behaves as in Traefik: an error or a timeout of the authentication server fails the request with a `500`, and a non-`2XX` response goes back to the client.

## Configuration Examples

```yaml tab="Structured (YAML)"
http:
  middlewares:
    bot:
      forwardAuth:
        address: http://10.249.255.254:9003/auth
        # The default timeout is 30s: set a short one for a verdict server.
        timeout: 200ms
        # Forward only these client headers, so that the client Cookie and Authorization stay out of the authentication request.
        authRequestHeaders:
          - User-Agent
          - Accept
          - Accept-Language
        requestHeaders:
          Authorization: "Bearer my-key"
        authResponseHeaders:
          - X-Fs-Verdict
          - X-Fs-Score
        failOpen:
          denyRequiresResponseHeader: X-Fs-Verdict
          header:
            name: X-Fs-Verdict
            value: unavailable
```

```toml tab="Structured (TOML)"
[http.middlewares.bot.forwardAuth]
  address = "http://10.249.255.254:9003/auth"
  # The default timeout is 30s: set a short one for a verdict server.
  timeout = "200ms"
  # Forward only these client headers, so that the client Cookie and Authorization stay out of the authentication request.
  authRequestHeaders = ["User-Agent", "Accept", "Accept-Language"]
  authResponseHeaders = ["X-Fs-Verdict", "X-Fs-Score"]
  [http.middlewares.bot.forwardAuth.requestHeaders]
    Authorization = "Bearer my-key"
  [http.middlewares.bot.forwardAuth.failOpen]
    denyRequiresResponseHeader = "X-Fs-Verdict"
    [http.middlewares.bot.forwardAuth.failOpen.header]
      name = "X-Fs-Verdict"
      value = "unavailable"
```

```yaml tab="Labels"
labels:
  - "traefik.http.middlewares.bot.forwardauth.address=http://10.249.255.254:9003/auth"
  - "traefik.http.middlewares.bot.forwardauth.timeout=200ms"
  - "traefik.http.middlewares.bot.forwardauth.authRequestHeaders=User-Agent,Accept,Accept-Language"
  - "traefik.http.middlewares.bot.forwardauth.requestHeaders.Authorization=Bearer my-key"
  - "traefik.http.middlewares.bot.forwardauth.authResponseHeaders=X-Fs-Verdict,X-Fs-Score"
  - "traefik.http.middlewares.bot.forwardauth.failOpen.denyRequiresResponseHeader=X-Fs-Verdict"
  - "traefik.http.middlewares.bot.forwardauth.failOpen.header.name=X-Fs-Verdict"
  - "traefik.http.middlewares.bot.forwardauth.failOpen.header.value=unavailable"
```

The Kubernetes `Middleware` resource doesn't support these options yet.

## Configuration Options

| Field | Description | Default | Required |
|:------|:------------|:--------|:---------|
| <a id="opt-requestHeaders" href="#opt-requestHeaders" title="#opt-requestHeaders">`requestHeaders`</a> | Headers set on the authentication request, replacing the client headers of the same name, for example an `Authorization` header with the key of the authentication server. Traefix treats the values as secrets: the API and the logs show them redacted. With an `http` or `h2c` address whose host is neither a loopback nor a private IP address, the middleware fails to load, so that a key doesn't cross a public network in clear text. With the `service` option, Traefix doesn't check the transport. | | No |
| <a id="opt-failOpen" href="#opt-failOpen" title="#opt-failOpen">`failOpen`</a> | Lets the request reach the backend when the authentication server gives no verdict, as described in [Fail-Open Behavior](#fail-open-behavior). An empty block, `failOpen: {}`, enables it with no option. When unset, the failures of the authentication server fail the request. | | No |
| <a id="opt-failOpen-denyRequiresResponseHeader" href="#opt-failOpen-denyRequiresResponseHeader" title="#opt-failOpen-denyRequiresResponseHeader">`failOpen.denyRequiresResponseHeader`</a> | Header that a `4XX` response of the authentication server must carry to deny the request, for example `X-Fs-Verdict`. A `4XX` response without it, such as a `401` for an invalid key or a `429` rate limit of the authentication server, lets the request through. When unset, every `4XX` response denies the request. | | No |
| <a id="opt-failOpen-header-name" href="#opt-failOpen-header-name" title="#opt-failOpen-header-name">`failOpen.header.name`</a> | Name of the header set on a request let through after a failure, for example `X-Fs-Verdict`. When unset, Traefix sets no header. | | No |
| <a id="opt-failOpen-header-value" href="#opt-failOpen-header-value" title="#opt-failOpen-header-value">`failOpen.header.value`</a> | Value of that header, for example `unavailable`. | | No |

!!! warning "List the forwarded headers in `authRequestHeaders`"

    With an empty `authRequestHeaders`, the authentication request gets every client header, `Cookie` and `Authorization` included. When `requestHeaders` sets `Authorization` and `authRequestHeaders` is empty, the middleware logs a warning at load, because the fixed value then replaces the client one in the authentication request.

The [entry point fingerprint](../../../install-configuration/entrypoints-fingerprint.md) headers of the request always reach the authentication server, even when `authRequestHeaders` doesn't list them.

## Fail-Open Behavior

With `failOpen`, the request reaches the backend in these cases, which the `reason` label of the metric names:

| Case | `reason` |
|:-----|:---------|
| <a id="opt-The-authentication-request-exceeds-timeout" href="#opt-The-authentication-request-exceeds-timeout" title="#opt-The-authentication-request-exceeds-timeout">The authentication request exceeds `timeout`.</a> | `timeout` |
| <a id="opt-The-authentication-server-is-unreachable-or-the-connection-fails-while-reading-its-response" href="#opt-The-authentication-server-is-unreachable-or-the-connection-fails-while-reading-its-response" title="#opt-The-authentication-server-is-unreachable-or-the-connection-fails-while-reading-its-response">The authentication server is unreachable, or the connection fails while reading its response.</a> | `network` |
| <a id="opt-The-authentication-server-answers-with-a-5XX-status-code" href="#opt-The-authentication-server-answers-with-a-5XX-status-code" title="#opt-The-authentication-server-answers-with-a-5XX-status-code">The authentication server answers with a `5XX` status code.</a> | `http_5xx` |
| <a id="opt-The-authentication-server-answers-with-a-4XX-status-code-without-the-denyRequiresResponseHeader-header" href="#opt-The-authentication-server-answers-with-a-4XX-status-code-without-the-denyRequiresResponseHeader-header" title="#opt-The-authentication-server-answers-with-a-4XX-status-code-without-the-denyRequiresResponseHeader-header">The authentication server answers with a `4XX` status code without the `denyRequiresResponseHeader` header.</a> | `http_4xx` |

When Traefix lets a request through, it removes the client headers listed in `authResponseHeaders` and matched by `authResponseHeadersRegex`, so that a client can't forge the verdict, then sets `failOpen.header`.

A `2XX` response grants access, and a `4XX` response with the `denyRequiresResponseHeader` header goes back to the client, as in Traefik. A request whose client goes away is not counted as a failure.

The worst case adds `timeout` to the request. Set `timeout` explicitly: `failOpen` doesn't change its default of `30s`.

Traefix counts each request let through in the `traefik_forwardauth_failopen_total` metric, with the `middleware` and `reason` labels, and logs at most one line per minute and middleware, with the count of the requests let through since the previous line.
