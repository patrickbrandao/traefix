package fingerprint

import (
	"crypto/tls"
	"encoding/base64"
	"slices"
	"strconv"
	"strings"
)

// The detail headers stay within the defaults of the common backends: 8 KB per line for nginx and 8190 bytes for
// Apache httpd, 16 KB for all the headers of a request for Node.js, 100 fields for Apache httpd.
const (
	// maxDetailHeader bounds a detail header; a longer list is cut at its last whole item, with a marker.
	maxDetailHeader = 2 << 10
	// maxClientHelloHeader bounds the raw ClientHello header, which is left out when longer.
	maxClientHelloHeader = 4 << 10
	// maxHeaderOrder and maxCookieNames bound the request lists.
	maxHeaderOrder = 1 << 10
	maxCookieNames = 512

	truncatedMarker = "truncated=1"
	truncatedItem   = "…truncated"
)

type headerDetail uint8

const (
	detailBasic headerDetail = iota
	detailFull
	detailRaw
)

// connDetail holds the detail header values of a connection, formatted once.
type connDetail struct {
	tls, tlsHello                       string
	ja3String, ja3nString, ja4r, ja4o   string
	p0f, h2Detail, syn, clientHello, tp string
}

// escapeValue percent-encodes the bytes that are not visible ASCII, and the given separators,
// so that a value from the client never breaks the key=value lists.
func escapeValue(s, separators string) string {
	const upperHex = "0123456789ABCDEF"

	clean := true
	for i := range len(s) {
		if c := s[i]; c <= ' ' || c >= 0x7f || c == '%' || strings.IndexByte(separators, c) >= 0 {
			clean = false
			break
		}
	}
	if clean {
		return s
	}

	var b strings.Builder
	for i := range len(s) {
		c := s[i]
		if c <= ' ' || c >= 0x7f || c == '%' || strings.IndexByte(separators, c) >= 0 {
			b.Write([]byte{'%', upperHex[c>>4], upperHex[c&0xf]})
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// boundedList joins the items with sep, up to max bytes; a cut list ends with the truncated marker.
func boundedList(items []string, sep string, limit int) string {
	var b strings.Builder
	for i, item := range items {
		need := len(item)
		if i > 0 {
			need += len(sep)
		}
		// Room is kept for the marker, unless the item is the last one.
		room := limit
		if i < len(items)-1 {
			room -= len(sep) + len(truncatedItem)
		}
		if b.Len()+need > room {
			if b.Len() > 0 {
				b.WriteString(sep)
			}
			b.WriteString(truncatedItem)
			return b.String()
		}
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(item)
	}
	return b.String()
}

// boundedPairs cuts a key=value list at its last whole pair under maxDetailHeader, with the truncated marker.
func boundedPairs(s string) string {
	if len(s) <= maxDetailHeader {
		return s
	}
	cut := strings.LastIndexByte(s[:maxDetailHeader-len(truncatedMarker)-1], ',')
	if cut < 0 {
		return truncatedMarker
	}
	return s[:cut] + "," + truncatedMarker
}

// hexDashList writes the 16-bit values in hexadecimal separated by '-', GREASE values as g.
func hexDashList(values []uint16) string {
	var b strings.Builder
	for i, v := range values {
		if i > 0 {
			b.WriteByte('-')
		}
		if isGREASE(v) {
			b.WriteByte('g')
			continue
		}
		b.WriteString(hex4List([]uint16{v}))
	}
	return b.String()
}

func tlsVersionShort(v uint16) string {
	switch v {
	case tls.VersionTLS13:
		return "1.3"
	case tls.VersionTLS12:
		return "1.2"
	case tls.VersionTLS11:
		return "1.1"
	case tls.VersionTLS10:
		return "1.0"
	default:
		return strconv.FormatUint(uint64(v), 16)
	}
}

// tlsDetail formats the negotiated TLS parameters. hrr tells whether the server asked for another key share,
// which shows when the negotiated group is not one of the key shares of the first ClientHello.
func tlsDetail(state tls.ConnectionState, hello *ClientHello, helloLen int, quic bool) string {
	w := transportWriter{b: make([]byte, 0, 256)}
	if state.HandshakeComplete {
		w.str("ver", tlsVersionShort(state.Version))
		w.str("cipher", tls.CipherSuiteName(state.CipherSuite))
		if state.CurveID != 0 {
			w.str("group", state.CurveID.String())
		}
		w.str("alpn", escapeValue(state.NegotiatedProtocol, ",="))
		w.bool("resumed", state.DidResume)
		w.bool("ech", state.ECHAccepted)
		w.bool("client_cert", len(state.PeerCertificates) > 0)
	}
	if hello != nil {
		w.str("sni", escapeValue(hello.ServerName, ",="))
		if state.HandshakeComplete && state.CurveID != 0 && len(hello.KeyShareGroups) > 0 {
			w.bool("hrr", !slices.Contains(hello.KeyShareGroups, uint16(state.CurveID)))
		}
	}
	if helloLen > 0 {
		w.int("ch_len", int64(helloLen))
	}
	if quic {
		w.str("transport", "quic")
	}
	return string(w.b)
}

// tlsHelloDetail formats the lists of the ClientHello, in the client order.
func tlsHelloDetail(h *ClientHello) string {
	w := transportWriter{b: make([]byte, 0, 768)}
	w.str("legacy_ver", hex4List([]uint16{h.LegacyVersion}))
	w.str("ciphers", hexDashList(h.CipherSuites))
	w.str("exts", hexDashList(h.Extensions))
	w.str("groups", hexDashList(h.SupportedGroups))
	w.str("key_shares", hexDashList(h.KeyShareGroups))
	w.str("sigalgs", hexDashList(h.SignatureAlgorithms))
	w.str("versions", hexDashList(h.SupportedVersions))
	if len(h.PointFormats) > 0 {
		formats := make([]string, len(h.PointFormats))
		for i, f := range h.PointFormats {
			formats[i] = strconv.Itoa(int(f))
		}
		w.str("point_formats", strings.Join(formats, "-"))
	}
	if len(h.PSKModes) > 0 {
		modes := make([]string, len(h.PSKModes))
		for i, m := range h.PSKModes {
			modes[i] = strconv.Itoa(int(m))
		}
		w.str("psk_modes", strings.Join(modes, "-"))
	}
	w.str("cert_compress", hexDashList(h.CertCompression))
	if h.RecordSizeLimit >= 0 {
		w.int("record_limit", int64(h.RecordSizeLimit))
	}
	if len(h.ALPN) > 0 {
		protos := make([]string, len(h.ALPN))
		for i, p := range h.ALPN {
			protos[i] = escapeValue(p, ",=-")
		}
		w.str("alpn", strings.Join(protos, "-"))
	}
	if len(h.ALPS) > 0 {
		protos := make([]string, len(h.ALPS))
		for i, p := range h.ALPS {
			protos[i] = escapeValue(p, ",=-")
		}
		w.str("alps", strings.Join(protos, "-"))
	}
	w.bool("ech_ext", h.ECH)
	return boundedPairs(string(w.b))
}

// h2Detail splits the Akamai fingerprint into its parts, and adds the priority and the flags of the first HEADERS
// frame, which the Akamai fingerprint leaves out.
func h2Detail(akamai string, prefix []byte) string {
	parts := strings.Split(akamai, "|")
	if len(parts) != 4 {
		return ""
	}

	w := transportWriter{b: make([]byte, 0, 256)}
	w.str("settings", parts[0])
	w.str("window_update", parts[1])
	w.str("priority", strings.ReplaceAll(parts[2], ",", "-"))
	if flags, priority, ok := firstHeaders(prefix); ok {
		w.str("headers_priority", priority)
		w.str("first_headers_flags", "0x"+strconv.FormatUint(uint64(flags), 16))
	}
	w.str("pseudo", strings.ReplaceAll(parts[3], ",", "-"))
	return boundedPairs(string(w.b))
}

// The QUIC transport parameters of RFC 9000 section 18.2 and of the common extensions, by identifier.
var quicTransportParameterNames = map[uint64]string{
	0x00:   "original_dcid",
	0x01:   "max_idle_timeout",
	0x02:   "stateless_reset_token",
	0x03:   "max_udp_payload",
	0x04:   "initial_max_data",
	0x05:   "initial_max_stream_data_bidi_local",
	0x06:   "initial_max_stream_data_bidi_remote",
	0x07:   "initial_max_stream_data_uni",
	0x08:   "initial_max_streams_bidi",
	0x09:   "initial_max_streams_uni",
	0x0a:   "ack_delay_exponent",
	0x0b:   "max_ack_delay",
	0x0c:   "disable_active_migration",
	0x0d:   "preferred_address",
	0x0e:   "active_connection_id_limit",
	0x0f:   "initial_scid",
	0x10:   "retry_scid",
	0x11:   "version_information",
	0x20:   "max_datagram_frame_size",
	0x2ab2: "grease_quic_bit",
}

// The transport parameters whose value is a variable-length integer.
var quicVarintParameters = map[uint64]bool{
	0x01: true, 0x03: true, 0x04: true, 0x05: true, 0x06: true, 0x07: true, 0x08: true, 0x09: true,
	0x0a: true, 0x0b: true, 0x0e: true, 0x20: true,
}

// quicTransportParametersDetail formats the QUIC transport parameters of a ClientHello message in the client
// order: name:value for an integer, name:<length>b for bytes, an empty value for a flag, and grease for the
// reserved identifiers.
func quicTransportParametersDetail(msg []byte) string {
	params, err := quicTransportParameters(msg)
	if err != nil {
		return ""
	}

	items := make([]string, 0, len(params))
	for _, p := range params {
		if p.id >= 27 && (p.id-27)%31 == 0 {
			items = append(items, "grease")
			continue
		}
		name, ok := quicTransportParameterNames[p.id]
		if !ok {
			name = "0x" + strconv.FormatUint(p.id, 16)
		}
		switch {
		case quicVarintParameters[p.id]:
			if v, n := quicVarint(p.value); n == len(p.value) && n > 0 {
				items = append(items, name+":"+strconv.FormatUint(v, 10))
				continue
			}
			items = append(items, name+":"+strconv.Itoa(len(p.value))+"b")
		case len(p.value) == 0:
			items = append(items, name)
		default:
			items = append(items, name+":"+strconv.Itoa(len(p.value))+"b")
		}
	}
	return boundedList(items, "-", maxDetailHeader-len("tp="))
}

// newConnDetail formats the detail header values of a connection once, when its record is computed.
func (e *EntryPoint) newConnDetail(info *connInfo, state tls.ConnectionState, hello *ClientHello, raw []byte, rec *connectionRecord) *connDetail {
	d := &connDetail{}

	if t := rec.TLS; t != nil {
		d.tls = tlsDetail(state, hello, len(raw), info.quic != nil)
		d.ja3String, d.ja3nString, d.ja4r, d.ja4o = t.JA3String, t.JA3NString, t.JA4R, t.JA4O
		if hello != nil {
			d.tlsHello = tlsHelloDetail(hello)
		}
		if info.quic != nil && len(raw) > 0 {
			d.tp = quicTransportParametersDetail(raw)
		}
		if e.headerDetail == detailRaw && len(raw) > 0 {
			if enc := base64.StdEncoding.EncodeToString(raw); len(enc) <= maxClientHelloHeader {
				d.clientHello = enc
			} else {
				d.tls += ",ch_raw=omitted"
			}
		}
	}

	if h := rec.HTTP2; h != nil && h.Akamai != "" && info.capture != nil {
		if prefix, _ := info.capture.H2Prefix(); prefix != nil {
			d.h2Detail = h2Detail(h.Akamai, prefix)
		}
	}

	if info.synView != nil {
		d.p0f = info.synView.P0F
		if e.headerDetail == detailRaw && len(info.syn) > 0 {
			d.syn = base64.StdEncoding.EncodeToString(info.syn)
		}
	}

	return d
}

// h3Detail formats the HTTP/3 SETTINGS and the QUIC transport parameters.
func h3Detail(info *connInfo) string {
	w := transportWriter{b: make([]byte, 0, 512)}
	if info.h3 != nil {
		w.str("settings", info.h3.Settings())
	}
	if d := info.detail; d != nil && d.tp != "" {
		w.str("tp", d.tp)
	}
	return boundedPairs(string(w.b))
}

// requestDetail returns the header order and the cookie names of a request, as read from the wire.
func requestDetail(r *Request) (string, string) {
	names := make([]string, len(r.Names))
	for i, n := range r.Names {
		names[i] = escapeValue(n, ",")
	}

	var cookies string
	if list := cookieNames(r.Cookies); len(list) > 0 {
		for i, c := range list {
			list[i] = escapeValue(c, ",")
		}
		cookies = boundedList(list, ",", maxCookieNames)
	}

	return boundedList(names, ",", maxHeaderOrder), cookies
}
