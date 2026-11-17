---
title: "Traefik ForwardAuth Documentation"
description: "In Traefik Proxy, the HTTP ForwardAuth middleware delegates authentication to an external Service. Read the technical documentation."
---

The `forwardAuth` middleware delegates authentication to an external service.
If the service answers with a 2XX code, access is granted, and the original request is performed.
Otherwise, the response from the authentication server is returned.

## Configuration Example

```yaml tab="Structured (YAML)"
# Forward authentication to example.com
http:
  middlewares:
    test-auth:
      forwardAuth:
        address: "https://example.com/auth"
```

```toml tab="Structured (TOML)"
# Forward authentication to example.com
[http.middlewares]
  [http.middlewares.test-auth.forwardAuth]
    address = "https://example.com/auth"
```

```yaml tab="Labels"
# Forward authentication to example.com
labels:
  - "traefik.http.middlewares.test-auth.forwardauth.address=https://example.com/auth"
```

```json tab="Tags"
// Forward authentication to example.com
{
  "Tags" : [
    "traefik.http.middlewares.test-auth.forwardauth.address=https://example.com/auth"
  ]
}
```

```yaml tab="Kubernetes"
# Forward authentication to example.com
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: test-auth
spec:
  forwardAuth:
    address: https://example.com/auth
```

## Configuration Options

| Field                                                                                                                                          | Description                                                                                                                                                                                                                                                                 | Default | Required |
|:-----------------------------------------------------------------------------------------------------------------------------------------------|:--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:--------|:---------|
| <a id="opt-address" href="#opt-address" title="#opt-address">`address`</a> | Authentication server address. <br />Supports the `http`, `https`, and `h2c` schemes. <br />Mutually exclusive with the `service` option; one of the two is required. <br />More information [here](#address).                                                                                                                                                                | "" | No[^1]      |
| <a id="opt-service" href="#opt-service" title="#opt-service">`service`</a> | Name of a Traefik service to use as the authentication server, instead of `address`. <br />Routing the authentication request through a service enables service discovery, load balancing, health checks and sticky sessions for the authentication server. <br />Mutually exclusive with the `address` and `tls` options. <br />In the Kubernetes CRD, this is a reference to a Kubernetes Service rather than a service name. <br />More information [here](#service).| "" | No[^1]       |
| <a id="opt-path" href="#opt-path" title="#opt-path">`path`</a> | Path (and optional query string) of the authentication request when using the `service` option. <br />Defaults to the root path.                                                                                                                                             | "" | No       |
| <a id="opt-trustForwardHeader" href="#opt-trustForwardHeader" title="#opt-trustForwardHeader">`trustForwardHeader`</a> | Trust all `X-Forwarded-*` headers.                                                                                                                                                                                                                                          <br/>The trustForwardHeader option is deprecated and will be removed in the next major version. <br/>More information [here](#trustforwardheader)| - | No      |
| <a id="opt-authResponseHeaders" href="#opt-authResponseHeaders" title="#opt-authResponseHeaders">`authResponseHeaders`</a> | List of headers to copy from the authentication server response and set on forwarded request, replacing any existing conflicting headers.                                                                                                                                   | [] | No      |
| <a id="opt-authResponseHeadersRegex" href="#opt-authResponseHeadersRegex" title="#opt-authResponseHeadersRegex">`authResponseHeadersRegex`</a> | Regex to match by the headers to copy from the authentication server response and set on forwarded request, after stripping all headers that match the regex.<br /> More information [here](#authresponseheadersregex).                                                     | "" | No      |
| <a id="opt-authRequestHeaders" href="#opt-authRequestHeaders" title="#opt-authRequestHeaders">`authRequestHeaders`</a> | List of the headers to copy from the request to the authentication server. <br /> It allows filtering headers that should not be passed to the authentication server. <br /> If not set or empty, then all request headers are passed.                                      | [] | No      |
| <a id="opt-addAuthCookiesToResponse" href="#opt-addAuthCookiesToResponse" title="#opt-addAuthCookiesToResponse">`addAuthCookiesToResponse`</a> | List of cookies to copy from the authentication server to the response, replacing any existing conflicting cookie from the forwarded response.<br /> Please note that all backend cookies matching the configured list will not be added to the response.<br /> When the authentication server returns no cookies, matching backend cookies are kept, unless `addAuthCookiesToResponseRegex` is set.                   | [] | No      |
| <a id="opt-addAuthCookiesToResponseRegex" href="#opt-addAuthCookiesToResponseRegex" title="#opt-addAuthCookiesToResponseRegex">`addAuthCookiesToResponseRegex`</a> | Regex matching cookie names to copy from the authentication server to the response. All matching backend cookies are removed, even when the authentication server returns no cookies.<br /> More information [here](#addauthcookiestoresponseregex). | "" | No |
| <a id="opt-forwardBody" href="#opt-forwardBody" title="#opt-forwardBody">`forwardBody`</a> | Sets the `forwardBody` option to `true` to send the Body. As body is read inside Traefik before forwarding, this breaks streaming.                                                                                                                                          | false | No      |
| <a id="opt-maxBodySize" href="#opt-maxBodySize" title="#opt-maxBodySize">`maxBodySize`</a> | Set the `maxBodySize` to limit the body size in bytes. If body is bigger than this, it returns a 401 (unauthorized). If left unset, the request body size is unrestricted which can have performance or security implications. <br/>More information [here](#maxbodysize). | -1 | No      |
| <a id="opt-maxResponseBodySize" href="#opt-maxResponseBodySize" title="#opt-maxResponseBodySize">`maxResponseBodySize`</a> | Set the `maxResponseBodySize` to limit the response body size from the authentication server in bytes. If the response body exceeds this limit, it returns a 401 (unauthorized). If left unset, the response body size is unrestricted which can have performance or security implications. <br/>More information [here](#maxresponsebodysize).| - | No      |
| <a id="opt-headerField" href="#opt-headerField" title="#opt-headerField">`headerField`</a> | Defines a header field to store the authenticated user.                                                                                                                                                                                                                     | "" | No      |
| <a id="opt-preserveLocationHeader" href="#opt-preserveLocationHeader" title="#opt-preserveLocationHeader">`preserveLocationHeader`</a> | Defines whether to forward the Location header to the client as is or prefix it with the domain name of the authentication server.                                                                                                                                          | false | No      |
| <a id="opt-preserveRequestMethod" href="#opt-preserveRequestMethod" title="#opt-preserveRequestMethod">`preserveRequestMethod`</a> | Defines whether to preserve the original request method while forwarding the request to the authentication server.                                                                                                                                                          | false | No      |
| <a id="opt-authSigninURL" href="#opt-authSigninURL" title="#opt-authSigninURL">`authSigninURL`</a> | Specifies the URL to redirect to when the authentication server returns 401 Unauthorized.                                                                                                                                                                                   | "" | No      |
| <a id="opt-timeout" href="#opt-timeout" title="#opt-timeout">`timeout`</a> | Defines the maximum duration of the request to the authentication server, including connection time and reading the response body. If the authentication server does not respond in time, the request is rejected with a 500 (Internal Server Error). Setting it to `0` disables the timeout. <br/>More information [here](#timeout). | 30s | No |
| <a id="opt-tls-ca" href="#opt-tls-ca" title="#opt-tls-ca">`tls.ca`</a> | Sets the path to the certificate authority used for the secured connection to the authentication server, it defaults to the system bundle.                                                                                                                                  | "" | No |
| <a id="opt-tls-cert" href="#opt-tls-cert" title="#opt-tls-cert">`tls.cert`</a> | Sets the path to the public certificate used for the secure connection to the authentication server. When using this option, setting the key option is required.                                                                                                            | "" | No |
| <a id="opt-tls-key" href="#opt-tls-key" title="#opt-tls-key">`tls.key`</a> | Sets the path to the private key used for the secure connection to the authentication server. When using this option, setting the `cert` option is required.                                                                                                                | "" | No |
| <a id="opt-tls-caSecret" href="#opt-tls-caSecret" title="#opt-tls-caSecret">`tls.caSecret`</a> | Defines the secret that contains the certificate authority used for the secured connection to the authentication server, it defaults to the system bundle. **This option is only available for the Kubernetes CRD**.                                                        | | No |
| <a id="opt-tls-caConfigMap" href="#opt-tls-caConfigMap" title="#opt-tls-caConfigMap">`tls.caConfigMap`</a> | Name of a ConfigMap in the Middleware namespace containing the certificate authority in `tls.ca` or `ca.crt`. Mutually exclusive with `tls.caSecret`. **This option is only available for the Kubernetes CRD**.<br />More information [here](#ca-certificates-from-a-configmap). | "" | No |
| <a id="opt-tls-certSecret" href="#opt-tls-certSecret" title="#opt-tls-certSecret">`tls.certSecret`</a> | Defines the secret that contains both the private and public certificates used for the secure connection to the authentication server. **This option is only available for the Kubernetes CRD**.                                                                            |  | No |
| <a id="opt-tls-insecureSkipVerify" href="#opt-tls-insecureSkipVerify" title="#opt-tls-insecureSkipVerify">`tls.insecureSkipVerify`</a> | During TLS connections, if this option is set to `true`, the authentication server will accept any certificate presented by the server regardless of the host names it covers.                                                                                              | false | No |

[^1]: Exactly one of `address` or `service` must be set.

### address

The `address` option accepts the `http`, `https`, and `h2c` schemes.

Use the `h2c` scheme to reach an authentication server that speaks unencrypted HTTP/2 (cleartext, with prior knowledge):

```yaml tab="Structured (YAML)"
http:
  middlewares:
    test-auth:
      forwardAuth:
        address: "h2c://auth-server:8080/auth"
```

```toml tab="Structured (TOML)"
[http.middlewares]
  [http.middlewares.test-auth.forwardAuth]
    address = "h2c://auth-server:8080/auth"
```

```yaml tab="Labels"
labels:
  - "traefik.http.middlewares.test-auth.forwardauth.address=h2c://auth-server:8080/auth"
```

```yaml tab="Kubernetes"
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: test-auth
spec:
  forwardAuth:
    address: h2c://auth-server:8080/auth
```

With `http`, a burst of concurrent authentication requests opens as many connections to the authentication server.
With `h2c`, the requests are multiplexed on a few HTTP/2 connections, which avoids exhausting the ephemeral ports at high request rates.

The `h2c` traffic is not encrypted: use it only on a trusted network.
The `h2c` scheme and the `tls` option are mutually exclusive.

### service

When `service` is set, the authentication request is routed through the named Traefik service
instead of being sent directly to an `address`. This makes the authentication server benefit from
the same load balancing, service discovery, health checks and sticky sessions as any other service.

The authentication request is sent to the `path` of the service (the root path by default).
The connection to the authentication server, including its TLS configuration, is handled by the
service and its `ServersTransport`; for this reason the `service` and `tls` options are mutually
exclusive.

```yaml tab="File (YAML)"
http:
  middlewares:
    test-auth:
      forwardAuth:
        service: auth-service
        path: "/verify"

  services:
    auth-service:
      loadBalancer:
        servers:
          - url: "http://auth-1:9000"
          - url: "http://auth-2:9000"
```

```yaml tab="Kubernetes"
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: test-auth
spec:
  forwardAuth:
    path: /verify
    service:
      name: auth-service
      port: 9000
```

In the Kubernetes CRD, `service` is a reference to a Kubernetes Service (`name`, `namespace`,
`port`, `kind`), following the same shape as the [`errors`](./errorpages.md) middleware, rather
than the name of a service defined by another provider.

!!! info "ServersTransport"

    To customize how Traefik connects to the authentication service (for example, to configure TLS
    to the backend), set a [`serversTransport`](../../kubernetes/crd/http/serverstransport.md) on
    the middleware's `service`.

!!! info "Relative Location headers"

    With `address`, a relative `Location` header returned by the authentication server is resolved
    against the authentication server URL when `preserveLocationHeader` is `false` (the default).
    With `service` there is no such URL to resolve against, as the service backends are internal
    addresses the client cannot reach, so a relative `Location` is forwarded to the client
    unchanged and resolved by the client against the original request.

    This only matters for an authentication server that redirects (for example to a login page)
    instead of answering `2xx`/`401` directly. A relative redirect resolved against the original
    request means the client's next request lands back on Traefik, under the same host — so, for
    that redirect to actually resolve to something, a router must also expose the target path.
    The `service` reused by `forwardAuth` can be reused by that router as well; it is an ordinary
    service like any other, referenced from wherever it is needed:

    ```yaml
    http:
      services:
        auth-service:
          loadBalancer:
            servers:
              - url: "http://auth-1:9000"
              - url: "http://auth-2:9000"

      middlewares:
        test-auth:
          forwardAuth:
            service: auth-service
            path: /verify

      routers:
        login-page:
          rule: "Host(`app.example.com`) && PathPrefix(`/login`)"
          service: auth-service
    ```

    An authentication server that only ever answers `2xx` or `401` — for example a bearer-token or
    API-key check — never sends a `Location` header, so none of this applies.

### authResponseHeadersRegex

It allows partial matching of the regular expression against the header key.

The start of string (`^`) and end of string (`$`) anchors should be used to ensure a full match against the header key.

Regular expressions and replacements can be tested using online tools such as [Go Playground](https://go.dev/play/p/mWU9p-wk2ru) or the [Regex101](https://regex101.com/r/58sIgx/2/substitution).

### addAuthCookiesToResponseRegex

The `addAuthCookiesToResponseRegex` option selects authentication response cookies by name.
A cookie is copied if its name matches either this regex or the `addAuthCookiesToResponse` list.
All backend cookies whose names match the regex are removed, even when the authentication server returns no cookies.

Regular expressions allow partial matches. Use the `^` and `$` anchors to match a complete cookie name.

```yaml tab="Structured (YAML)"
http:
  middlewares:
    test-auth:
      forwardAuth:
        address: "https://example.com/auth"
        addAuthCookiesToResponseRegex: "^auth_cookie"
```

```toml tab="Structured (TOML)"
[http.middlewares.test-auth.forwardAuth]
  address = "https://example.com/auth"
  addAuthCookiesToResponseRegex = "^auth_cookie"
```

```yaml tab="Labels"
labels:
  - "traefik.http.middlewares.test-auth.forwardauth.addauthcookiestoresponseregex=^auth_cookie"
```

```json tab="Tags"
{
  "Tags": [
    "traefik.http.middlewares.test-auth.forwardauth.addauthcookiestoresponseregex=^auth_cookie"
  ]
}
```

```yaml tab="Kubernetes"
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: test-auth
spec:
  forwardAuth:
    address: https://example.com/auth
    addAuthCookiesToResponseRegex: "^auth_cookie"
```

### maxBodySize

The `maxBodySize` option controls the maximum size of request bodies that will be forwarded to the authentication server.

**⚠️ Important Security Consideration**

By default, `maxBodySize` is not set (value: -1), which means request body size is unlimited. This can have significant security and performance implications:

- **Security Risk**: Attackers can send extremely large request bodies, potentially causing DoS attacks or memory exhaustion
- **Performance Impact**: Large request bodies consume memory and processing resources, affecting overall system performance
- **Resource Consumption**: Unlimited body size can lead to unexpected resource usage patterns

**Recommended Configuration**

It is strongly recommended to set an appropriate `maxBodySize` value for your use case:

```yaml
# For most web applications (1MB limit)
maxBodySize: 1048576  # 1MB in bytes

# For API endpoints expecting larger payloads (10MB limit)  
maxBodySize: 10485760  # 10MB in bytes

# For file upload authentication (100MB limit)
maxBodySize: 104857600  # 100MB in bytes
```

**Guidelines for Setting maxBodySize**

- **Web Forms**: 1-5MB is typically sufficient for most form submissions
- **API Endpoints**: Consider your largest expected JSON/XML payload + buffer
- **File Uploads**: Set based on your maximum expected file size
- **High-Traffic Services**: Use smaller limits to prevent resource exhaustion

### maxResponseBodySize

The `maxResponseBodySize` option defines the maximum allowed response body size in bytes from the authentication server.
If the response body exceeds the configured limit, the request is rejected with a 401 (Unauthorized) status.
If left unset, the request body size is unrestricted which can have performance or security implications.

!!! warning

    It is strongly recommended to set this option to a suitable value.
    Not setting it (or setting it to `-1`) allows unlimited response body sizes which can lead to DoS attacks and memory exhaustion.

### timeout

The `timeout` option defines how long Traefik waits for the authentication server to answer,
from establishing the connection to reading the full response body.

The value can be provided as a number of seconds or as a valid duration format (see [time.ParseDuration](https://pkg.go.dev/time#ParseDuration)).

When the timeout is reached, the request is not forwarded to the service and Traefik responds with a 500 (Internal Server Error).
With the `service` option, the timeout applies to the request sent through the service, and Traefik responds with the error status of the service, usually a 504 (Gateway Timeout).

Setting `timeout` to `0` disables the timeout,
which means a slow or unresponsive authentication server can hold requests indefinitely.

```yaml
# Allow the authentication server up to 2 minutes to answer
timeout: 2m
```

### CA Certificates From a ConfigMap

Set `tls.caConfigMap` to load CA certificates from a ConfigMap in the same namespace as the Middleware.
The ConfigMap must contain a `tls.ca` or `ca.crt` key; `tls.ca` takes precedence when both keys are present.
Set only one of `tls.caConfigMap` and `tls.caSecret`.
If the selected ConfigMap is missing or has neither key, Traefik rejects the Middleware configuration.

```yaml tab="Kubernetes"
apiVersion: v1
kind: ConfigMap
metadata:
  name: auth-ca
  namespace: default
data:
  ca.crt: |
    -----BEGIN CERTIFICATE-----
    # Replace with your CA certificate.
    -----END CERTIFICATE-----
---
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: test-auth
  namespace: default
spec:
  forwardAuth:
    address: https://example.com/auth
    tls:
      caConfigMap: auth-ca
```

!!! warning

    The CA decides which authentication servers Traefik trusts.
    Make sure that the principals allowed to update the ConfigMap are the ones allowed to update the Middleware.

### trustForwardHeader

!!! warning

    `trustForwardHeader` option is deprecated and will be removed in the next major version.
    
    Configure the trusted IPs at the [EntryPoint level](../../../install-configuration/entrypoints.md#forwarded-headers) using `forwardedHeaders.trustedIPs`, 
    and set `trustForwardHeader` to `true` on this middleware.

    With this setup, the EntryPoint is responsible for sanitizing incoming `X-Forwarded-*` headers:
    it strips any such headers sent by untrusted clients and only preserves those coming from trusted upstream proxies.
    By the time the ForwardAuth middleware processes the request, all `X-Forwarded-*` headers are guaranteed to be trustworthy,
    including those intentionally added by other middlewares in the chain — for example, the `X-Forwarded-Prefix` header set by the [StripPrefix](stripprefix.md) middleware.

    If `trustForwardHeader` is not explicitly set, Traefik will log a warning at startup and use a legacy behavior where some `X-Forwarded-*` headers (e.g. `X-Forwarded-For`, `X-Forwarded-Proto`) are removed but others (e.g. `X-Forwarded-Prefix`) are forwarded untouched.
    To silence this warning, explicitly set `trustForwardHeader` to `true` or `false`.

Set the `trustForwardHeader` option to `true` to trust all `X-Forwarded-*` headers.

## Forward-Request Headers

The following request properties are provided to the forward-auth target endpoint as `X-Forwarded-` headers.

| Property          | Forward-Request Header |
|-------------------|------------------------|
| <a id="opt-HTTP-Method" href="#opt-HTTP-Method" title="#opt-HTTP-Method">HTTP Method</a> | `X-Forwarded-Method`     |
| <a id="opt-Protocol" href="#opt-Protocol" title="#opt-Protocol">Protocol</a> | `X-Forwarded-Proto`      |
| <a id="opt-Host" href="#opt-Host" title="#opt-Host">Host</a> | `X-Forwarded-Host`       |
| <a id="opt-Request-URI" href="#opt-Request-URI" title="#opt-Request-URI">Request URI</a> | `X-Forwarded-Uri`        |
| <a id="opt-Source-IP-Address" href="#opt-Source-IP-Address" title="#opt-Source-IP-Address">Source IP-Address</a> | `X-Forwarded-For`        |

{% include-markdown "includes/traefik-for-business-applications.md" %}
