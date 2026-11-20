---
title: "Traefix Fingerprint Signals"
description: "Specification of every value Traefix gives about a connection and a request, in the X-Traefix headers and in the exported records, with its source, unit, precision and edge cases, and how a detection service reads them to tell browsers, bots, proxies and tunnels apart."
---

This page specifies the values the [`fingerprint`](./entrypoints-fingerprint.md) option of a Traefix entry point gives about each connection and each request, for a service that builds a detection mechanism on them: bot detection, proxy and tunnel detection, client consistency checks. Traefix measures and reports; it never decides.

The values reach that service in two ways, with the same content:

- the request headers, added before any router and sent to the backend and to the [`forwardAuth`](../routing-configuration/http/middlewares/forwardauth-failopen.md) server;
- the [exported records](./entrypoints-fingerprint.md#exported-records), described by the [JSON Schema](traefix-fingerprint-v1.schema.json).

The names below use the default `X-Traefix-` prefix, set by `headerPrefix`.

## Contract

- **Trust.** Traefix removes every client header whose name starts with the prefix, in any case and with `_` in place of `-`. A value under the prefix always comes from Traefix.
- **Absence.** A header or a key that Traefix couldn't measure is left out, never sent empty or as `0`. A missing value means "unknown", not "no".
- **Stability.** A new header, a new key or a new value can appear in a release: ignore what you don't know. Removing or changing the meaning of a header or a key only comes with the `traefix.fingerprint/v2` schema, as [Schema Versioning](./entrypoints-fingerprint.md#schema-versioning) describes.
- **Format.** The headers with several values use `key=value` pairs separated by `,`. The values are integers, `0` or `1` for the flags, hexadecimal with a `0x` prefix where noted, or text. A list inside a value is separated by `-`. A value that comes from the client has its bytes outside visible ASCII, `%` and the separators of its header percent-encoded, for example `%2C`. The headers that are lists of names use `,` with no key.
- **Size.** Each detail header is bounded to 2 KiB, 4 KiB for `X-Traefix-Client-Hello`. A cut list ends with `truncated=1`, or with `…truncated` in the lists of names. With `headerDetail: full`, the headers of Traefix take about 3.5 KB for a browser over HTTP/2.
- **Clocks.** All the absolute times are UTC in RFC 3339 format with nanoseconds, from `CLOCK_REALTIME`. The kernel times and the Traefix times use the same clock and can be subtracted.

## Availability

| Group | Needs | HTTP/1 | HTTP/2 | HTTP/3 |
|:------|:------|:------:|:------:|:------:|
| Connection data, `Time`, fingerprints of TLS and HTTP | `fingerprint` | ✔ | ✔ | ✔ (JA4H with `http3HeaderOrder`) |
| TCP view: `Ja4t`, `Ttl`, `P0f`, the SYN and `TCP_INFO` keys of `Transport` | `fingerprint.tcp`, Linux | ✔ | ✔ | — |
| UDP view: `Ttl` and the `ttl`, `ttl_init`, `hops`, `dscp`, `ecn` keys of `Transport`, `udp` record | `fingerprint`, Linux, macOS or FreeBSD | — | — | ✔ |
| Detail headers, `Timing` | `headerDetail: full` (default) or `raw` | ✔ | ✔ | ✔ |
| Kernel arrival times in `Timing` (`src=kernel`) | `fingerprint.tcp`, Linux | ✔ | ✔ | — |
| QUIC Initial data, transport parameters | `export.rawBytes` | — | — | ✔ |
| Raw SYN and ClientHello | `headerDetail: raw` | ✔ | ✔ | ✔ (ClientHello with `rawBytes`) |

## Headers

### Connection

| Header | Type | Content | Example |
|:-------|:-----|:--------|:--------|
| <a id="opt-sig-Src-Ip" href="#opt-sig-Src-Ip" title="#opt-sig-Src-Ip">`X-Traefix-Src-Ip`</a>, `X-Traefix-Src-Port` | IP, integer | Peer of the connection, from the socket or from the PROXY protocol header. | `189.113.138.178`, `51556` |
| <a id="opt-sig-Server-Ip" href="#opt-sig-Server-Ip" title="#opt-sig-Server-Ip">`X-Traefix-Server-Ip`</a>, `X-Traefix-Server-Port` | IP, integer | Address and port the connection arrived on. Behind a destination NAT, the address after the NAT. No address over HTTP/3. | `10.249.255.253`, `443` |
| <a id="opt-sig-Dst-Port" href="#opt-sig-Dst-Port" title="#opt-sig-Dst-Port">`X-Traefix-Dst-Port`</a> | integer | Same value as `X-Traefix-Server-Port`. | `443` |
| <a id="opt-sig-Conn-Id" href="#opt-sig-Conn-Id" title="#opt-sig-Conn-Id">`X-Traefix-Conn-Id`</a> | ULID | Identifier of the connection, the `connId` of the records. The same on every request of the connection, and across an HTTP/3 path migration. | `01M7T69KA9ZW23EFYFDTG4WMHV` |
| <a id="opt-sig-Proto" href="#opt-sig-Proto" title="#opt-sig-Proto">`X-Traefix-Proto`</a> | `h1`, `h2`, `h3` | HTTP version of the request. | `h2` |
| <a id="opt-sig-Time" href="#opt-sig-Time" title="#opt-sig-Time">`X-Traefix-Time`</a> | time | End of the reading of the request header block, by Traefix. | `2026-11-19T15:20:09.340881995Z` |

### TLS

| Header | Content | Example |
|:-------|:--------|:--------|
| <a id="opt-sig-Alpn" href="#opt-sig-Alpn" title="#opt-sig-Alpn">`X-Traefix-Alpn`</a> | Application protocols offered, in the client order. | `h2,http/1.1` |
| <a id="opt-sig-Ja3" href="#opt-sig-Ja3" title="#opt-sig-Ja3">`X-Traefix-Ja3`</a>, `X-Traefix-Ja3-String` | JA3 (Salesforce) hash and string, without GREASE. Changes on each connection for a client shuffling its extensions. | `375c6162…`, `771,4867-4866-…,43-51-0-…,29-23-24-25,0` |
| <a id="opt-sig-Ja3n" href="#opt-sig-Ja3n" title="#opt-sig-Ja3n">`X-Traefix-Ja3n`</a>, `X-Traefix-Ja3n-String` | JA3 with the extensions sorted: stable for a client shuffling its extensions. | `90a369eb…` |
| <a id="opt-sig-Ja4" href="#opt-sig-Ja4" title="#opt-sig-Ja4">`X-Traefix-Ja4`</a>, `X-Traefix-Ja4r`, `X-Traefix-Ja4o` | FoxIO JA4: sorted and hashed, sorted and raw, client order and hashed. `t` over TCP, `q` over QUIC. Checked against the FoxIO tool. | `t13d4907h2_0d8feac7bc37_7395dae3b2f3` |
| <a id="opt-sig-Tls" href="#opt-sig-Tls" title="#opt-sig-Tls">`X-Traefix-Tls`</a> | Negotiated parameters, keys below. | `ver=1.3,cipher=TLS_AES_128_GCM_SHA256,group=X25519MLKEM768,…` |
| <a id="opt-sig-Tls-Hello" href="#opt-sig-Tls-Hello" title="#opt-sig-Tls-Hello">`X-Traefix-Tls-Hello`</a> | Lists of the ClientHello, keys below. | `legacy_ver=0303,ciphers=g-1301-…` |
| <a id="opt-sig-Client-Hello" href="#opt-sig-Client-Hello" title="#opt-sig-Client-Hello">`X-Traefix-Client-Hello`</a> | Raw ClientHello in base64, records included over TCP, with `raw`. | `FgMBBuIBAAbeAwM…` |

`X-Traefix-Tls` keys:

| Key | Content |
|:----|:--------|
| `ver` | Negotiated version: `1.3`, `1.2`, `1.1`, `1.0`. |
| `cipher` | IANA name of the negotiated cipher suite. |
| `group` | Key exchange group, for example `X25519`, `X25519MLKEM768`, `CurveP256`. |
| `alpn` | Negotiated application protocol. |
| `resumed`, `ech`, `client_cert` | Session resumption, accepted encrypted client hello, client certificate. |
| `sni` | Server name of the ClientHello. It can differ from the `Host` of the request. |
| `hrr` | `1` when the negotiated group isn't one of the key shares of the first ClientHello: the server sent a HelloRetryRequest, and the handshake took one more round trip. |
| `ch_len` | Length of the ClientHello in bytes, records included over TCP. |
| `transport` | `quic` over HTTP/3. |
| `ch_raw` | `omitted` when the raw ClientHello exceeds the size of its header. |

`X-Traefix-Tls-Hello` keys, in the client order, 16-bit values in hexadecimal with `g` for the GREASE values (RFC 8701):

| Key | Content |
|:----|:--------|
| `legacy_ver` | Legacy version field. |
| `ciphers`, `exts` | Cipher suites, extension types. |
| `groups`, `key_shares` | Supported groups, groups of the key shares. |
| `sigalgs`, `versions` | Signature algorithms, supported versions. |
| `point_formats`, `psk_modes` | EC point formats and PSK key exchange modes, in decimal. |
| `cert_compress`, `record_limit` | Certificate compression algorithms (RFC 8879), record size limit (RFC 8449). |
| `alpn`, `alps` | ALPN protocols, application settings protocols. |
| `ech_ext` | `1` when an encrypted client hello extension is present, real or GREASE. |

### HTTP

| Header | Content | Example |
|:-------|:--------|:--------|
| <a id="opt-sig-H2" href="#opt-sig-H2" title="#opt-sig-H2">`X-Traefix-H2`</a> | Akamai HTTP/2 fingerprint, HTTP/2 only. | `1:65536;2:0;4:6291456;6:262144\|15663105\|0\|m,a,s,p` |
| <a id="opt-sig-H2-Detail" href="#opt-sig-H2-Detail" title="#opt-sig-H2-Detail">`X-Traefix-H2-Detail`</a> | `settings` (`id:value;…` in client order), `window_update`, `priority` (frames, `-` separated), `headers_priority` (`stream:exclusive:dependency:weight` of the first `HEADERS`), `first_headers_flags` (hexadecimal), `pseudo` (first letters, `-` separated). | `settings=1:65536;2:0;4:6291456;6:262144,window_update=15663105,priority=0,headers_priority=1:1:0:256,first_headers_flags=0x25,pseudo=m-a-s-p` |
| <a id="opt-sig-H3" href="#opt-sig-H3" title="#opt-sig-H3">`X-Traefix-H3`</a> | `settings` in client order, GREASE included (`http3HeaderOrder`), `tp`: QUIC transport parameters in client order (`rawBytes`). | `settings=1:65536;7:100;6:65536,tp=max_idle_timeout:30000-max_udp_payload:1472-…-grease` |
| <a id="opt-sig-Ja4h" href="#opt-sig-Ja4h" title="#opt-sig-Ja4h">`X-Traefix-Ja4h`</a>, `X-Traefix-Ja4h-R` | FoxIO JA4H of the request, hashed and raw, from the header order and case on the wire. Checked against the FoxIO tool. | `ge20nn020000_5594a17e7e7e_000000000000_000000000000`, `ge20nn020000_user-agent,accept__` |
| <a id="opt-sig-Header-Order" href="#opt-sig-Header-Order" title="#opt-sig-Header-Order">`X-Traefix-Header-Order`</a> | Header names in the order and case the client sent, without pseudo-headers. The backend itself gets them reordered and canonicalized. | `Host,User-Agent,Accept,Accept-Language,Accept-Encoding,Cookie` |
| <a id="opt-sig-Cookie-Names" href="#opt-sig-Cookie-Names" title="#opt-sig-Cookie-Names">`X-Traefix-Cookie-Names`</a> | Cookie names in the received order, without the values. | `sid,_ga,theme` |

### TCP and IP

| Header | Content | Example |
|:-------|:--------|:--------|
| <a id="opt-sig-Ja4t" href="#opt-sig-Ja4t" title="#opt-sig-Ja4t">`X-Traefix-Ja4t`</a> | FoxIO JA4T of the SYN. Checked against the FoxIO tool. | `65535_2-1-3-1-1-8-4-0-0_1460_6` |
| <a id="opt-sig-Ttl" href="#opt-sig-Ttl" title="#opt-sig-Ttl">`X-Traefix-Ttl`</a> | TTL or hop limit as received: of the SYN over TCP (`fingerprint.tcp`), of the first Initial datagram over HTTP/3. Absent when unknown: see [TTL](#ttl). | `50` |
| <a id="opt-sig-P0f" href="#opt-sig-P0f" title="#opt-sig-P0f">`X-Traefix-P0f`</a> | p0f v3 `raw_sig` of the SYN. Checked against p0f 3.09b. Match it against a `p0f.fp` database for an operating system guess. | `4:50+14:0:1460:65535,6:mss,nop,ws,nop,nop,ts,sok,eol+1:df,ecn:0` |
| <a id="opt-sig-Syn" href="#opt-sig-Syn" title="#opt-sig-Syn">`X-Traefix-Syn`</a> | Raw SYN from its IP header, base64, with `raw`. | `RQAAPAAAQAAyBg…` |
| <a id="opt-sig-Transport" href="#opt-sig-Transport" title="#opt-sig-Transport">`X-Traefix-Transport`</a> | IP, TCP and QUIC data, specified in [Transport Header](./entrypoints-fingerprint.md#transport-header). | `proto=tcp,ipv=4,ttl=50,…` |

### Timing

`X-Traefix-Timing` gives the milestones of the start of the connection, as durations in microseconds. The same values are in the `timing` object of the connection record, with the record names below.

Over TCP:

```text
src=kernel,proto=tcp,tcp_rtt_us=120412,est_to_first_us=124980,ch_us=35,srv_us=410,tls_rtt_us=240115,tls_us=245300,rtt_gap_us=119703,req_us=246020
```

| Key | Record field | From → to | Source |
|:----|:-------------|:----------|:-------|
| `src` | `source` | `kernel` when the arrival times come from the kernel, `user` when they come from the Traefix clock. | — |
| `tcp_rtt_us` | `tcpRttMicros` | Round trip of the TCP handshake, SYN-ACK sent → ACK received, measured by the kernel with the peer of the TCP connection. Needs the `tcp` option. | kernel |
| `est_to_first_us` | `estToFirstMicros` | Accept of the connection by Traefix → arrival of the first byte of the client. Negative when the bytes were already waiting at the accept. | arrival: `src`; accept: Traefix |
| `ch_us` | `clientHelloMicros` | First byte → last byte of the ClientHello. `0` when it fits in one read. | `src` |
| `srv_us` | `serverMicros` | End of the ClientHello → end of the server flight. The processing time of Traefix and the write of the flight. | Traefix |
| `tls_rtt_us` | `tlsRttMicros` | End of the server flight → arrival of the first client bytes that answer it: the client `Finished` in TLS 1.3, the `ClientKeyExchange` flight in TLS 1.2. One round trip with the TLS client, which includes the client computation of the keys. | arrival: `src`; flight: Traefix |
| `tls_us` | `handshakeMicros` | First byte → end of the TLS handshake in Traefix. | Traefix |
| `rtt_gap_us` | `rttGapMicros` | `tls_rtt_us − tcp_rtt_us`. | — |
| `req_us` | — | Accept → end of the request header block. Changes with each request of the connection. | Traefix |

The record also carries `firstByte`, the absolute arrival time of the first byte.

Over HTTP/3:

```text
src=user,proto=udp,initial_to_hs_us=48210,initial_to_acc_us=9400,quic_rtt_us=23500,quic_min_rtt_us=23100,req_us=39800
```

| Key | Record field | Content |
|:----|:-------------|:--------|
| `initial_to_hs_us` | `initialToHandshakeMicros` | First Initial datagram → end of the QUIC handshake. Without `rawBytes`, the first Initial is timed when its ClientHello is read. |
| `initial_to_acc_us` | `initialToAcceptMicros` | First Initial datagram → accept of the QUIC connection, the origin of `req_us`: add it to `req_us` to place a request on the same scale as `initial_to_hs_us`. |
| `quic_rtt_us`, `quic_min_rtt_us` | `quicRttMicros`, `quicMinRttMicros` | Smoothed and lowest round trip of the QUIC connection, end to end. |
| `req_us` | — | Connection → end of the request header block. |

Precision: with `src=kernel`, the arrival times are the software receive timestamps of the kernel (`SO_TIMESTAMPNS`), taken when the segment reaches the socket layer, within a few microseconds of the network interface. With `src=user`, they are the time Traefix read the bytes, which adds the scheduling delay of the process: microseconds on an idle host, up to milliseconds under load. The kernel only times the handshake: Traefix stops the timestamps once the TLS handshake is timed.

## Detection with these signals

The recipes below are what the signals allow. The thresholds are starting points, to tune with the traffic of each site.

### Proxy and relay (SOCKS5, HTTP CONNECT, TCP relays)

A relay opens the TCP connection to Traefix itself, and carries the TLS of the real client end to end. The kernel measures `tcp_rtt_us` with the relay, while `tls_rtt_us` and `est_to_first_us` include the path between the client and the relay.

| Signal | Direct client | Behind a relay |
|:-------|:--------------|:---------------|
| `est_to_first_us` | 0 to 2 ms: the client sends the ClientHello as soon as the connection is up. | About the round trip client ↔ relay: the relay answers the client only once connected to Traefix, then the ClientHello crosses the relay. |
| `rtt_gap_us` | Close to 0, within the jitter and the key computation of the client (up to a few ms on a slow device). | About the round trip client ↔ relay. |

- Flag a relay when `rtt_gap_us` exceeds about 15 to 20 ms and 30% of `tcp_rtt_us`, and confirm with `est_to_first_us` of the same order. A single connection is noisy: aggregate the connections of the same client.
- A relay near the client (on the same host or network) adds almost nothing, and these signals don't show it.
- A layer 3 VPN carries the TCP end to end: `rtt_gap_us` stays near 0. Its signal is the MSS (below).
- A proxy that terminates the TLS, such as a corporate inspection proxy, gives both round trips with the proxy, and its own JA4: compare the JA4 with the `User-Agent`.
- `hrr=1` in `X-Traefix-Tls` adds one round trip to `tls_us`, not to `tls_rtt_us`.
- Over HTTP/3, the round trip is end to end: compare `quic_rtt_us` with the `tcp_rtt_us` of the same client on its TCP connections, or with the round trip the geolocation of the address suggests.

### Tunnels and links

- `syn_mss` and `mtu_est` of `X-Traefix-Transport`: 1460 (1440 in IPv6) is a plain Ethernet path, 1452 PPPoE, around 1380 to 1420 a VPN such as WireGuard or IPsec, lower values another encapsulation. The MSS is the one received, so a device on the path that lowers it (MSS clamping) shows here.
- `hops` and `ttl_init`: a TTL far from its initial value is a long path; an initial value that doesn't match the operating system of the `User-Agent` is a mismatch (below).

### TTL

The TTL (IPv4) or hop limit (IPv6) is the initial value of the sender's stack minus one per router on the path.

- **Source.** Over TCP, the SYN that the kernel saves (`fingerprint.tcp`, Linux). Over HTTP/3, the first Initial datagram, from the control messages of the UDP socket (Linux, macOS, FreeBSD). Both describe the first packet of the connection, before any option is negotiated.
- **Absent when** the kernel answered with SYN cookies (a full accept queue: no SYN saved), the `tcp` option is off or the system isn't Linux (TCP), the system doesn't report it (HTTP/3 on Windows, IPv4 on an IPv6 socket on macOS), or a PROXY protocol or load balancer stands before Traefix (the value is then the one of that device). Treat a missing value as unknown, never as 0.
- **Initial value.** `ttl_init` rounds the TTL up to the usual initial values 32, 64, 128 and 255, and `hops` is the difference. 64 is Linux, Android, macOS, iOS and the BSDs, 128 is Windows, 255 is network equipment and some Unix systems. Paths of more than 30 hops are rare, so the guess holds, except for the rare 32.
- **Docker.** A container on a bridge network receives the packets after the routing of the host, which takes one off the TTL: `hops` counts one more. With `network_mode: host`, it doesn't.
- **TCP against UDP.** The same client gives the same `ttl_init`, and usually the same `hops`, on its TCP and HTTP/3 connections. A difference means two stacks or two paths, for example a SOCKS5 proxy that relays the TCP and not the UDP, or a VPN that carries only one of them: aggregate the connections of the same address.

### Operating system and client consistency

- Compare the operating system of the `User-Agent` with `X-Traefix-P0f` (match it against `p0f.fp`), `X-Traefix-Ja4t` and `ttl_init`: Windows starts at 128, Linux, Android, macOS and iOS at 64.
- Compare the browser of the `User-Agent` with `X-Traefix-Ja4` (or `Ja3n`), `X-Traefix-H2` and `X-Traefix-Ja4h`: a library that imitates the TLS of a browser often misses its HTTP/2 settings, the priority of its first `HEADERS`, or its header order.
- `X-Traefix-Header-Order` and `X-Traefix-Ja4h-R` show the header order and case of the client, which an HTTP library changes.
- `X-Traefix-Tls` `sni` different from `Host` is domain fronting, or a client that doesn't send the name it asks for.
- `X-Traefix-Alpn` shows the protocols offered: a client that offers `h2` but sends HTTP/1.1, or a browser `User-Agent` without `h2`, is inconsistent.

### Connection reuse

`X-Traefix-Conn-Id` and `req_seq` in `X-Traefix-Transport` tell the requests of the same connection apart: the connection signals (TLS, TCP, timing) repeat, and only the request signals change.

## Records

The connection record carries the same values with their record names: `tls.*`, `http2.*`, `http3.*`, `tcp.*` and `timing.*`. It leaves the connection once, at its first request. The request record carries `ja4h`, `fieldNames`, `cookieNames` and the `fields` listed in `export.fields`, and the `status`, `router` and `service` of the response, which no request header can carry.
