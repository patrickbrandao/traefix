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
| <a id="opt-fingerprint-headerDetail" href="#opt-fingerprint-headerDetail" title="#opt-fingerprint-headerDetail">`fingerprint.headerDetail`</a> | Detail of the request headers: `basic` adds the fingerprints and the connection data; `full` also adds the TLS, HTTP/2, HTTP/3 and request header details described in [Detail Headers](#detail-headers); `raw` also adds the raw SYN and ClientHello. Another value fails the start of the entry point. | `full` | No |
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
| <a id="opt-X-Traefix-Server-Ip" href="#opt-X-Traefix-Server-Ip" title="#opt-X-Traefix-Server-Ip">`X-Traefix-Server-Ip`</a> | IP address the connection arrived on, from the socket. Behind a destination NAT, such as the published ports of a Docker container in a bridge network, it is the address after the NAT, the one of the container. Absent on HTTP/3, whose UDP socket is bound to all the addresses. |
| <a id="opt-X-Traefix-Server-Port" href="#opt-X-Traefix-Server-Port" title="#opt-X-Traefix-Server-Port">`X-Traefix-Server-Port`</a> | Port the connection arrived on, the same value as `X-Traefix-Dst-Port`. |
| <a id="opt-X-Traefix-Time" href="#opt-X-Traefix-Time" title="#opt-X-Traefix-Time">`X-Traefix-Time`</a> | Time Traefix finished reading the request header block, in RFC 3339 format with nanoseconds, UTC, from the system clock. It isn't the arrival time of the packets at the network interface. |
| <a id="opt-X-Traefix-Dst-Port" href="#opt-X-Traefix-Dst-Port" title="#opt-X-Traefix-Dst-Port">`X-Traefix-Dst-Port`</a> | Port of the entry point that accepted the connection. |
| <a id="opt-X-Traefix-Conn-Id" href="#opt-X-Traefix-Conn-Id" title="#opt-X-Traefix-Conn-Id">`X-Traefix-Conn-Id`</a> | Identifier of the connection, a [ULID](https://github.com/ulid/spec) of 26 characters, ordered by accept time. |
| <a id="opt-X-Traefix-Proto" href="#opt-X-Traefix-Proto" title="#opt-X-Traefix-Proto">`X-Traefix-Proto`</a> | HTTP version of the request: `h1`, `h2` or `h3`. |
| <a id="opt-X-Traefix-Ja3" href="#opt-X-Traefix-Ja3" title="#opt-X-Traefix-Ja3">`X-Traefix-Ja3`</a> | MD5 hash of the [JA3](https://github.com/salesforce/ja3) fingerprint of the TLS ClientHello. |
| <a id="opt-X-Traefix-Ja3n" href="#opt-X-Traefix-Ja3n" title="#opt-X-Traefix-Ja3n">`X-Traefix-Ja3n`</a> | MD5 hash of the JA3N fingerprint: the JA3 with the extensions sorted, which stays the same when a client shuffles its extensions, as Chrome does. |
| <a id="opt-X-Traefix-Ja4" href="#opt-X-Traefix-Ja4" title="#opt-X-Traefix-Ja4">`X-Traefix-Ja4`</a> | [JA4](https://github.com/FoxIO-LLC/ja4) fingerprint of the TLS ClientHello. |
| <a id="opt-X-Traefix-Alpn" href="#opt-X-Traefix-Alpn" title="#opt-X-Traefix-Alpn">`X-Traefix-Alpn`</a> | Application protocols offered in the TLS ClientHello, in the client order, separated by commas, for example `h2,http/1.1`. The bytes of a protocol name that are not visible ASCII, the comma and the percent sign are percent-encoded, for example `%2C`. |
| <a id="opt-X-Traefix-Ja4t" href="#opt-X-Traefix-Ja4t" title="#opt-X-Traefix-Ja4t">`X-Traefix-Ja4t`</a> | [JA4T](https://github.com/FoxIO-LLC/ja4) fingerprint of the TCP SYN of the connection, with the `tcp` option. |
| <a id="opt-X-Traefix-Ttl" href="#opt-X-Traefix-Ttl" title="#opt-X-Traefix-Ttl">`X-Traefix-Ttl`</a> | TTL, or hop limit, of the TCP SYN as received, with the `tcp` option. |
| <a id="opt-X-Traefix-Transport" href="#opt-X-Traefix-Transport" title="#opt-X-Traefix-Transport">`X-Traefix-Transport`</a> | What the IP, TCP and QUIC layers tell of the connection, as `key=value` pairs separated by commas, described in [Transport Header](#transport-header). Over TCP, it needs the `tcp` option. |
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

### Transport Header

`X-Traefix-Transport` carries `key=value` pairs separated by commas. The values are integers, `0` or `1` for the flags, and durations in microseconds with an `_us` suffix. A list inside a value is separated by `-`. A receiving service must ignore the keys it doesn't know, as new keys can appear.

Over TCP, with the `tcp` option, on Linux:

```text
proto=tcp,ipv=4,ttl=48,ttl_init=64,hops=16,dscp=0,ecn=0,df=1,ipid=0x0,ipopt=0,syn_len=64,syn_win=65535,syn_mss=1460,mtu_est=1500,syn_wscale=6,syn_sack=1,syn_ts=1,syn_ecn=0,syn_tfo=0,syn_opts=2-1-3-1-1-8-4-0-0,mss_snd=1448,mss_rcv=1448,mss_adv=1460,pmtu=1500,wscale_snd=6,wscale_rcv=7,opt_ts=1,opt_sack=1,opt_ecn=0,rtt_syn_us=23412,rtt_us=23980,rttvar_us=4210,rtt_min_us=23101,cwnd=10,retrans=0,lost=0,cc=cubic,accepted=2026-11-19T14:39:13.266226Z,conn_age_us=48211,req_seq=3
```

| Keys | Source | Contain |
|:-----|:-------|:--------|
| <a id="opt-proto-ipv" href="#opt-proto-ipv" title="#opt-proto-ipv">`proto`, `ipv`</a> | Connection | `tcp` or `udp`, and the IP version, `4` or `6`. |
| <a id="opt-ttl-ttl-init-hops" href="#opt-ttl-ttl-init-hops" title="#opt-ttl-ttl-init-hops">`ttl`, `ttl_init`, `hops`</a> | IP header of the SYN | TTL or hop limit as received, the usual initial value closest above it (32, 64, 128 or 255), and the distance in hops this guess gives. |
| <a id="opt-dscp-ecn" href="#opt-dscp-ecn" title="#opt-dscp-ecn">`dscp`, `ecn`</a> | IP header of the SYN | DSCP and ECN bits of the TOS or traffic class. |
| <a id="opt-df-ipid-ipopt" href="#opt-df-ipid-ipopt" title="#opt-df-ipid-ipopt">`df`, `ipid`, `ipopt`</a> | IPv4 header of the SYN | DF flag, identification in hexadecimal, length of the IP options. |
| <a id="opt-flow-label" href="#opt-flow-label" title="#opt-flow-label">`flow_label`</a> | IPv6 header of the SYN | Flow label, in hexadecimal. |
| <a id="opt-syn-len-syn-win" href="#opt-syn-len-syn-win" title="#opt-syn-len-syn-win">`syn_len`, `syn_win`</a> | SYN | Length of the packet, and TCP window, unscaled. |
| <a id="opt-syn-mss-mtu-est" href="#opt-syn-mss-mtu-est" title="#opt-syn-mss-mtu-est">`syn_mss`, `mtu_est`</a> | SYN | MSS option as received, which a device on the path may have lowered (PPPoE, VPN, MSS clamping), and the MTU it suggests: the MSS plus 40 bytes in IPv4, 60 in IPv6. 1500 is plain Ethernet, 1492 PPPoE, about 1420 WireGuard. |
| <a id="opt-syn-wscale-syn-sack-syn-ts" href="#opt-syn-wscale-syn-sack-syn-ts" title="#opt-syn-wscale-syn-sack-syn-ts">`syn_wscale`, `syn_sack`, `syn_ts`</a> | SYN | Window scale, SACK permitted and timestamps options. |
| <a id="opt-syn-ecn-syn-tfo-syn-opts" href="#opt-syn-ecn-syn-tfo-syn-opts" title="#opt-syn-ecn-syn-tfo-syn-opts">`syn_ecn`, `syn_tfo`, `syn_opts`</a> | SYN | ECN asked with the ECE and CWR flags, TCP Fast Open option, and the kinds of the options in order, as in JA4T. |
| <a id="opt-mss-snd-mss-rcv-mss-adv-pmtu" href="#opt-mss-snd-mss-rcv-mss-adv-pmtu" title="#opt-mss-snd-mss-rcv-mss-adv-pmtu">`mss_snd`, `mss_rcv`, `mss_adv`, `pmtu`</a> | `TCP_INFO` | MSS Traefix sends with, the lower of the client MSS and the path MTU, without the TCP options; the kernel estimate of the segment size it receives, which changes during the connection; the MSS Traefix advertised; and the path MTU the kernel knows. |
| <a id="opt-wscale-snd-wscale-rcv-opt-ts-opt-sack-opt-ecn" href="#opt-wscale-snd-wscale-rcv-opt-ts-opt-sack-opt-ecn" title="#opt-wscale-snd-wscale-rcv-opt-ts-opt-sack-opt-ecn">`wscale_snd`, `wscale_rcv`, `opt_ts`, `opt_sack`, `opt_ecn`</a> | `TCP_INFO` | Options negotiated, not only offered. |
| <a id="opt-rtt-syn-us" href="#opt-rtt-syn-us" title="#opt-rtt-syn-us">`rtt_syn_us`</a> | `TCP_INFO` at the accept | Round-trip time of the handshake. |
| <a id="opt-rtt-us-rttvar-us-rtt-min-us" href="#opt-rtt-us-rttvar-us-rtt-min-us" title="#opt-rtt-us-rttvar-us-rtt-min-us">`rtt_us`, `rttvar_us`, `rtt_min_us`</a> | `TCP_INFO` at the request | Smoothed round-trip time, its variation, and the lowest one. |
| <a id="opt-cwnd-retrans-lost-cc" href="#opt-cwnd-retrans-lost-cc" title="#opt-cwnd-retrans-lost-cc">`cwnd`, `retrans`, `lost`, `cc`</a> | `TCP_INFO`, `TCP_CONGESTION` | Congestion window in segments, retransmitted and lost segments, congestion control algorithm. |
| <a id="opt-accepted-conn-age-us-req-seq" href="#opt-accepted-conn-age-us-req-seq" title="#opt-accepted-conn-age-us-req-seq">`accepted`, `conn_age_us`, `req_seq`</a> | Traefix | Accept time of the connection (RFC 3339 with nanoseconds, UTC), time since then, and the rank of the request on the connection, from 1. |

Over HTTP/3:

```text
proto=udp,ipv=4,quic_ver=1,initial_size=1250,dcid_len=8,scid_len=8,initial_pkts=1,rtt_us=23500,rttvar_us=3900,rtt_min_us=23100,cwnd=13500,lost=0,conn_age_us=31020,req_seq=1
```

`quic_ver`, `initial_size` (the size of the first datagram of the client, padded by each client to its own size), `dcid_len`, `scid_len` and `initial_pkts` need `export.rawBytes`, which reads the Initial packets. `rtt_us`, `rttvar_us`, `rtt_min_us`, `cwnd` (in bytes) and `lost` (packets) come from the QUIC library. The TTL and the traffic class of the UDP datagrams aren't available.

Traefix reads `TCP_INFO` with one system call per request.

### Detail Headers

With `headerDetail` set to `full`, the default, or `raw`, Traefix also adds these headers. They use the same `key=value` format as `X-Traefix-Transport`, except the lists of names, which are separated by commas. A value from the client, such as the server name or a header name, has its bytes outside visible ASCII and its separators percent-encoded.

| Header | Contains | Example |
|:-------|:---------|:--------|
| <a id="opt-X-Traefix-Tls" href="#opt-X-Traefix-Tls" title="#opt-X-Traefix-Tls">`X-Traefix-Tls`</a> | Negotiated TLS version, cipher suite and key exchange group, ALPN, session resumption, accepted ECH, client certificate, server name of the ClientHello, HelloRetryRequest (the negotiated group isn't one of the key shares of the first ClientHello), ClientHello length, and `transport=quic` over HTTP/3. | `ver=1.3,cipher=TLS_AES_128_GCM_SHA256,group=X25519MLKEM768,alpn=h2,resumed=0,ech=0,client_cert=0,sni=example.com,hrr=0,ch_len=1795` |
| <a id="opt-X-Traefix-Tls-Hello" href="#opt-X-Traefix-Tls-Hello" title="#opt-X-Traefix-Tls-Hello">`X-Traefix-Tls-Hello`</a> | The lists of the ClientHello in the client order, in hexadecimal, with `g` for the GREASE values: legacy version, cipher suites, extensions, supported groups, key share groups, signature algorithms, supported versions, point formats, PSK modes, certificate compression, record size limit, ALPN, ALPS, and the presence of an ECH extension. | `legacy_ver=0303,ciphers=g-1301-1302-1303-c02b-…,exts=g-0000-0017-ff01-…,groups=g-11ec-001d-0017-0018,key_shares=g-11ec-001d,…,ech_ext=1` |
| <a id="opt-X-Traefix-Ja3-String-X-Traefix-Ja3n-String" href="#opt-X-Traefix-Ja3-String-X-Traefix-Ja3n-String" title="#opt-X-Traefix-Ja3-String-X-Traefix-Ja3n-String">`X-Traefix-Ja3-String`, `X-Traefix-Ja3n-String`</a> | JA3 and JA3N before hashing. | `771,4865-4866-4867-49195-…,0-23-65281-…,4588-29-23-24,0` |
| <a id="opt-X-Traefix-Ja4r-X-Traefix-Ja4o" href="#opt-X-Traefix-Ja4r-X-Traefix-Ja4o" title="#opt-X-Traefix-Ja4r-X-Traefix-Ja4o">`X-Traefix-Ja4r`, `X-Traefix-Ja4o`</a> | JA4_r and JA4_o, as in the connection record. | `t13d1516h2_002f,0035,…_0005,000a,…_0403,0804,…` |
| <a id="opt-X-Traefix-Ja4h-R" href="#opt-X-Traefix-Ja4h-R" title="#opt-X-Traefix-Ja4h-R">`X-Traefix-Ja4h-R`</a> | JA4H before hashing, as the FoxIO tool prints `ja4h_r`: header names in order, sorted cookie names, sorted cookie pairs. | `ge20nn13ptbr_sec-ch-ua,sec-ch-ua-mobile,…,priority__` |
| <a id="opt-X-Traefix-P0f" href="#opt-X-Traefix-P0f" title="#opt-X-Traefix-P0f">`X-Traefix-P0f`</a> | p0f v3 signature of the SYN, with the `tcp` option. | `4:50+14:0:1460:65535,6:mss,nop,ws,nop,nop,ts,sok,eol+1:df,ecn:0` |
| <a id="opt-X-Traefix-H2-Detail" href="#opt-X-Traefix-H2-Detail" title="#opt-X-Traefix-H2-Detail">`X-Traefix-H2-Detail`</a> | The parts of the Akamai fingerprint, with `-` between the list items, the priority and the flags of the first `HEADERS` frame. HTTP/2 only. | `settings=1:65536;2:0;4:6291456;6:262144,window_update=15663105,priority=0,headers_priority=1:1:0:256,first_headers_flags=0x25,pseudo=m-a-s-p` |
| <a id="opt-X-Traefix-H3" href="#opt-X-Traefix-H3" title="#opt-X-Traefix-H3">`X-Traefix-H3`</a> | `SETTINGS` of the HTTP/3 connection in the client order, with `http3HeaderOrder`, and the QUIC transport parameters in the client order, with `export.rawBytes`: `name:value` for an integer, `name:<length>b` for bytes, the name alone for a flag, `grease` for a reserved identifier. HTTP/3 only. | `settings=1:65536;7:100;6:65536,tp=max_idle_timeout:30000-max_udp_payload:1472-initial_max_data:15728640-…-grease` |
| <a id="opt-X-Traefix-Header-Order" href="#opt-X-Traefix-Header-Order" title="#opt-X-Traefix-Header-Order">`X-Traefix-Header-Order`</a> | Names of the request headers in the order and case the client sent, without the HTTP/2 and HTTP/3 pseudo-headers. | `Host,User-Agent,Accept,Accept-Language,Accept-Encoding,Cookie` |
| <a id="opt-X-Traefix-Cookie-Names" href="#opt-X-Traefix-Cookie-Names" title="#opt-X-Traefix-Cookie-Names">`X-Traefix-Cookie-Names`</a> | Names of the cookies in the received order, without their values. | `sid,_ga,theme` |
| <a id="opt-X-Traefix-Syn" href="#opt-X-Traefix-Syn" title="#opt-X-Traefix-Syn">`X-Traefix-Syn`</a> | Raw SYN from its IP header, in base64. `raw` level and `tcp` option. | `RQAAPAAAQAAyBg…` |
| <a id="opt-X-Traefix-Client-Hello" href="#opt-X-Traefix-Client-Hello" title="#opt-X-Traefix-Client-Hello">`X-Traefix-Client-Hello`</a> | Raw ClientHello, records included over TCP, in base64. `raw` level. A ClientHello longer than 4 KiB once encoded is left out, and `X-Traefix-Tls` ends with `ch_raw=omitted`. | `FgMBBuIBAAbeAwM…` |

The headers stay within the defaults of the common backends: 8 KB per header line for nginx, 8190 bytes for Apache httpd and its 100 fields, 16 KB for all the headers of a request for Node.js. Each detail header is bounded to 2 KiB, 4 KiB for the raw ClientHello, and a longer list ends with `truncated=1`, or `…truncated` in the lists of names. With `full`, the headers of Traefix take about 3.5 KB for a browser over HTTP/2, and about 6 KB with `raw`. Use `headerDetail: basic` for a backend with a lower limit.

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
| <a id="opt-http3-settings" href="#opt-http3-settings" title="#opt-http3-settings">`http3.settings`</a> | `SETTINGS` of the HTTP/3 connection, as `id:value` separated by `;`, in the client order, GREASE identifiers included, with `http3HeaderOrder`. |
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
