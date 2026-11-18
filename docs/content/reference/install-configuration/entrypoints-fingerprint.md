---
title: "Traefik EntryPoint Fingerprint Documentation"
description: "Identify each connection of a Traefix entry point, compute its JA3, JA4, JA4T, p0f, Akamai HTTP/2 and JA4H fingerprints, add them to the requests as headers, and export connection and request records to an HTTP endpoint."
---

The `fingerprint` option of an entry point identifies each connection, computes the TCP, TLS, HTTP/2 and HTTP request fingerprints from the bytes the client sent, adds them to the requests as headers, and exports connection and request records to an HTTP endpoint. The option is specific to Traefix, the fork of Traefik Proxy.

Without the `fingerprint` option, the entry point behaves as in Traefik. Without the `export` block, Traefix computes the fingerprints and adds the headers, and sends nothing.

## Configuration Examples

```yaml tab="File (YAML)"
entryPoints:
  websecure:
    address: ":443"
    fingerprint:
      # Prefix of the headers added to the requests. The client headers with this prefix are removed.
      headerPrefix: X-Traefix-
      # JA4T and p0f signature of the TCP SYN, on Linux.
      tcp: true
      export:
        url: http://10.0.0.10:9000/ingest
        headers:
          Authorization: "Bearer my-key"
        rawBytes: true
        fields:
          - User-Agent
          - Accept
          - Accept-Language
          - Accept-Encoding
          - Sec-CH-UA*
          - Sec-Fetch-*
        queueSize: 4096
        queueMaxBytes: 16777216
        batchSize: 64
        flushInterval: 100ms
        sendTimeout: 2s
```

```toml tab="File (TOML)"
[entryPoints.websecure]
  address = ":443"
  [entryPoints.websecure.fingerprint]
    # Prefix of the headers added to the requests. The client headers with this prefix are removed.
    headerPrefix = "X-Traefix-"
    tcp = true
    [entryPoints.websecure.fingerprint.export]
      url = "http://10.0.0.10:9000/ingest"
      rawBytes = true
      fields = ["User-Agent", "Accept", "Accept-Language", "Accept-Encoding", "Sec-CH-UA*", "Sec-Fetch-*"]
      queueSize = 4096
      queueMaxBytes = 16777216
      batchSize = 64
      flushInterval = "100ms"
      sendTimeout = "2s"
      [entryPoints.websecure.fingerprint.export.headers]
        Authorization = "Bearer my-key"
```

```bash tab="CLI"
# The fingerprint option alone, with the default prefix and no export.
--entryPoints.websecure.fingerprint=true
# The prefix and the export.
--entryPoints.websecure.fingerprint.headerPrefix=X-Traefix-
--entryPoints.websecure.fingerprint.tcp=true
--entryPoints.websecure.fingerprint.export.url=http://10.0.0.10:9000/ingest
--entryPoints.websecure.fingerprint.export.headers.Authorization=Bearer my-key
--entryPoints.websecure.fingerprint.export.rawBytes=true
```

The environment variables follow the same names, for example `TRAEFIK_ENTRYPOINTS_WEBSECURE_FINGERPRINT_EXPORT_URL`. The environment variables carry the names of the `export.headers` entries in lowercase, which works because header names are case-insensitive.

## Configuration Options

| Field | Description | Default | Required |
|:------|:------------|:--------|:---------|
| <a id="opt-fingerprint-headerPrefix" href="#opt-fingerprint-headerPrefix" title="#opt-fingerprint-headerPrefix">`fingerprint.headerPrefix`</a> | Prefix of the headers that Traefix adds to the requests, listed in [Request Headers](#request-headers). Traefix removes every client request header whose name starts with this prefix. An empty prefix fails the start of the entry point. Example: `X-Edge-`. | `X-Traefix-` | No |
| <a id="opt-fingerprint-http3HeaderOrder" href="#opt-fingerprint-http3HeaderOrder" title="#opt-fingerprint-http3HeaderOrder">`fingerprint.http3HeaderOrder`</a> | Captures the header order and the `SETTINGS` of the HTTP/3 requests, for their JA4H and the `fieldNames` and `http3.settings` of the records. Traefix enables the qlog events of the QUIC library for that, which costs CPU on every QUIC packet. When unset, the HTTP/3 requests have no JA4H. | `false` | No |
| <a id="opt-fingerprint-tcp" href="#opt-fingerprint-tcp" title="#opt-fingerprint-tcp">`fingerprint.tcp`</a> | Reads the SYN of each TCP connection, which the Linux kernel saves with `TCP_SAVE_SYN`, for the JA4T fingerprint, the p0f signature and the TTL of the client, and the round-trip time that the kernel measured during the handshake. On another operating system, Traefix logs a warning at start and the connections get no TCP fingerprint. Described in [TCP Fingerprint](#tcp-fingerprint). | `false` | No |
| <a id="opt-fingerprint-export-url" href="#opt-fingerprint-export-url" title="#opt-fingerprint-export-url">`fingerprint.export.url`</a> | URL that receives the [batches of records](#exported-records), sent with `POST` requests. The URL must be absolute, with the `http` or `https` scheme, or the start of the entry point fails. | | Yes, with `export` |
| <a id="opt-fingerprint-export-headers" href="#opt-fingerprint-export-headers" title="#opt-fingerprint-export-headers">`fingerprint.export.headers`</a> | Headers sent with each batch, for example an `Authorization` header with the key of the receiving service. Traefix treats the values as secrets and removes them from the configuration it logs. | | No |
| <a id="opt-fingerprint-export-rawBytes" href="#opt-fingerprint-export-rawBytes" title="#opt-fingerprint-export-rawBytes">`fingerprint.export.rawBytes`</a> | Exports the raw bytes of the TLS ClientHello, of the HTTP/2 connection prefix and of the TCP SYN, so that the receiving service can compute its own fingerprints. When unset, the connection records only carry the computed fingerprints. | `false` | No |
| <a id="opt-fingerprint-export-fields" href="#opt-fingerprint-export-fields" title="#opt-fingerprint-export-fields">`fingerprint.export.fields`</a> | Request headers whose values the request records carry, in the received order. A trailing `*` matches the names starting with the given prefix, for example `Sec-Fetch-*`. Each value is cut at 512 bytes. The request records always carry the names of all the headers. | `User-Agent`, `Accept`, `Accept-Language`, `Accept-Encoding`, `Sec-CH-UA*`, `Sec-Fetch-*` | No |
| <a id="opt-fingerprint-export-allowSensitiveFields" href="#opt-fingerprint-export-allowSensitiveFields" title="#opt-fingerprint-export-allowSensitiveFields">`fingerprint.export.allowSensitiveFields`</a> | Allows exporting the values of the `Cookie`, `Authorization`, `Proxy-Authorization` and `Set-Cookie` headers listed in `fields`. When unset, Traefix doesn't export these values, even with a `*` pattern that matches them. When set, Traefix logs a warning at start with the names of the exported credential headers. | `false` | No |
| <a id="opt-fingerprint-export-queueSize" href="#opt-fingerprint-export-queueSize" title="#opt-fingerprint-export-queueSize">`fingerprint.export.queueSize`</a> | Maximum number of records waiting to be sent. When the queue is full, Traefix drops the new record and counts it. | `4096` | No |
| <a id="opt-fingerprint-export-queueMaxBytes" href="#opt-fingerprint-export-queueMaxBytes" title="#opt-fingerprint-export-queueMaxBytes">`fingerprint.export.queueMaxBytes`</a> | Maximum size in bytes of the records waiting to be sent, for example `16777216` for 16 MiB. When the queue is full, Traefix drops the new record and counts it. | `16777216` | No |
| <a id="opt-fingerprint-export-batchSize" href="#opt-fingerprint-export-batchSize" title="#opt-fingerprint-export-batchSize">`fingerprint.export.batchSize`</a> | Number of queued records that triggers sending a batch, between 1 and 1024. A batch carries up to 1024 records and 4 MiB. | `64` | No |
| <a id="opt-fingerprint-export-flushInterval" href="#opt-fingerprint-export-flushInterval" title="#opt-fingerprint-export-flushInterval">`fingerprint.export.flushInterval`</a> | Maximum time a record waits before its batch is sent, whichever of `batchSize` and `flushInterval` comes first. | `100ms` | No |
| <a id="opt-fingerprint-export-sendTimeout" href="#opt-fingerprint-export-sendTimeout" title="#opt-fingerprint-export-sendTimeout">`fingerprint.export.sendTimeout`</a> | Timeout of the request that sends a batch. | `2s` | No |

### Export Behavior

Traefix sends the batches from a goroutine of its own, so that a request never waits on the receiving service. The request side only adds the record to the queue.

- Traefix makes one attempt per batch. A network error, a timeout, or a response with a `4xx` or `5xx` status code drops the batch, and Traefix counts the dropped records by reason.
- Traefix doesn't follow redirections, so that a batch and its headers only go to the configured URL.
- The connection record leaves before the request records of the same connection, unless the queue drops it.
- Traefix logs the dropped records at most once per minute, with the counts by reason.
- When the entry point stops, Traefix sends the queued records once, within the `transport.lifeCycle.graceTimeOut` of the entry point.

## Request Headers

Traefix adds these headers to each request of the entry point, before any router, middleware or service. The names below use the default prefix.

| Header | Contains |
|:-------|:---------|
| <a id="opt-X-Traefix-Src-Ip" href="#opt-X-Traefix-Src-Ip" title="#opt-X-Traefix-Src-Ip">`X-Traefix-Src-Ip`</a> | IP address of the client, from the socket, or from the PROXY protocol header when the entry point accepts it. An IPv4 client shows in dotted form. |
| <a id="opt-X-Traefix-Src-Port" href="#opt-X-Traefix-Src-Port" title="#opt-X-Traefix-Src-Port">`X-Traefix-Src-Port`</a> | Port of the client, from the same source as `X-Traefix-Src-Ip`. |
| <a id="opt-X-Traefix-Dst-Port" href="#opt-X-Traefix-Dst-Port" title="#opt-X-Traefix-Dst-Port">`X-Traefix-Dst-Port`</a> | Port of the entry point that accepted the connection. |
| <a id="opt-X-Traefix-Conn-Id" href="#opt-X-Traefix-Conn-Id" title="#opt-X-Traefix-Conn-Id">`X-Traefix-Conn-Id`</a> | Identifier of the connection, a [ULID](https://github.com/ulid/spec) of 26 characters, ordered by accept time. |
| <a id="opt-X-Traefix-Proto" href="#opt-X-Traefix-Proto" title="#opt-X-Traefix-Proto">`X-Traefix-Proto`</a> | HTTP version of the request: `h1`, `h2` or `h3`. |
| <a id="opt-X-Traefix-Ja3" href="#opt-X-Traefix-Ja3" title="#opt-X-Traefix-Ja3">`X-Traefix-Ja3`</a> | MD5 hash of the [JA3](https://github.com/salesforce/ja3) fingerprint of the TLS ClientHello. |
| <a id="opt-X-Traefix-Ja4" href="#opt-X-Traefix-Ja4" title="#opt-X-Traefix-Ja4">`X-Traefix-Ja4`</a> | [JA4](https://github.com/FoxIO-LLC/ja4) fingerprint of the TLS ClientHello. |
| <a id="opt-X-Traefix-Alpn" href="#opt-X-Traefix-Alpn" title="#opt-X-Traefix-Alpn">`X-Traefix-Alpn`</a> | Application protocols offered in the TLS ClientHello, in the client order, separated by commas, for example `h2,http/1.1`. The bytes of a protocol name that are not visible ASCII, the comma and the percent sign are percent-encoded, for example `%2C`. |
| <a id="opt-X-Traefix-Ja4t" href="#opt-X-Traefix-Ja4t" title="#opt-X-Traefix-Ja4t">`X-Traefix-Ja4t`</a> | [JA4T](https://github.com/FoxIO-LLC/ja4) fingerprint of the TCP SYN of the connection, with the `tcp` option. |
| <a id="opt-X-Traefix-H2" href="#opt-X-Traefix-H2" title="#opt-X-Traefix-H2">`X-Traefix-H2`</a> | Akamai fingerprint of the HTTP/2 connection: settings, window update, priority frames and pseudo-header order. Present on HTTP/2 requests only. |
| <a id="opt-X-Traefix-Ja4h" href="#opt-X-Traefix-Ja4h" title="#opt-X-Traefix-Ja4h">`X-Traefix-Ja4h`</a> | [JA4H](https://github.com/FoxIO-LLC/ja4) fingerprint of the request, computed with the header order and case the client sent, and with the cookie values, which only leave Traefix with `allowSensitiveFields`. |

The headers carry the values of the socket even when the entry point trusts the forwarded headers of its peer. `X-Real-Ip` and `X-Forwarded-For` keep their own rules, set by [`forwardedHeaders`](./entrypoints.md#forwarded-headers).

`X-Traefix-Ja3`, `X-Traefix-Ja4` and `X-Traefix-Alpn` are absent when the connection isn't TLS, and when the ClientHello over TCP is longer than 16 KiB. `X-Traefix-Ja4h` is absent when Traefix couldn't follow the header blocks of the connection, for example after a malformed request.

!!! warning "Use one prefix per installation"

    Each entry point removes the client headers with its own prefix. A router that serves several entry points with different prefixes receives the headers of the other prefixes as the client sent them. Use the same `headerPrefix` on every entry point.

### TCP Fingerprint

With the `tcp` option, Traefix asks the Linux kernel to keep the SYN of each connection the entry point accepts, and reads it once the connection is accepted, which frees it. The SYN gives:

- the JA4T fingerprint, as the FoxIO tool computes it: the window, the kinds of the TCP options, the MSS and the window scale, for example `64240_2-4-8-1-3_1460_7`;
- the p0f v3 signature of the SYN, the `raw_sig` that `p0f` prints, for example `4:57+7:0:1460:mss*44,7:mss,sok,ts,nop,ws:df,id+:0`. Traefix doesn't embed the p0f database, so the operating system guess is left to the receiving service;
- the TTL or hop limit of the SYN, as received.

The SYN is the one of the peer that opened the TCP connection to Traefix. It tells about the client only when nothing between the client and Traefix ends the TCP connection:

- On an entry point with the [`proxyProtocol`](./entrypoints.md#opt-proxyProtocol-trustedIPs) option, the connections get no TCP fingerprint, as the SYN of a connection through the PROXY protocol is the one of the proxy.
- Behind a load balancer or a CDN that opens its own connections, the fingerprint is the one of the load balancer.
- In a Docker container with published ports, the SYN of an IPv4 client crosses the NAT rules of Docker with its options, and keeps its fingerprint. A client that the userland proxy of Docker serves, such as an IPv6 client of an IPv4-only container, shows the SYN of the proxy, with the address of the Docker gateway.
- The kernel doesn't keep the SYN of the connections it accepted with SYN cookies, for example during a SYN flood.

The kernel keeps about 100 bytes per connection until Traefix reads them, at the accept.

### HTTP/3

On HTTP/3 connections, Traefix adds the same headers, with `X-Traefix-Proto` set to `h3`. The connection keeps its identifier when the client migrates it to another network path.

- `X-Traefix-Ja3` and `X-Traefix-Ja4` come from the ClientHello that the QUIC library hands to the TLS handshake. The JA4 starts with `q`.
- `X-Traefix-Ja4h`, and the `fieldNames` of the request records, need the `http3HeaderOrder` option, described in [Configuration Options](#opt-fingerprint-http3HeaderOrder).
- With `export.rawBytes`, Traefix also decrypts the Initial packets of each new connection, with the keys that RFC 9001 derives from their connection identifier, to export the raw ClientHello and the order of the QUIC transport parameters. Traefix then reads the UDP socket through a wrapper, which disables the batch reads and the offloads of the QUIC library on that socket.

## Rule Matchers

On an entry point with the `fingerprint` option, the router rules accept these matchers, in the v3 rule syntax:

| Matcher | Matches |
|:--------|:--------|
| <a id="opt-JA3fp" href="#opt-JA3fp" title="#opt-JA3fp">```JA3(`fp`)```</a> | The MD5 hash of the JA3 fingerprint of the connection, the value of `X-Traefix-Ja3`. |
| <a id="opt-JA4fp" href="#opt-JA4fp" title="#opt-JA4fp">```JA4(`fp`)```</a> | The JA4 fingerprint of the connection, the value of `X-Traefix-Ja4`. |
| <a id="opt-JA4Hfp" href="#opt-JA4Hfp" title="#opt-JA4Hfp">```JA4H(`fp`)```</a> | The JA4H fingerprint of the request, the value of `X-Traefix-Ja4h`. |
| <a id="opt-JA4Tfp" href="#opt-JA4Tfp" title="#opt-JA4Tfp">```JA4T(`fp`)```</a> | The JA4T fingerprint of the connection, the value of `X-Traefix-Ja4t`, with the `tcp` option. |

Each matcher takes one fingerprint. A trailing `*` matches the fingerprints starting with the given prefix, for example ```JA4(`t13d1516h2_*`)```. Combine several values with `||`. A request without the fingerprint, such as a plain HTTP request for `JA4`, matches no value.

A router whose rule uses one of these matchers on an entry point without the `fingerprint` option gets an error and isn't served, as with any invalid rule.

```yaml tab="Structured (YAML)"
http:
  routers:
    bots:
      entryPoints:
        - websecure
      # Sends the curl clients to a dedicated service.
      rule: "Host(`example.com`) && JA4(`t13d4907h2_*`)"
      service: bots
```

To deny or allow requests by fingerprint without a router per fingerprint, use the [`fingerprintList`](../routing-configuration/http/middlewares/fingerprintlist.md) middleware.

## Access Logs and Traces

On the routed requests, Traefix adds the fingerprints to the access log and to the span of the router:

| Access log field | Trace attribute | Contains |
|:-----------------|:----------------|:---------|
| <a id="opt-FingerprintConnId" href="#opt-FingerprintConnId" title="#opt-FingerprintConnId">`FingerprintConnId`</a> | `traefix.connection.id` | Identifier of the connection. |
| <a id="opt-FingerprintJA3" href="#opt-FingerprintJA3" title="#opt-FingerprintJA3">`FingerprintJA3`</a> | `tls.client.ja3` | MD5 hash of the JA3 fingerprint. |
| <a id="opt-FingerprintJA4" href="#opt-FingerprintJA4" title="#opt-FingerprintJA4">`FingerprintJA4`</a> | `traefix.tls.client.ja4` | JA4 fingerprint. |
| <a id="opt-FingerprintH2" href="#opt-FingerprintH2" title="#opt-FingerprintH2">`FingerprintH2`</a> | `traefix.http2.akamai` | Akamai HTTP/2 fingerprint, on HTTP/2 requests. |
| <a id="opt-FingerprintJA4H" href="#opt-FingerprintJA4H" title="#opt-FingerprintJA4H">`FingerprintJA4H`</a> | `traefix.http.request.ja4h` | JA4H fingerprint. |
| <a id="opt-FingerprintJA4T" href="#opt-FingerprintJA4T" title="#opt-FingerprintJA4T">`FingerprintJA4T`</a> | `traefix.tcp.client.ja4t` | JA4T fingerprint, with the `tcp` option. |

The access log shows the fields in the JSON format, and its `fields` options filter them as the other fields. A request that no router matched has no fingerprint field.

A WAF middleware on the router, such as the [`coraza`](../routing-configuration/http/middlewares/coraza.md) middleware, reads the fingerprints from the request headers, for example with the `REQUEST_HEADERS:X-Traefix-Ja4` SecLang variable.

## Exported Records

Each batch is a JSON object sent with `Content-Type: application/json`:

```json
{
  "schema": "traefix.fingerprint/v1",
  "sender": "traefix/fork-point-215-g1a2b3c4d5",
  "entryPoint": "websecure",
  "sentAt": "2026-11-17T21:30:00.123456789Z",
  "records": []
}
```

`sender` carries the version shown by `traefik version`. The published images and the deployment scripts set it to the output of `git describe`, `fork-point-<n>-g<commit>`, which names the commit of the build, and the image carries the same commit in its `org.opencontainers.image.revision` label. The `records` array mixes the two record types, told apart by their `type` field. Traefix omits the fields with no value.

The [JSON Schema of the batches](traefix-fingerprint-v1.schema.json) describes every field below. The Traefix tests check the batches of HTTP/1, HTTP/2 and HTTP/3 connections against it, and the fingerprints against the outputs of tshark, of the FoxIO ja4 tool and of p0f on real client captures.

### Schema Versioning

The `schema` field changes only with an incompatible change of the batches or of the request headers:

- A new field, a new record type, a new header or a new value of an enumerated field keeps `traefix.fingerprint/v1`. A receiving service must ignore the fields and record types it doesn't know.
- Removing or renaming a field or a header, changing the type, the unit or the meaning of a field, or making an optional field required, gives `traefix.fingerprint/v2`. Traefix then states the change in this page before the release that makes it.

### Connection Record

Traefix exports one connection record per connection, at the first request, or when the connection closes without a request. A TLS connection that closes during the handshake still gets its connection record, with the ClientHello data.

| Field | Contains |
|:------|:---------|
| <a id="opt-type" href="#opt-type" title="#opt-type">`type`</a> | `connection`. |
| <a id="opt-connId" href="#opt-connId" title="#opt-connId">`connId`</a> | Identifier of the connection, the value of `X-Traefix-Conn-Id`. |
| <a id="opt-time" href="#opt-time" title="#opt-time">`time`</a> | Accept time, in RFC 3339 format with nanoseconds, UTC. |
| <a id="opt-entryPoint" href="#opt-entryPoint" title="#opt-entryPoint">`entryPoint`</a> | Name of the entry point. |
| <a id="opt-src-dst" href="#opt-src-dst" title="#opt-src-dst">`src`, `dst`</a> | Objects with the `ip` and `port` of the client and of the entry point. |
| <a id="opt-proto" href="#opt-proto" title="#opt-proto">`proto`</a> | HTTP version of the connection: `h1`, `h2` or `h3`. |
| <a id="opt-tls-sni" href="#opt-tls-sni" title="#opt-tls-sni">`tls.sni`</a> | Server name sent by the client. |
| <a id="opt-tls-alpnOffered" href="#opt-tls-alpnOffered" title="#opt-tls-alpnOffered">`tls.alpnOffered`</a> | Application protocols offered by the client, in the client order. |
| <a id="opt-tls-alpn" href="#opt-tls-alpn" title="#opt-tls-alpn">`tls.alpn`</a> | Application protocol negotiated. |
| <a id="opt-tls-version-tls-cipher" href="#opt-tls-version-tls-cipher" title="#opt-tls-version-tls-cipher">`tls.version`, `tls.cipher`</a> | TLS version and cipher suite negotiated, with their IANA names, for example `TLS 1.3` and `TLS_AES_128_GCM_SHA256`. |
| <a id="opt-tls-ja3-tls-ja3String-tls-ja4" href="#opt-tls-ja3-tls-ja3String-tls-ja4" title="#opt-tls-ja3-tls-ja3String-tls-ja4">`tls.ja3`, `tls.ja3String`, `tls.ja4`</a> | JA3 hash, JA3 string before hashing, and JA4. |
| <a id="opt-tls-ja3n-tls-ja3nString" href="#opt-tls-ja3n-tls-ja3nString" title="#opt-tls-ja3n-tls-ja3nString">`tls.ja3n`, `tls.ja3nString`</a> | JA3N hash and string: the JA3 with the extensions sorted, which stays the same when a client shuffles its extensions on each connection, as Chrome does. |
| <a id="opt-tls-ja4r-tls-ja4o" href="#opt-tls-ja4r-tls-ja4o" title="#opt-tls-ja4r-tls-ja4o">`tls.ja4r`, `tls.ja4o`</a> | JA4_r, the JA4 before hashing, with the sorted cipher suites, extensions and signature algorithms in hexadecimal, and JA4_o, the JA4 hashed in the client order, with the server name and ALPN extensions, as the FoxIO tool prints them. With these fields and `ja3String`, a receiving service gets the ClientHello lists without `rawBytes`. |
| <a id="opt-tls-clientHello" href="#opt-tls-clientHello" title="#opt-tls-clientHello">`tls.clientHello`</a> | Raw bytes of the ClientHello, records included, in base64. Present with `rawBytes`. |
| <a id="opt-tls-clientHelloTruncated" href="#opt-tls-clientHelloTruncated" title="#opt-tls-clientHelloTruncated">`tls.clientHelloTruncated`</a> | `true` when the ClientHello is longer than 16 KiB, in which case Traefix keeps the first 16 KiB and computes no TLS fingerprint. |
| <a id="opt-http2-akamai" href="#opt-http2-akamai" title="#opt-http2-akamai">`http2.akamai`</a> | Akamai fingerprint of the HTTP/2 connection, described in [HTTP/2 Fingerprint](#http2-fingerprint). |
| <a id="opt-http2-headersPriority" href="#opt-http2-headersPriority" title="#opt-http2-headersPriority">`http2.headersPriority`</a> | Priority fields of the first `HEADERS` frame, as `stream:exclusive:dependency:weight`, for example `1:1:0:256`. The Akamai fingerprint leaves them out. Absent when the frame has no priority fields. |
| <a id="opt-http2-prefix" href="#opt-http2-prefix" title="#opt-http2-prefix">`http2.prefix`</a> | Raw bytes the client sent after the HTTP/2 preface, up to the end of the first header block, in base64. Present with `rawBytes`. |
| <a id="opt-http2-prefixTruncated" href="#opt-http2-prefixTruncated" title="#opt-http2-prefixTruncated">`http2.prefixTruncated`</a> | `true` when these bytes exceed 16 KiB, in which case Traefix keeps the first 16 KiB. |
| <a id="opt-http3-settings" href="#opt-http3-settings" title="#opt-http3-settings">`http3.settings`</a> | `SETTINGS` of the HTTP/3 connection, as `id:value` separated by `;`, with `http3HeaderOrder`. |
| <a id="opt-tcp-ja4t" href="#opt-tcp-ja4t" title="#opt-tcp-ja4t">`tcp.ja4t`</a> | JA4T fingerprint of the SYN, the value of `X-Traefix-Ja4t`. Present with the `tcp` option, as the whole `tcp` object. |
| <a id="opt-tcp-p0f" href="#opt-tcp-p0f" title="#opt-tcp-p0f">`tcp.p0f`</a> | p0f v3 signature of the SYN. |
| <a id="opt-tcp-ttl" href="#opt-tcp-ttl" title="#opt-tcp-ttl">`tcp.ttl`</a> | TTL, or hop limit, of the SYN as received. |
| <a id="opt-tcp-rttMicros" href="#opt-tcp-rttMicros" title="#opt-tcp-rttMicros">`tcp.rttMicros`</a> | Round-trip time that the kernel measured during the TCP handshake, in microseconds. |
| <a id="opt-tcp-syn" href="#opt-tcp-syn" title="#opt-tcp-syn">`tcp.syn`</a> | Raw bytes of the SYN, from its IP header, in base64. Present with `rawBytes`. |

### HTTP/2 Fingerprint

`http2.akamai` and `X-Traefix-H2` follow the format of the Akamai white paper, as tls.peet.ws computes it: four parts separated by `|`, for example `1:65536;2:0;4:6291456;6:262144|15663105|0|m,a,s,p`.

| Part | Contains |
|:-----|:---------|
| <a id="opt-SETTINGS" href="#opt-SETTINGS" title="#opt-SETTINGS">`SETTINGS`</a> | The settings of the first `SETTINGS` frames, as `id:value` separated by `;`, in the client order. |
| <a id="opt-WINDOW-UPDATE" href="#opt-WINDOW-UPDATE" title="#opt-WINDOW-UPDATE">`WINDOW_UPDATE`</a> | The increment of the first `WINDOW_UPDATE` frame of the connection, or `00` when the client sent none before its first request. |
| <a id="opt-PRIORITY" href="#opt-PRIORITY" title="#opt-PRIORITY">`PRIORITY`</a> | The `PRIORITY` frames, as `stream:exclusive:dependency:weight` separated by `,`, with the weight of RFC 9113, from 1 to 256, or `0` when the client sent none. |
| <a id="opt-pseudo-headers" href="#opt-pseudo-headers" title="#opt-pseudo-headers">pseudo-headers</a> | The order of the pseudo-headers of the first request, by their first letter: `m` for `:method`, `a` for `:authority`, `s` for `:scheme`, `p` for `:path`. |

Splitting the string on `|`, then on `;` or `,`, gives these values without the raw bytes of `http2.prefix`.

### Request Record

Traefix exports one request record per request, once the response ends.

| Field | Contains |
|:------|:---------|
| <a id="opt-type-2" href="#opt-type-2" title="#opt-type-2">`type`</a> | `request`. |
| <a id="opt-connId-2" href="#opt-connId-2" title="#opt-connId-2">`connId`</a> | Identifier of the connection of the request. |
| <a id="opt-start-end" href="#opt-start-end" title="#opt-start-end">`start`, `end`</a> | Start of the request and end of the response, in RFC 3339 format with nanoseconds, UTC. |
| <a id="opt-entryPoint-2" href="#opt-entryPoint-2" title="#opt-entryPoint-2">`entryPoint`</a> | Name of the entry point. |
| <a id="opt-proto-2" href="#opt-proto-2" title="#opt-proto-2">`proto`</a> | `h1`, `h2` or `h3`. |
| <a id="opt-method-authority" href="#opt-method-authority" title="#opt-method-authority">`method`, `authority`</a> | Method and authority of the request. |
| <a id="opt-path" href="#opt-path" title="#opt-path">`path`</a> | Path of the request without the query, cut at 256 bytes. Traefix never exports the query nor the body. |
| <a id="opt-httpVersion" href="#opt-httpVersion" title="#opt-httpVersion">`httpVersion`</a> | Protocol of the request, for example `HTTP/1.1`. |
| <a id="opt-fieldNames" href="#opt-fieldNames" title="#opt-fieldNames">`fieldNames`</a> | Names of the request headers, in the order and case the client sent, without the HTTP/2 pseudo-headers. |
| <a id="opt-fields" href="#opt-fields" title="#opt-fields">`fields`</a> | Array of objects with the `name` and `value` of the headers listed in `export.fields`, in the received order. |
| <a id="opt-cookieNames" href="#opt-cookieNames" title="#opt-cookieNames">`cookieNames`</a> | Names of the cookies, in the received order, without their values. |
| <a id="opt-ja4h" href="#opt-ja4h" title="#opt-ja4h">`ja4h`</a> | JA4H fingerprint, the value of `X-Traefix-Ja4h`. |
| <a id="opt-headersTruncated" href="#opt-headersTruncated" title="#opt-headersTruncated">`headersTruncated`</a> | `true` when the header block exceeds the capture limits: 256 fields, or 8 KiB for an HTTP/1 line. |
| <a id="opt-status" href="#opt-status" title="#opt-status">`status`</a> | Status code of the response. `0` when the connection was taken over, for example by a WebSocket. |
| <a id="opt-router-service" href="#opt-router-service" title="#opt-router-service">`router`, `service`</a> | Router and service that served the request. Absent when no router matched. |

## Metrics

Traefix reports these metrics to every configured metrics backend. The names below are the Prometheus and OpenTelemetry ones. Datadog, StatsD and InfluxDB use the same names with dots, for example `traefik.fingerprint.export.records.total`, and `traefik.fingerprint.export.send.duration` for the sending duration.

The `forwardAuth` middleware counts its fail-opens in `traefik_forwardauth_failopen_total`, described in [ForwardAuth Fail-Open](../routing-configuration/http/middlewares/forwardauth-failopen.md).

| Metric | Labels | Contains |
|:-------|:-------|:---------|
| <a id="opt-traefik-fingerprint-export-records-total" href="#opt-traefik-fingerprint-export-records-total" title="#opt-traefik-fingerprint-export-records-total">`traefik_fingerprint_export_records_total`</a> | `entrypoint`, `type` | Records queued for export. |
| <a id="opt-traefik-fingerprint-export-dropped-total" href="#opt-traefik-fingerprint-export-dropped-total" title="#opt-traefik-fingerprint-export-dropped-total">`traefik_fingerprint_export_dropped_total`</a> | `entrypoint`, `reason` | Records dropped: `queue_full`, `send_error`, `http_4xx`, `http_5xx`, `too_large`. |
| <a id="opt-traefik-fingerprint-export-batches-total" href="#opt-traefik-fingerprint-export-batches-total" title="#opt-traefik-fingerprint-export-batches-total">`traefik_fingerprint_export_batches_total`</a> | `entrypoint`, `code` | Batches sent, by response status code, or `error` when no response came. |
| <a id="opt-traefik-fingerprint-export-queue-records" href="#opt-traefik-fingerprint-export-queue-records" title="#opt-traefik-fingerprint-export-queue-records">`traefik_fingerprint_export_queue_records`</a> | `entrypoint` | Records waiting to be sent. |
| <a id="opt-traefik-fingerprint-export-send-seconds" href="#opt-traefik-fingerprint-export-send-seconds" title="#opt-traefik-fingerprint-export-send-seconds">`traefik_fingerprint_export_send_seconds`</a> | `entrypoint` | Duration of the batch sending. |
| <a id="opt-traefik-fingerprint-capture-truncated-total" href="#opt-traefik-fingerprint-capture-truncated-total" title="#opt-traefik-fingerprint-capture-truncated-total">`traefik_fingerprint_capture_truncated_total`</a> | `entrypoint`, `part` | Captures cut at their size limit: `client_hello`, `h2_prefix`, `headers`. |

## Resource Use

Traefix reads the bytes the server already reads, and never changes them. Measured on Linux (Xeon Gold 6148, HTTPS keep-alive connections, a handler without a backend, so the shares below are upper bounds), the option adds about 14 µs of CPU per HTTP/1 request and 18 µs per HTTP/2 request without `export`, about +21% and +16%, and about twice that with `export`. The cost is per request: reading the header block from the wire, the JA4H, and the added headers. The `tcp` option adds one system call per connection. While the ClientHello and the HTTP/2 prefix are captured, a connection holds up to 16 KiB for each. To follow the header order of every HTTP/2 request, a connection also keeps its own HPACK decoder, bounded by `http2.maxDecoderHeaderTableSize`, for its whole life.
