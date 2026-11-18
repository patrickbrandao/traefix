# Reference captures

Copied from the reference set of finger-sniffer (`test/golden/tls/pcaps` and `test/golden/pcaps` of
`github.com/patrickbrandao/finger-sniffer`), so that the Traefix tests compare its fingerprints with the same
outputs as the services that read its records.

## tls/

Real clients against a TLS + HTTP/2 test server, captured with `tcpdump -s0` on 2026-11-09 (versions in
`tls/VERSIONS.txt`). The client address is replaced by `198.51.100.10`.

- `<case>.pcap`: the capture.
- `<case>.ja4.jsonl`: output of `ja4 -j -r` (FoxIO tool 0.18.6) on the capture with the TLS keys.
- `<case>.ref.json`: the part of the finger-sniffer `ref.json` the tests use: per ClientHello, the JA3 of tshark
  4.4 and the JA4 and JA4_r of the FoxIO tool; per HTTP/2 connection, the Akamai fingerprint built from the frames
  tshark decoded and the priority of the first `HEADERS` frame; the JA4H of the FoxIO tool.
- `<case>.client.h2`: the decrypted client side of the main HTTP/2 connection, from the preface on.

## tcp/

SYN captures of curl, Go and Chrome on macOS and Linux, IPv4 and IPv6, and of the FoxIO Windows sample, with the
JA4T of the FoxIO tool (`<case>.ja4.jsonl`) and the observations of `p0f -r` 3.09b (`<case>.p0f.txt`). The pcaps
keep only the packets with the SYN flag (`tshark -Y 'tcp.flags.syn==1'`), the reference outputs come from the full
captures.

JA4T belongs to the JA4+ suite, under the FoxIO License 1.1 (see `NOTICE` at the repository root).
