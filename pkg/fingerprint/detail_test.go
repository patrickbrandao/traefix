package fingerprint

import (
	"bytes"
	"crypto/tls"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
)

func TestEscapeValue(t *testing.T) {
	assert.Equal(t, "example.com", escapeValue("example.com", ",="))
	assert.Equal(t, "a%2Cb%3Dc%25%20%FF", escapeValue("a,b=c% \xff", ",="))
}

func TestBoundedList(t *testing.T) {
	assert.Equal(t, "a,b,c", boundedList([]string{"a", "b", "c"}, ",", 100))
	assert.Empty(t, boundedList(nil, ",", 100))

	items := make([]string, 300)
	for i := range items {
		items[i] = "header-name"
	}
	got := boundedList(items, ",", 1024)
	assert.LessOrEqual(t, len(got), 1024)
	assert.True(t, strings.HasSuffix(got, ","+truncatedItem), got)
}

func TestBoundedPairs(t *testing.T) {
	assert.Equal(t, "a=1,b=2", boundedPairs("a=1,b=2"))

	long := strings.Repeat("key=value,", 400)
	got := boundedPairs(long)
	assert.LessOrEqual(t, len(got), maxDetailHeader)
	assert.True(t, strings.HasSuffix(got, ","+truncatedMarker), got)
}

func TestTLSHelloDetail(t *testing.T) {
	hello := &ClientHello{
		LegacyVersion:       0x0303,
		CipherSuites:        []uint16{0x2a2a, 0x1301, 0xc02b},
		Extensions:          []uint16{0x0a0a, 0, 16, 51, 0xfe0d},
		ALPN:                []string{"h2", "http/1.1"},
		SupportedGroups:     []uint16{0x4a4a, 0x11ec, 29},
		PointFormats:        []uint8{0},
		SignatureAlgorithms: []uint16{0x0403},
		SupportedVersions:   []uint16{0x3a3a, 0x0304},
		KeyShareGroups:      []uint16{0x4a4a, 0x11ec, 29},
		PSKModes:            []uint8{1},
		CertCompression:     []uint16{2},
		RecordSizeLimit:     16385,
		ALPS:                []string{"h2"},
		ECH:                 true,
	}

	assert.Equal(t, "legacy_ver=0303,ciphers=g-1301-c02b,exts=g-0000-0010-0033-fe0d,groups=g-11ec-001d,"+
		"key_shares=g-11ec-001d,sigalgs=0403,versions=g-0304,point_formats=0,psk_modes=1,cert_compress=0002,"+
		"record_limit=16385,alpn=h2-http/1.1,alps=h2,ech_ext=1", tlsHelloDetail(hello))
}

func TestTLSDetail(t *testing.T) {
	hello := &ClientHello{ServerName: "example.com", KeyShareGroups: []uint16{uint16(tls.X25519)}}
	state := tls.ConnectionState{
		HandshakeComplete: true, Version: tls.VersionTLS13, CipherSuite: tls.TLS_AES_128_GCM_SHA256,
		CurveID: tls.X25519MLKEM768, NegotiatedProtocol: "h2",
	}

	// The negotiated group is not a key share of the first ClientHello: the server asked for another one.
	assert.Equal(t, "ver=1.3,cipher=TLS_AES_128_GCM_SHA256,group=X25519MLKEM768,alpn=h2,resumed=0,ech=0,"+
		"client_cert=0,sni=example.com,hrr=1,ch_len=517", tlsDetail(state, hello, 517, false))

	state.CurveID = tls.X25519
	assert.Contains(t, tlsDetail(state, hello, 517, false), ",hrr=0,")
}

func TestH2Detail(t *testing.T) {
	var buf bytes.Buffer
	fr := http2.NewFramer(&buf, nil)
	require.NoError(t, fr.WriteSettings(http2.Setting{ID: http2.SettingHeaderTableSize, Val: 65536}))
	require.NoError(t, fr.WriteWindowUpdate(0, 15663105))
	require.NoError(t, fr.WriteHeaders(http2.HeadersFrameParam{
		StreamID: 1, BlockFragment: []byte{0x82}, EndHeaders: true, EndStream: true,
		Priority: http2.PriorityParam{Exclusive: true, Weight: 255},
	}))

	akamai := AkamaiH2(buf.Bytes(), []string{":method", ":authority", ":scheme", ":path"})
	assert.Equal(t, "settings=1:65536,window_update=15663105,priority=0,headers_priority=1:1:0:256,"+
		"first_headers_flags=0x25,pseudo=m-a-s-p", h2Detail(akamai, buf.Bytes()))
}

func TestQUICTransportParametersDetail(t *testing.T) {
	// A ClientHello message with only the QUIC transport parameters extension.
	params := []byte{
		0x01, 0x02, 0x67, 0x10, // max_idle_timeout: 10000
		0x0c, 0x00, // disable_active_migration
		0x0f, 0x04, 1, 2, 3, 4, // initial_scid: 4 bytes
		0x1b, 0x01, 0x00, // GREASE (31*0+27)
		0x40, 0x99, 0x01, 0x05, // 0x99: 5
	}
	ext := append([]byte{0x00, 0x39, 0x00, byte(len(params))}, params...)
	body := []byte{0x03, 0x03}
	body = append(body, make([]byte, 32)...)
	body = append(body, 0x00)                   // session id
	body = append(body, 0x00, 0x02, 0x13, 0x01) // ciphers
	body = append(body, 0x01, 0x00)             // compression
	body = append(body, 0x00, byte(len(ext)))
	body = append(body, ext...)
	msg := append([]byte{0x01, 0x00, 0x00, byte(len(body))}, body...)

	assert.Equal(t, "max_idle_timeout:10000-disable_active_migration-initial_scid:4b-grease-0x99:1b",
		quicTransportParametersDetail(msg))

	ids, err := QUICTransportParameterIDs(msg)
	require.NoError(t, err)
	assert.Equal(t, []uint64{0x01, 0x0c, 0x0f, 0x1b, 0x99}, ids)
}

func TestRequestDetail(t *testing.T) {
	order, cookies := requestDetail(&Request{
		ProtoMajor: 1,
		Names:      []string{"Host", "User-Agent", "Cookie", "X,Odd"},
		Cookies:    []string{"sid=1; theme=dark"},
	})
	assert.Equal(t, "Host,User-Agent,Cookie,X%2COdd", order)
	assert.Equal(t, "sid,theme", cookies)
}
