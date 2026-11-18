package fingerprint

import (
	"context"
	"crypto/aes"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"net"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}

func TestQUICInitialSecrets(t *testing.T) {
	// RFC 9001 appendix A.1 and RFC 9369 appendix A.1, for the destination connection ID 0x8394c8f03e515708.
	testCases := []struct {
		desc    string
		version uint32
		key     string
		iv      string
		hp      string
	}{
		{desc: "v1", version: quicVersion1, key: "1f369613dd76d5467730efcbe3b1a22d", iv: "fa044b2f42a3fd3b46fb255c", hp: "9f50449e04a0e810283a1e9933adedd2"},
		{desc: "v2", version: quicVersion2, key: "8b1a0bc121284290a29e0971b5cd045d", iv: "91f73e2351d8fa91660e909f", hp: "45b95e15235d6f45a6b19cbcb0294ba9"},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			key, iv, hp, err := quicInitialSecrets(test.version, mustHex(t, "8394c8f03e515708"))
			require.NoError(t, err)

			assert.Equal(t, test.key, hex.EncodeToString(key))
			assert.Equal(t, test.iv, hex.EncodeToString(iv))
			assert.Equal(t, test.hp, hex.EncodeToString(hp))
		})
	}
}

func appendVarint(b []byte, v uint64) []byte {
	switch {
	case v < 1<<6:
		return append(b, byte(v))
	case v < 1<<14:
		return binary.BigEndian.AppendUint16(b, uint16(v)|0x4000)
	case v < 1<<30:
		return binary.BigEndian.AppendUint32(b, uint32(v)|0x80000000)
	default:
		return binary.BigEndian.AppendUint64(b, v|0xc0<<56)
	}
}

func cryptoFrame(offset uint64, data []byte) []byte {
	b := appendVarint([]byte{quicFrameCrypto}, offset)
	b = appendVarint(b, uint64(len(data)))
	return append(b, data...)
}

// sealInitial builds a protected client Initial packet, the inverse of the decryption under test.
func sealInitial(t *testing.T, version uint32, dcid []byte, pn uint32, payload []byte) []byte {
	t.Helper()

	keys, err := newQUICInitialKeys(version, dcid)
	require.NoError(t, err)

	typ := byte(0)
	if version == quicVersion2 {
		typ = 1
	}
	// A 4-byte packet number, so that the header protection sample is always inside the packet.
	hdr := []byte{0xc0 | typ<<4 | 0x03}
	hdr = binary.BigEndian.AppendUint32(hdr, version)
	hdr = append(hdr, byte(len(dcid)))
	hdr = append(hdr, dcid...)
	hdr = append(hdr, 0) // Empty source connection ID.
	hdr = append(hdr, 0) // Empty token.
	hdr = appendVarint(hdr, uint64(4+len(payload)+keys.aead.Overhead()))
	pnOffset := len(hdr)
	hdr = binary.BigEndian.AppendUint32(hdr, pn)

	nonce := slices.Clone(keys.iv)
	for i := range 4 {
		nonce[len(nonce)-1-i] ^= byte(pn >> (8 * i))
	}
	pkt := keys.aead.Seal(slices.Clone(hdr), nonce, payload, hdr)

	var mask [aes.BlockSize]byte
	keys.hp.Encrypt(mask[:], pkt[pnOffset+4:pnOffset+4+quicSampleLen])
	pkt[0] ^= mask[0] & 0x0f
	for i := range 4 {
		pkt[pnOffset+i] ^= mask[1+i]
	}

	return pkt
}

func TestDecryptQUICInitials_coalescedAndSplit(t *testing.T) {
	msg := append([]byte{tlsHandshakeClientHello, 0, 0x03, 0xe8}, make([]byte, 1000)...)
	for i := range msg[4:] {
		msg[4+i] = byte(i)
	}
	dcid := mustHex(t, "0011223344556677")

	for _, version := range []uint32{quicVersion1, quicVersion2} {
		// The second half first, as Chrome does, then the first half coalesced with a Handshake packet to skip.
		second := sealInitial(t, version, dcid, 1, append(cryptoFrame(600, msg[600:]), make([]byte, 40)...))
		first := sealInitial(t, version, dcid, 0, append([]byte{quicFramePing}, cryptoFrame(0, msg[:600])...))

		handshakeType := byte(2)
		if version == quicVersion2 {
			handshakeType = 3
		}
		handshake := []byte{0xc0 | handshakeType<<4}
		handshake = binary.BigEndian.AppendUint32(handshake, version)
		handshake = append(handshake, 0, 0, 25)
		handshake = append(handshake, make([]byte, 25)...)

		datagram1 := slices.Clone(second)
		datagram2 := append(slices.Clone(first), handshake...)
		original2 := slices.Clone(datagram2)

		var a QUICClientHelloAssembler

		gotDCID, frames, err := DecryptQUICInitials(datagram1)
		require.NoError(t, err)
		assert.Equal(t, dcid, gotDCID)
		require.Len(t, frames, 1)
		_, done, err := a.Add(frames[0])
		require.NoError(t, err)
		assert.False(t, done)

		_, frames, err = DecryptQUICInitials(datagram2)
		require.NoError(t, err)
		assert.Equal(t, original2, datagram2)
		require.Len(t, frames, 1)
		got, done, err := a.Add(frames[0])
		require.NoError(t, err)
		assert.True(t, done)
		assert.Equal(t, msg, got)
	}
}

func TestQUICClientHelloAssembler_limits(t *testing.T) {
	var a QUICClientHelloAssembler
	_, _, err := a.Add(QUICCryptoFrame{Offset: maxClientHello - 10, Data: make([]byte, 20)})
	require.Error(t, err)

	// A ClientHello announcing a length past the bound fails once its header is known.
	var b QUICClientHelloAssembler
	_, _, err = b.Add(QUICCryptoFrame{Offset: 0, Data: []byte{tlsHandshakeClientHello, 0, 0x50, 0}})
	require.ErrorContains(t, err, "longer than")

	var c QUICClientHelloAssembler
	_, _, err = c.Add(QUICCryptoFrame{Offset: 0, Data: []byte{0x02, 0, 0, 1, 0}})
	require.ErrorContains(t, err, "does not start with a ClientHello")

	// Overlapping and duplicated frames, in any order.
	msg := append([]byte{tlsHandshakeClientHello, 0, 0, 10}, []byte("0123456789")...)
	var d QUICClientHelloAssembler
	for _, f := range []QUICCryptoFrame{{Offset: 6, Data: msg[6:]}, {Offset: 2, Data: msg[2:8]}, {Offset: 6, Data: msg[6:9]}} {
		_, done, err := d.Add(f)
		require.NoError(t, err)
		assert.False(t, done)
	}
	got, done, err := d.Add(QUICCryptoFrame{Offset: 0, Data: msg[:3]})
	require.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, msg, got)
}

func TestDecryptQUICInitials_errors(t *testing.T) {
	testCases := []struct {
		desc     string
		datagram []byte
	}{
		{desc: "short header", datagram: []byte{0x40, 1, 2, 3}},
		{desc: "truncated long header", datagram: []byte{0xc0, 0, 0, 0, 1}},
		{desc: "unknown version", datagram: append([]byte{0xc0, 0xff, 0, 0, 0x1d}, make([]byte, 40)...)},
		{desc: "version negotiation", datagram: append([]byte{0xc0, 0, 0, 0, 0}, make([]byte, 40)...)},
		{desc: "bad authentication", datagram: func() []byte {
			pkt := sealInitial(t, quicVersion1, []byte{1, 2, 3, 4, 5, 6, 7, 8}, 0, append(cryptoFrame(0, []byte("hello")), make([]byte, 30)...))
			pkt[len(pkt)-1] ^= 1
			return pkt
		}()},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			_, frames, _ := DecryptQUICInitials(test.datagram)
			assert.Empty(t, frames)
		})
	}
}

// recordingConn records the datagrams a client sends, and never answers.
type recordingConn struct {
	net.PacketConn

	mu        sync.Mutex
	datagrams [][]byte
}

func (c *recordingConn) record(ctx context.Context) {
	buf := make([]byte, 65536)
	for ctx.Err() == nil {
		_ = c.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		n, _, err := c.ReadFrom(buf)
		if err != nil {
			continue
		}
		c.mu.Lock()
		c.datagrams = append(c.datagrams, slices.Clone(buf[:n]))
		c.mu.Unlock()
	}
}

func recordQUICClient(t *testing.T, version quic.Version) [][]byte {
	t.Helper()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	rc := &recordingConn{PacketConn: pc}
	t.Cleanup(func() { _ = pc.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); rc.record(ctx) }()

	dialCtx, dialCancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer dialCancel()
	_, _ = quic.DialAddr(dialCtx, pc.LocalAddr().String(),
		&tls.Config{ServerName: "example.com", NextProtos: []string{"h3"}},
		&quic.Config{Versions: []quic.Version{version}})

	cancel()
	<-done

	rc.mu.Lock()
	defer rc.mu.Unlock()
	require.NotEmpty(t, rc.datagrams)
	return rc.datagrams
}

func TestDecryptQUICInitials_quicGoClient(t *testing.T) {
	for _, version := range []quic.Version{quic.Version1, quic.Version2} {
		t.Run(version.String(), func(t *testing.T) {
			datagrams := recordQUICClient(t, version)

			var a QUICClientHelloAssembler
			var msg []byte
			for _, d := range datagrams {
				_, frames, err := DecryptQUICInitials(d)
				require.NoError(t, err)
				for _, f := range frames {
					got, done, err := a.Add(f)
					require.NoError(t, err)
					if done {
						msg = got
					}
				}
				if msg != nil {
					break
				}
			}
			require.NotNil(t, msg)

			hello, err := ParseClientHelloMessage(msg)
			require.NoError(t, err)
			assert.Equal(t, "example.com", hello.ServerName)
			assert.Equal(t, []string{"h3"}, hello.ALPN)
			assert.Contains(t, hello.Extensions, uint16(tlsExtQUICTransportParameters))
			assert.Regexp(t, `^q13d\d{4}h3_`, hello.JA4(true))

			ids, err := QUICTransportParameterIDs(msg)
			require.NoError(t, err)
			// initial_max_data, initial_max_stream_data_bidi_local, initial_source_connection_id.
			assert.Subset(t, ids, []uint64{0x04, 0x05, 0x0f})
		})
	}
}

func TestQUICTransportParameterIDs(t *testing.T) {
	// A transport parameters extension with a GREASE ID (31 * 3 + 27), kept in order.
	params := appendVarint(nil, 0x04)
	params = appendVarint(params, 2)
	params = append(params, 0x40, 0x10)
	params = appendVarint(params, 31*3+27)
	params = appendVarint(params, 0)
	params = appendVarint(params, 0x0f)
	params = appendVarint(params, 1)
	params = append(params, 7)

	ext := binary.BigEndian.AppendUint16(nil, tlsExtQUICTransportParameters)
	ext = binary.BigEndian.AppendUint16(ext, uint16(len(params)))
	ext = append(ext, params...)

	body := binary.BigEndian.AppendUint16(nil, 0x0303)
	body = append(body, make([]byte, 32)...)
	body = append(body, 0)                // Session ID.
	body = append(body, 0, 2, 0x13, 0x01) // Cipher suites.
	body = append(body, 1, 0)             // Compression methods.
	body = binary.BigEndian.AppendUint16(body, uint16(len(ext)))
	body = append(body, ext...)

	msg := []byte{tlsHandshakeClientHello, 0, byte(len(body) >> 8), byte(len(body))}
	msg = append(msg, body...)

	ids, err := QUICTransportParameterIDs(msg)
	require.NoError(t, err)
	assert.Equal(t, []uint64{0x04, 31*3 + 27, 0x0f}, ids)

	_, err = QUICTransportParameterIDs(msg[:len(msg)-3])
	require.Error(t, err)
}

func FuzzDecryptQUICInitials(f *testing.F) {
	dcid := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	keys := func() []byte {
		t := &testing.T{}
		return sealInitial(t, quicVersion1, dcid, 0, append(cryptoFrame(0, []byte{1, 0, 0, 1, 0}), make([]byte, 30)...))
	}()
	f.Add(keys)
	f.Add([]byte{0xc0, 0, 0, 0, 1, 8, 1, 2, 3, 4, 5, 6, 7, 8, 0, 0, 0x41, 0})
	f.Add([]byte{0x40})

	f.Fuzz(func(t *testing.T, datagram []byte) {
		original := slices.Clone(datagram)
		_, frames, _ := DecryptQUICInitials(datagram)
		assert.Equal(t, original, datagram)

		var a QUICClientHelloAssembler
		for _, fr := range frames {
			if msg, done, err := a.Add(fr); err == nil && done {
				_, _ = ParseClientHelloMessage(msg)
				_, _ = QUICTransportParameterIDs(msg)
			}
		}
	})
}
