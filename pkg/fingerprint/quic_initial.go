package fingerprint

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
)

// The client Initial packets of QUIC are protected with keys derived from public values (RFC 9001 section 5.2),
// so that any observer can read the ClientHello they carry: decrypting them is how the raw ClientHello and the
// transport parameters in their received order are recovered, which quic-go does not expose.

const (
	quicVersion1 = 0x00000001
	quicVersion2 = 0x6b3343cf

	quicFramePadding         = 0x00
	quicFramePing            = 0x01
	quicFrameACK             = 0x02
	quicFrameACKECN          = 0x03
	quicFrameCrypto          = 0x06
	quicFrameConnectionClose = 0x1c

	tlsExtQUICTransportParameters = 0x39

	quicSampleLen = 16
)

var (
	quicSaltV1 = []byte{0x38, 0x76, 0x2c, 0xf7, 0xf5, 0x59, 0x34, 0xb3, 0x4d, 0x17, 0x9a, 0xe6, 0xa4, 0xc8, 0x0c, 0xad, 0xcc, 0xbb, 0x7f, 0x0a}
	quicSaltV2 = []byte{0x0d, 0xed, 0xe3, 0xde, 0xf7, 0x00, 0xa6, 0xdb, 0x81, 0x93, 0x81, 0xbe, 0x6e, 0x26, 0x9d, 0xcb, 0xf9, 0xbd, 0x2e, 0xd9}

	errQUICTruncated = errors.New("truncated QUIC packet")
)

// QUICCryptoFrame is the data of a CRYPTO frame of an Initial packet.
type QUICCryptoFrame struct {
	Offset uint64
	Data   []byte
}

type quicInitialKeys struct {
	aead cipher.AEAD
	iv   []byte
	hp   cipher.Block
}

// hkdfExpandLabel is the TLS 1.3 HKDF-Expand-Label (RFC 8446 section 7.1) with an empty context.
func hkdfExpandLabel(secret []byte, label string, length int) ([]byte, error) {
	full := "tls13 " + label
	info := make([]byte, 0, 4+len(full))
	info = binary.BigEndian.AppendUint16(info, uint16(length))
	info = append(info, byte(len(full)))
	info = append(info, full...)
	info = append(info, 0)

	return hkdf.Expand(sha256.New, secret, string(info), length)
}

// quicInitialSecrets derives the client Initial packet key, IV and header protection key.
func quicInitialSecrets(version uint32, dcid []byte) ([]byte, []byte, []byte, error) {
	salt, prefix := quicSaltV1, "quic "
	if version == quicVersion2 {
		salt, prefix = quicSaltV2, "quicv2 "
	}

	initial, err := hkdf.Extract(sha256.New, dcid, salt)
	if err != nil {
		return nil, nil, nil, err
	}
	client, err := hkdfExpandLabel(initial, "client in", 32)
	if err != nil {
		return nil, nil, nil, err
	}
	key, err := hkdfExpandLabel(client, prefix+"key", 16)
	if err != nil {
		return nil, nil, nil, err
	}
	iv, err := hkdfExpandLabel(client, prefix+"iv", 12)
	if err != nil {
		return nil, nil, nil, err
	}
	hpKey, err := hkdfExpandLabel(client, prefix+"hp", 16)
	if err != nil {
		return nil, nil, nil, err
	}

	return key, iv, hpKey, nil
}

func newQUICInitialKeys(version uint32, dcid []byte) (*quicInitialKeys, error) {
	key, iv, hpKey, err := quicInitialSecrets(version, dcid)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	hp, err := aes.NewCipher(hpKey)
	if err != nil {
		return nil, err
	}

	return &quicInitialKeys{aead: aead, iv: iv, hp: hp}, nil
}

// quicVarint reads a variable-length integer (RFC 9000 section 16), and returns its size, or 0 when truncated.
func quicVarint(b []byte) (uint64, int) {
	if len(b) == 0 {
		return 0, 0
	}
	n := 1 << (b[0] >> 6)
	if len(b) < n {
		return 0, 0
	}
	v := uint64(b[0] & 0x3f)
	for _, c := range b[1:n] {
		v = v<<8 | uint64(c)
	}
	return v, n
}

func isQUICInitial(version uint32, first byte) bool {
	typ := (first >> 4) & 0x03
	switch version {
	case quicVersion1:
		return typ == 0
	case quicVersion2:
		return typ == 1
	default:
		return false
	}
}

// DecryptQUICInitials decrypts the client Initial packets of one UDP datagram (coalesced packets included,
// non-Initial packets skipped) with the keys derived from their destination connection ID
// (RFC 9001 section 5.2 for QUIC v1, RFC 9369 for v2), and returns the destination connection ID of the first
// Initial packet and the CRYPTO frames found. It never modifies datagram.
func DecryptQUICInitials(datagram []byte) ([]byte, []QUICCryptoFrame, error) {
	var (
		dcid    []byte
		frames  []QUICCryptoFrame
		lastErr error
	)

	b := datagram
	for len(b) > 0 {
		// A short header packet carries no length: it ends the datagram, and is not an Initial packet.
		if b[0]&0x80 == 0 {
			break
		}

		pkt, rest, err := parseQUICLongHeader(b)
		if err != nil {
			lastErr = err
			break
		}
		b = rest

		if !pkt.initial {
			continue
		}
		if dcid == nil {
			dcid = slices.Clone(pkt.dcid)
		}

		got, err := pkt.decrypt()
		if err != nil {
			// The Initial packets sent after the server answered use a connection ID of the server,
			// while their keys stay derived from the first one: they are not needed for the ClientHello.
			lastErr = err
			continue
		}
		frames = append(frames, got...)
	}

	if len(frames) == 0 && lastErr != nil {
		return dcid, nil, lastErr
	}

	return dcid, frames, nil
}

type quicLongPacket struct {
	version uint32
	initial bool
	dcid    []byte
	// packet is the whole packet, header included, and pnOffset the offset of its packet number.
	packet   []byte
	pnOffset int
}

func parseQUICLongHeader(b []byte) (*quicLongPacket, []byte, error) {
	if len(b) < 7 {
		return nil, nil, errQUICTruncated
	}

	p := &quicLongPacket{version: binary.BigEndian.Uint32(b[1:5])}
	if p.version == 0 {
		return nil, nil, errors.New("QUIC version negotiation packet")
	}
	if p.version != quicVersion1 && p.version != quicVersion2 {
		return nil, nil, fmt.Errorf("unsupported QUIC version %#x", p.version)
	}
	p.initial = isQUICInitial(p.version, b[0])

	off := 5
	dcidLen := int(b[off])
	off++
	if dcidLen > 20 || len(b) < off+dcidLen+1 {
		return nil, nil, errQUICTruncated
	}
	p.dcid = b[off : off+dcidLen]
	off += dcidLen

	scidLen := int(b[off])
	off++
	if scidLen > 20 || len(b) < off+scidLen {
		return nil, nil, errQUICTruncated
	}
	off += scidLen

	typ := (b[0] >> 4) & 0x03
	retry := (p.version == quicVersion1 && typ == 3) || (p.version == quicVersion2 && typ == 0)
	if retry {
		// A Retry packet has no length field, and is never sent by a client.
		return nil, nil, errors.New("QUIC retry packet")
	}

	if p.initial {
		tokenLen, n := quicVarint(b[off:])
		if n == 0 || uint64(len(b)-off-n) < tokenLen {
			return nil, nil, errQUICTruncated
		}
		off += n + int(tokenLen)
	}

	length, n := quicVarint(b[off:])
	if n == 0 {
		return nil, nil, errQUICTruncated
	}
	off += n
	if uint64(len(b)-off) < length {
		return nil, nil, errQUICTruncated
	}

	end := off + int(length)
	p.packet = b[:end]
	p.pnOffset = off

	return p, b[end:], nil
}

// decrypt removes the header protection and the packet protection, working on a copy of the packet.
func (p *quicLongPacket) decrypt() ([]QUICCryptoFrame, error) {
	if len(p.packet) < p.pnOffset+4+quicSampleLen {
		return nil, errQUICTruncated
	}

	keys, err := newQUICInitialKeys(p.version, p.dcid)
	if err != nil {
		return nil, err
	}

	pkt := slices.Clone(p.packet)

	var mask [aes.BlockSize]byte
	keys.hp.Encrypt(mask[:], pkt[p.pnOffset+4:p.pnOffset+4+quicSampleLen])

	pkt[0] ^= mask[0] & 0x0f
	pnLen := int(pkt[0]&0x03) + 1

	var pn uint64
	for i := range pnLen {
		pkt[p.pnOffset+i] ^= mask[1+i]
		pn = pn<<8 | uint64(pkt[p.pnOffset+i])
	}

	nonce := slices.Clone(keys.iv)
	for i := range 8 {
		nonce[len(nonce)-1-i] ^= byte(pn >> (8 * i))
	}

	header := pkt[:p.pnOffset+pnLen]
	payload, err := keys.aead.Open(nil, nonce, pkt[p.pnOffset+pnLen:], header)
	if err != nil {
		return nil, fmt.Errorf("decrypting QUIC Initial packet: %w", err)
	}

	return parseQUICInitialFrames(payload)
}

// parseQUICInitialFrames reads the frames an Initial packet may carry (RFC 9000 section 12.4), keeping the CRYPTO ones.
func parseQUICInitialFrames(b []byte) ([]QUICCryptoFrame, error) {
	var frames []QUICCryptoFrame

	for len(b) > 0 {
		typ, n := quicVarint(b)
		if n == 0 {
			return nil, errQUICTruncated
		}
		b = b[n:]

		switch typ {
		case quicFramePadding, quicFramePing:
		case quicFrameACK, quicFrameACKECN:
			// Largest acknowledged, delay, range count and first range, then the ranges.
			var vals [4]uint64
			for i := range vals {
				v, n := quicVarint(b)
				if n == 0 {
					return nil, errQUICTruncated
				}
				vals[i] = v
				b = b[n:]
			}
			fields := vals[2] * 2
			if typ == quicFrameACKECN {
				fields += 3
			}
			if fields > uint64(len(b)) {
				return nil, errQUICTruncated
			}
			for range fields {
				_, n := quicVarint(b)
				if n == 0 {
					return nil, errQUICTruncated
				}
				b = b[n:]
			}
		case quicFrameCrypto:
			offset, n := quicVarint(b)
			if n == 0 {
				return nil, errQUICTruncated
			}
			b = b[n:]
			length, n := quicVarint(b)
			if n == 0 || uint64(len(b)-n) < length {
				return nil, errQUICTruncated
			}
			b = b[n:]
			frames = append(frames, QUICCryptoFrame{Offset: offset, Data: b[:length]})
			b = b[length:]
		case quicFrameConnectionClose:
			// Error code, frame type and reason phrase.
			for range 2 {
				_, n := quicVarint(b)
				if n == 0 {
					return nil, errQUICTruncated
				}
				b = b[n:]
			}
			reasonLen, n := quicVarint(b)
			if n == 0 || uint64(len(b)-n) < reasonLen {
				return nil, errQUICTruncated
			}
			b = b[n+int(reasonLen):]
		default:
			return nil, fmt.Errorf("unexpected QUIC frame type %#x in an Initial packet", typ)
		}
	}

	return frames, nil
}

// QUICClientHelloAssembler reassembles the CRYPTO data of a connection, received out of order as Chrome sends it,
// until the ClientHello handshake message is complete. Its memory is bounded by maxClientHello.
type QUICClientHelloAssembler struct {
	buf []byte
	// ranges are the received byte ranges of buf, sorted and merged.
	ranges [][2]int
	msg    []byte
}

// Add adds a CRYPTO frame, and returns the bare ClientHello handshake message once it is complete.
func (a *QUICClientHelloAssembler) Add(f QUICCryptoFrame) ([]byte, bool, error) {
	if a.msg != nil {
		return a.msg, true, nil
	}

	if f.Offset > maxClientHello || uint64(len(f.Data)) > maxClientHello-f.Offset {
		return nil, false, fmt.Errorf("QUIC ClientHello longer than %d bytes", maxClientHello)
	}
	if len(f.Data) == 0 {
		return nil, false, nil
	}

	start := int(f.Offset)
	end := start + len(f.Data)
	if end > len(a.buf) {
		a.buf = append(a.buf, make([]byte, end-len(a.buf))...)
	}
	copy(a.buf[start:end], f.Data)
	a.addRange(start, end)

	if len(a.ranges) == 0 || a.ranges[0][0] != 0 || a.ranges[0][1] < 4 {
		return nil, false, nil
	}

	if a.buf[0] != tlsHandshakeClientHello {
		return nil, false, errors.New("the QUIC CRYPTO stream does not start with a ClientHello")
	}
	total := 4 + (int(a.buf[1])<<16 | int(a.buf[2])<<8 | int(a.buf[3]))
	if total > maxClientHello {
		return nil, false, fmt.Errorf("QUIC ClientHello longer than %d bytes", maxClientHello)
	}
	if a.ranges[0][1] < total {
		return nil, false, nil
	}

	a.msg = a.buf[:total:total]
	a.ranges = nil

	return a.msg, true, nil
}

func (a *QUICClientHelloAssembler) addRange(start, end int) {
	a.ranges = append(a.ranges, [2]int{start, end})
	slices.SortFunc(a.ranges, func(x, y [2]int) int { return x[0] - y[0] })

	merged := a.ranges[:1]
	for _, r := range a.ranges[1:] {
		last := &merged[len(merged)-1]
		if r[0] <= last[1] {
			last[1] = max(last[1], r[1])
			continue
		}
		merged = append(merged, r)
	}
	a.ranges = merged
}

// QUICTransportParameterIDs returns the IDs of the quic_transport_parameters extension of a bare ClientHello message,
// in the received order, GREASE included.
func QUICTransportParameterIDs(msg []byte) ([]uint64, error) {
	r := &reader{b: msg}
	if r.u8() != tlsHandshakeClientHello {
		return nil, errors.New("not a ClientHello")
	}
	body := &reader{b: r.bytes(r.u24())}
	body.bytes(2 + 32)
	body.vec(1)
	body.vec(2)
	body.vec(1)
	exts := body.vec(2)
	if r.err || body.err {
		return nil, errors.New("malformed ClientHello")
	}

	for len(exts.b) > 0 {
		typ := exts.u16()
		data := exts.vec(2)
		if exts.err {
			return nil, errors.New("malformed ClientHello extension")
		}
		if typ != tlsExtQUICTransportParameters {
			continue
		}

		var ids []uint64
		b := data.b
		for len(b) > 0 {
			id, n := quicVarint(b)
			if n == 0 {
				return nil, errors.New("malformed QUIC transport parameter")
			}
			b = b[n:]
			length, n := quicVarint(b)
			if n == 0 || uint64(len(b)-n) < length {
				return nil, errors.New("malformed QUIC transport parameter")
			}
			b = b[n+int(length):]
			ids = append(ids, id)
		}
		return ids, nil
	}

	return nil, errors.New("no QUIC transport parameters extension")
}
