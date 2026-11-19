package fingerprint

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"strconv"
	"strings"
)

const (
	tlsRecordHeaderLen      = 5
	tlsRecordTypeHandshake  = 0x16
	tlsHandshakeClientHello = 0x01
	tlsHandshakeHeaderLen   = 4
	tlsExtServerName        = 0
	tlsExtSupportedGroups   = 10
	tlsExtPointFormats      = 11
	tlsExtSignatureAlgs     = 13
	tlsExtALPN              = 16
	tlsExtSupportedVersions = 43

	tlsExtCertCompression   = 27
	tlsExtRecordSizeLimit   = 28
	tlsExtPSKModes          = 45
	tlsExtKeyShare          = 51
	tlsExtALPSOld           = 17513
	tlsExtALPS              = 17613
	tlsExtEncryptedHello    = 0xfe0d
	tlsVersionUnknownMarker = "00"
	ja4EmptyHash            = "000000000000"
)

var errClientHelloTruncated = errors.New("truncated ClientHello")

// ClientHello holds the fields of a TLS ClientHello used by the fingerprints.
type ClientHello struct {
	LegacyVersion       uint16
	CipherSuites        []uint16
	Extensions          []uint16 // in received order, GREASE included
	ServerName          string
	ALPN                []string // in received order
	SupportedGroups     []uint16
	PointFormats        []uint8
	SignatureAlgorithms []uint16
	SupportedVersions   []uint16

	// The fields below are only read for the headers, and a malformed value leaves them empty.

	// KeyShareGroups are the groups of the key shares, in order, GREASE included.
	KeyShareGroups []uint16
	// PSKModes are the PSK key exchange modes.
	PSKModes []uint8
	// CertCompression are the certificate compression algorithms (RFC 8879).
	CertCompression []uint16
	// RecordSizeLimit is the record size limit (RFC 8449), -1 when absent.
	RecordSizeLimit int
	// ALPS are the protocols of the application settings extension, old or new code point.
	ALPS []string
	// ECH tells whether the ClientHello carries an encrypted client hello extension, real or GREASE.
	ECH bool
}

// isGREASE reports whether v is a GREASE value of RFC 8701, which clients send at random to keep servers tolerant,
// and which the fingerprints ignore for being random.
func isGREASE(v uint16) bool {
	return v&0x0f0f == 0x0a0a && v>>8 == v&0xff
}

// clientHelloMessage reassembles the ClientHello handshake message from the handshake records of raw.
// It returns the message, the count of bytes of raw up to the end of the record holding its last byte,
// and errClientHelloTruncated while raw does not hold the whole message.
func clientHelloMessage(raw []byte) ([]byte, int, error) {
	var msg []byte
	off := 0
	for {
		if len(raw)-off < tlsRecordHeaderLen {
			return nil, 0, errClientHelloTruncated
		}
		if raw[off] != tlsRecordTypeHandshake {
			return nil, 0, errors.New("not a TLS handshake record")
		}
		recLen := int(raw[off+3])<<8 | int(raw[off+4])
		if recLen == 0 {
			return nil, 0, errors.New("empty TLS handshake record")
		}
		off += tlsRecordHeaderLen
		if len(raw)-off < recLen {
			return nil, 0, errClientHelloTruncated
		}
		msg = append(msg, raw[off:off+recLen]...)
		off += recLen

		if len(msg) >= tlsHandshakeHeaderLen {
			if msg[0] != tlsHandshakeClientHello {
				return nil, 0, errors.New("not a ClientHello")
			}
			n := tlsHandshakeHeaderLen + (int(msg[1])<<16 | int(msg[2])<<8 | int(msg[3]))
			if len(msg) >= n {
				return msg[:n], off, nil
			}
		}
	}
}

// ClientHelloLen returns the count of bytes of raw, from the first record, up to the end of the record holding the
// end of the ClientHello message, and false while raw does not hold a whole ClientHello yet.
func ClientHelloLen(raw []byte) (int, bool) {
	_, n, err := clientHelloMessage(raw)
	if err != nil {
		return 0, false
	}

	return n, true
}

// ParseClientHello parses the raw bytes read from the client from the first TLS record:
// record headers included, the handshake message possibly fragmented over several records,
// and possibly followed by other bytes (ignored). It returns an error on a truncated or malformed ClientHello.
func ParseClientHello(raw []byte) (*ClientHello, error) {
	msg, _, err := clientHelloMessage(raw)
	if err != nil {
		return nil, err
	}

	return ParseClientHelloMessage(msg)
}

// reader reads big-endian fields, and records the first out-of-bounds read instead of panicking.
type reader struct {
	b   []byte
	err bool
}

func (r *reader) bytes(n int) []byte {
	if r.err || n < 0 || len(r.b) < n {
		r.err = true
		return nil
	}
	v := r.b[:n]
	r.b = r.b[n:]
	return v
}

func (r *reader) u8() int {
	b := r.bytes(1)
	if b == nil {
		return 0
	}
	return int(b[0])
}

func (r *reader) u16() int {
	b := r.bytes(2)
	if b == nil {
		return 0
	}
	return int(b[0])<<8 | int(b[1])
}

func (r *reader) u24() int {
	b := r.bytes(3)
	if b == nil {
		return 0
	}
	return int(b[0])<<16 | int(b[1])<<8 | int(b[2])
}

// vec reads a vector prefixed by a length of lenBytes bytes.
func (r *reader) vec(lenBytes int) *reader {
	var n int
	switch lenBytes {
	case 1:
		n = r.u8()
	case 2:
		n = r.u16()
	}
	b := r.bytes(n)
	return &reader{b: b, err: r.err}
}

func (r *reader) u16s() []uint16 {
	var v []uint16
	for len(r.b) >= 2 {
		v = append(v, uint16(r.u16()))
	}
	if len(r.b) != 0 {
		r.err = true
	}
	return v
}

// ParseClientHelloMessage parses a bare ClientHello handshake message (type, 24-bit length and body),
// as carried by the TLS records or by the CRYPTO frames of a QUIC Initial packet.
func ParseClientHelloMessage(msg []byte) (*ClientHello, error) {
	r := &reader{b: msg}
	if r.u8() != tlsHandshakeClientHello {
		return nil, errors.New("not a ClientHello")
	}
	body := &reader{b: r.bytes(r.u24())}
	if r.err {
		return nil, errClientHelloTruncated
	}

	h := &ClientHello{LegacyVersion: uint16(body.u16()), RecordSizeLimit: -1}
	body.bytes(32)
	body.vec(1)
	h.CipherSuites = body.vec(2).u16s()
	body.vec(1)
	if body.err {
		return nil, errors.New("malformed ClientHello")
	}

	// The extensions block is optional (RFC 5246 section 7.4.1.2).
	if len(body.b) == 0 {
		return h, nil
	}

	exts := body.vec(2)
	if body.err {
		return nil, errors.New("malformed ClientHello extensions")
	}
	for len(exts.b) > 0 {
		typ := uint16(exts.u16())
		data := exts.vec(2)
		if exts.err {
			return nil, errors.New("malformed ClientHello extension")
		}
		h.Extensions = append(h.Extensions, typ)

		var bad bool
		switch typ {
		case tlsExtServerName:
			names := data.vec(2)
			for len(names.b) > 0 {
				nameType := names.u8()
				name := names.vec(2)
				if nameType == 0 && h.ServerName == "" {
					h.ServerName = string(name.b)
				}
				bad = bad || names.err
				if names.err {
					break
				}
			}
			bad = bad || data.err
		case tlsExtALPN:
			protos := data.vec(2)
			for len(protos.b) > 0 {
				proto := protos.vec(1)
				if protos.err {
					break
				}
				h.ALPN = append(h.ALPN, string(proto.b))
			}
			bad = data.err || protos.err
		case tlsExtSupportedGroups:
			list := data.vec(2)
			h.SupportedGroups = list.u16s()
			bad = data.err || list.err
		case tlsExtPointFormats:
			list := data.vec(1)
			h.PointFormats = slices.Clone(list.b)
			bad = data.err
		case tlsExtSignatureAlgs:
			list := data.vec(2)
			h.SignatureAlgorithms = list.u16s()
			bad = data.err || list.err
		case tlsExtSupportedVersions:
			list := data.vec(1)
			h.SupportedVersions = list.u16s()
			bad = data.err || list.err
		default:
			h.parseExtra(typ, data)
		}
		if bad {
			return nil, errors.New("malformed ClientHello extension " + strconv.Itoa(int(typ)))
		}
	}

	return h, nil
}

func withoutGREASE(v []uint16) []uint16 {
	out := make([]uint16, 0, len(v))
	for _, x := range v {
		if !isGREASE(x) {
			out = append(out, x)
		}
	}
	return out
}

func joinDecimal(b *strings.Builder, v []uint16) {
	for i, x := range v {
		if i > 0 {
			b.WriteByte('-')
		}
		b.WriteString(strconv.Itoa(int(x)))
	}
}

// JA3 returns the JA3 string of the ClientHello (github.com/salesforce/ja3), with the GREASE values removed.
func (h *ClientHello) JA3() string {
	return h.ja3(withoutGREASE(h.Extensions))
}

// JA3Hash returns the MD5 of the JA3 string, which is the usual form of JA3.
func (h *ClientHello) JA3Hash() string {
	return md5Hex(h.JA3())
}

// JA3N returns the JA3 string with the extensions sorted, which stays the same when a client shuffles its extensions
// on each connection, as Chrome does since version 110.
func (h *ClientHello) JA3N() string {
	exts := withoutGREASE(h.Extensions)
	slices.Sort(exts)
	return h.ja3(exts)
}

// JA3NHash returns the MD5 of the JA3N string.
func (h *ClientHello) JA3NHash() string {
	return md5Hex(h.JA3N())
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func ja4Version(v uint16) string {
	switch v {
	case 0x0304:
		return "13"
	case 0x0303:
		return "12"
	case 0x0302:
		return "11"
	case 0x0301:
		return "10"
	case 0x0300:
		return "s3"
	case 0x0002:
		return "s2"
	default:
		return tlsVersionUnknownMarker
	}
}

// ja4ALPNChars mirrors the reference implementation, which replaces a non-ASCII byte with '9'.
func ja4ALPNChars(alpn string) string {
	if alpn == "" {
		return "00"
	}
	// The reference decodes the protocol as UTF-8 and takes the first and last characters.
	runes := []rune(alpn)
	c := func(r rune) byte {
		if r > 0x7f {
			return '9'
		}
		return byte(r)
	}
	return string([]byte{c(runes[0]), c(runes[len(runes)-1])})
}

func hex4List(v []uint16) string {
	var b strings.Builder
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		const digits = "0123456789abcdef"
		b.Write([]byte{digits[x>>12], digits[x>>8&0xf], digits[x>>4&0xf], digits[x&0xf]})
	}
	return b.String()
}

func ja4Hash12(s string) string {
	if s == "" {
		return ja4EmptyHash
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:6])
}

// JA4 returns the FoxIO JA4 fingerprint of the ClientHello (github.com/FoxIO-LLC/ja4, rust/ja4/src/tls.rs).
func (h *ClientHello) JA4(quic bool) string {
	return h.JA4Variants(quic).JA4
}

// JA4Variants are the forms of JA4 the FoxIO tool prints: sorted or in the client order, hashed or raw.
type JA4Variants struct {
	// JA4 is sorted and hashed.
	JA4 string
	// R (JA4_r) is sorted and raw.
	R string
	// O (JA4_o) is in the client order and hashed: the order of the extensions tells apart clients
	// with the same JA4, but a client shuffling its extensions gets a new JA4_o on each connection.
	O string
}

// JA4Variants returns the JA4, JA4_r and JA4_o fingerprints of the ClientHello.
func (h *ClientHello) JA4Variants(quic bool) JA4Variants {
	var b strings.Builder
	if quic {
		b.WriteByte('q')
	} else {
		b.WriteByte('t')
	}

	version := h.LegacyVersion
	if slices.Contains(h.Extensions, tlsExtSupportedVersions) {
		version = 0
		for _, v := range h.SupportedVersions {
			if !isGREASE(v) && v > version {
				version = v
			}
		}
	}
	b.WriteString(ja4Version(version))

	exts := withoutGREASE(h.Extensions)
	if slices.Contains(exts, tlsExtServerName) {
		b.WriteByte('d')
	} else {
		b.WriteByte('i')
	}

	ciphers := withoutGREASE(h.CipherSuites)
	b.WriteString(twoDigits(len(ciphers)))
	b.WriteString(twoDigits(len(exts)))

	alpn := ""
	if len(h.ALPN) > 0 {
		alpn = h.ALPN[0]
	}
	b.WriteString(ja4ALPNChars(alpn))
	prefix := b.String()

	sigs := hex4List(withoutGREASE(h.SignatureAlgorithms))
	// The specification ends the string without an underscore when there are no signature algorithms.
	withSigs := func(exts string) string {
		if sigs == "" {
			return exts
		}
		return exts + "_" + sigs
	}

	// The client order keeps the server name and ALPN extensions, which the sorted form leaves out.
	origCiphers := hex4List(ciphers)
	origExts := withSigs(hex4List(exts))

	sortedCiphers := slices.Clone(ciphers)
	slices.Sort(sortedCiphers)
	sortedExts := make([]uint16, 0, len(exts))
	for _, e := range exts {
		if e != tlsExtServerName && e != tlsExtALPN {
			sortedExts = append(sortedExts, e)
		}
	}
	slices.Sort(sortedExts)
	ciphersHex := hex4List(sortedCiphers)
	extsHex := withSigs(hex4List(sortedExts))

	return JA4Variants{
		JA4: prefix + "_" + ja4Hash12(ciphersHex) + "_" + ja4Hash12(extsHex),
		R:   prefix + "_" + ciphersHex + "_" + extsHex,
		O:   prefix + "_" + ja4Hash12(origCiphers) + "_" + ja4Hash12(origExts),
	}
}

// parseExtra reads the extensions only the headers use; a malformed one leaves its field empty.
func (h *ClientHello) parseExtra(typ uint16, data *reader) {
	switch typ {
	case tlsExtKeyShare:
		shares := data.vec(2)
		var groups []uint16
		for len(shares.b) > 0 && !shares.err {
			groups = append(groups, uint16(shares.u16()))
			shares.vec(2)
		}
		if !shares.err && !data.err {
			h.KeyShareGroups = groups
		}
	case tlsExtPSKModes:
		if list := data.vec(1); !data.err {
			h.PSKModes = slices.Clone(list.b)
		}
	case tlsExtCertCompression:
		list := data.vec(1)
		if algs := list.u16s(); !data.err && !list.err {
			h.CertCompression = algs
		}
	case tlsExtRecordSizeLimit:
		if v := data.u16(); !data.err {
			h.RecordSizeLimit = v
		}
	case tlsExtALPS, tlsExtALPSOld:
		protos := data.vec(2)
		var list []string
		for len(protos.b) > 0 && !protos.err {
			list = append(list, string(protos.vec(1).b))
		}
		if !protos.err && !data.err {
			h.ALPS = list
		}
	case tlsExtEncryptedHello:
		h.ECH = true
	}
}

func (h *ClientHello) ja3(exts []uint16) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(int(h.LegacyVersion)))
	b.WriteByte(',')
	joinDecimal(&b, withoutGREASE(h.CipherSuites))
	b.WriteByte(',')
	joinDecimal(&b, exts)
	b.WriteByte(',')
	joinDecimal(&b, withoutGREASE(h.SupportedGroups))
	b.WriteByte(',')
	for i, f := range h.PointFormats {
		if i > 0 {
			b.WriteByte('-')
		}
		b.WriteString(strconv.Itoa(int(f)))
	}

	return b.String()
}

func twoDigits(n int) string {
	n = min(n, 99)
	return string([]byte{byte('0' + n/10), byte('0' + n%10)})
}
