---
title: "Traefik EntryPoint Fingerprint Documentation"
description: "Identify each connection of a Traefix entry point, compute its JA3, JA4, Akamai HTTP/2 and JA4H fingerprints, add them to the requests as headers, and export connection and request records to an HTTP endpoint."
---

The `fingerprint` option of an entry point identifies each connection, computes the TLS, HTTP/2 and HTTP request fingerprints from the bytes the client sent, adds them to the requests as headers, and exports connection and request records to an HTTP endpoint. The option is specific to Traefix, the fork of Traefik Proxy.

Without the `fingerprint` option, the entry point behaves as in Traefik. Without the `export` block, Traefix computes the fingerprints and adds the headers, and sends nothing.

## Configuration Examples

```yaml tab="File (YAML)"
entryPoints:
  websecure:
    address: ":443"
    fingerprint:
      # Prefix of the headers added to the requests. The client headers with this prefix are removed.
      headerPrefix: X-Traefix-
      export:
        url: http://10.249.255.254:9003/ingest
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
    [entryPoints.websecure.fingerprint.export]
      url = "http://10.249.255.254:9003/ingest"
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
--entryPoints.websecure.fingerprint.export.url=http://10.249.255.254:9003/ingest
--entryPoints.websecure.fingerprint.export.headers.Authorization=Bearer my-key
--entryPoints.websecure.fingerprint.export.rawBytes=true
```

The environment variables follow the same names, for example `TRAEFIK_ENTRYPOINTS_WEBSECURE_FINGERPRINT_EXPORT_URL`. The environment variables carry the names of the `export.headers` entries in lowercase, which works because header names are case-insensitive.

## Configuration Options

| Field | Description | Default | Required |
|:------|:------------|:--------|:---------|
| <a id="opt-fingerprint-headerPrefix" href="#opt-fingerprint-headerPrefix" title="#opt-fingerprint-headerPrefix">`fingerprint.headerPrefix`</a> | Prefix of the headers that Traefix adds to the requests, listed in [Request Headers](#request-headers). Traefix removes every client request header whose name starts with this prefix. An empty prefix fails the start of the entry point. Example: `X-Fs-`. | `X-Traefix-` | No |
| <a id="opt-fingerprint-export-url" href="#opt-fingerprint-export-url" title="#opt-fingerprint-export-url">`fingerprint.export.url`</a> | URL that receives the [batches of records](#exported-records), sent with `POST` requests. The URL must be absolute, with the `http` or `https` scheme, or the start of the entry point fails. | | Yes, with `export` |
| <a id="opt-fingerprint-export-headers" href="#opt-fingerprint-export-headers" title="#opt-fingerprint-export-headers">`fingerprint.export.headers`</a> | Headers sent with each batch, for example an `Authorization` header with the key of the receiving service. Traefix treats the values as secrets and removes them from the configuration it logs. | | No |
| <a id="opt-fingerprint-export-rawBytes" href="#opt-fingerprint-export-rawBytes" title="#opt-fingerprint-export-rawBytes">`fingerprint.export.rawBytes`</a> | Exports the raw bytes of the TLS ClientHello and of the HTTP/2 connection prefix, so that the receiving service can compute its own fingerprints. When unset, the connection records only carry the computed fingerprints. | `false` | No |
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
| <a id="opt-X-Traefix-H2" href="#opt-X-Traefix-H2" title="#opt-X-Traefix-H2">`X-Traefix-H2`</a> | Akamai fingerprint of the HTTP/2 connection: settings, window update, priority frames and pseudo-header order. Present on HTTP/2 requests only. |
| <a id="opt-X-Traefix-Ja4h" href="#opt-X-Traefix-Ja4h" title="#opt-X-Traefix-Ja4h">`X-Traefix-Ja4h`</a> | [JA4H](https://github.com/FoxIO-LLC/ja4) fingerprint of the request, computed with the header order and case the client sent, and with the cookie values, which only leave Traefix with `allowSensitiveFields`. |

The headers carry the values of the socket even when the entry point trusts the forwarded headers of its peer. `X-Real-Ip` and `X-Forwarded-For` keep their own rules, set by [`forwardedHeaders`](./entrypoints.md#forwarded-headers).

`X-Traefix-Ja3` and `X-Traefix-Ja4` are absent when the connection isn't TLS, and when the ClientHello is longer than 16 KiB. `X-Traefix-Ja4h` is absent when Traefix couldn't follow the header blocks of the connection, for example after a malformed request.

!!! warning "Use one prefix per installation"

    Each entry point removes the client headers with its own prefix. A router that serves several entry points with different prefixes receives the headers of the other prefixes as the client sent them. Use the same `headerPrefix` on every entry point.

### HTTP/3

On HTTP/3 connections, Traefix adds `X-Traefix-Src-Ip`, `X-Traefix-Src-Port`, `X-Traefix-Dst-Port`, `X-Traefix-Conn-Id` and `X-Traefix-Proto`, and exports the connection and request records. The request records of HTTP/3 carry no header names and no JA4H, and the connection records carry no ClientHello fingerprint, because the QUIC library doesn't expose the received header order nor the ClientHello of the Initial packet.

## Exported Records

Each batch is a JSON object sent with `Content-Type: application/json`:

```json
{
  "schema": "traefix.fingerprint/v1",
  "sender": "traefix/3.8.0",
  "entryPoint": "websecure",
  "sentAt": "2026-11-17T21:30:00.123456789Z",
  "records": []
}
```

`sender` carries the version shown by `traefik version`. The `records` array mixes the two record types, told apart by their `type` field. Traefix omits the fields with no value.

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
| <a id="opt-tls-clientHello" href="#opt-tls-clientHello" title="#opt-tls-clientHello">`tls.clientHello`</a> | Raw bytes of the ClientHello, records included, in base64. Present with `rawBytes`. |
| <a id="opt-tls-clientHelloTruncated" href="#opt-tls-clientHelloTruncated" title="#opt-tls-clientHelloTruncated">`tls.clientHelloTruncated`</a> | `true` when the ClientHello is longer than 16 KiB, in which case Traefix keeps the first 16 KiB and computes no TLS fingerprint. |
| <a id="opt-http2-akamai" href="#opt-http2-akamai" title="#opt-http2-akamai">`http2.akamai`</a> | Akamai fingerprint of the HTTP/2 connection. |
| <a id="opt-http2-prefix" href="#opt-http2-prefix" title="#opt-http2-prefix">`http2.prefix`</a> | Raw bytes the client sent after the HTTP/2 preface, up to the end of the first header block, in base64. Present with `rawBytes`. |
| <a id="opt-http2-prefixTruncated" href="#opt-http2-prefixTruncated" title="#opt-http2-prefixTruncated">`http2.prefixTruncated`</a> | `true` when these bytes exceed 16 KiB, in which case Traefix keeps the first 16 KiB. |

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

| Metric | Labels | Contains |
|:-------|:-------|:---------|
| <a id="opt-traefik-fingerprint-export-records-total" href="#opt-traefik-fingerprint-export-records-total" title="#opt-traefik-fingerprint-export-records-total">`traefik_fingerprint_export_records_total`</a> | `entrypoint`, `type` | Records queued for export. |
| <a id="opt-traefik-fingerprint-export-dropped-total" href="#opt-traefik-fingerprint-export-dropped-total" title="#opt-traefik-fingerprint-export-dropped-total">`traefik_fingerprint_export_dropped_total`</a> | `entrypoint`, `reason` | Records dropped: `queue_full`, `send_error`, `http_4xx`, `http_5xx`, `too_large`. |
| <a id="opt-traefik-fingerprint-export-batches-total" href="#opt-traefik-fingerprint-export-batches-total" title="#opt-traefik-fingerprint-export-batches-total">`traefik_fingerprint_export_batches_total`</a> | `entrypoint`, `code` | Batches sent, by response status code, or `error` when no response came. |
| <a id="opt-traefik-fingerprint-export-queue-records" href="#opt-traefik-fingerprint-export-queue-records" title="#opt-traefik-fingerprint-export-queue-records">`traefik_fingerprint_export_queue_records`</a> | `entrypoint` | Records waiting to be sent. |
| <a id="opt-traefik-fingerprint-export-send-seconds" href="#opt-traefik-fingerprint-export-send-seconds" title="#opt-traefik-fingerprint-export-send-seconds">`traefik_fingerprint_export_send_seconds`</a> | `entrypoint` | Duration of the batch sending. |
| <a id="opt-traefik-fingerprint-capture-truncated-total" href="#opt-traefik-fingerprint-capture-truncated-total" title="#opt-traefik-fingerprint-capture-truncated-total">`traefik_fingerprint_capture_truncated_total`</a> | `entrypoint`, `part` | Captures cut at their size limit: `client_hello`, `h2_prefix`, `headers`. |

## Resource Use

Traefix reads the bytes the server already reads, and never changes them. While the ClientHello and the HTTP/2 prefix are captured, a connection holds up to 16 KiB for each. To follow the header order of every HTTP/2 request, a connection also keeps its own HPACK decoder, bounded by `http2.maxDecoderHeaderTableSize`, for its whole life.
