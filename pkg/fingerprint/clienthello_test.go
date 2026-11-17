package fingerprint

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goClientHellos returns the ClientHellos written by crypto/tls clients (go1.27.4), from testdata/clienthello-go.txt.
func goClientHellos(t *testing.T) map[string][]byte {
	t.Helper()

	f, err := os.Open("testdata/clienthello-go.txt")
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	hellos := map[string][]byte{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(nil, 1<<20)
	for scanner.Scan() {
		name, hexBytes, ok := strings.Cut(scanner.Text(), " ")
		require.True(t, ok)
		raw, err := hex.DecodeString(hexBytes)
		require.NoError(t, err)
		hellos[name] = raw
	}
	require.NoError(t, scanner.Err())

	return hellos
}

// The expected JA3 come from an independent Python implementation of the Salesforce specification run on the same bytes.
func TestClientHello_JA3(t *testing.T) {
	testCases := []struct {
		name     string
		ja3      string
		ja3Hash  string
		sni      string
		alpn     []string
		versions []uint16
	}{
		{
			name:     "go-tls13-sni-alpn",
			ja3:      "771,49195-49199-49196-49200-52393-52392-49161-49171-49162-49172-4865-4866-4867,0-11-65281-23-18-5-10-13-50-16-43-51,4588-4587-4589-29-23-24-25,0",
			ja3Hash:  "03117a8ed39ef02427ebbc39f121275c",
			sni:      "example.com",
			alpn:     []string{"h2", "http/1.1"},
			versions: []uint16{0x0304, 0x0303},
		},
		{
			name:     "go-tls12-no-sni",
			ja3:      "771,49199-49196,11-65281-23-18-5-10-13-50-43,29-23,0",
			ja3Hash:  "b94e92bcb378d92d4b5df78812e8c285",
			versions: []uint16{0x0303},
		},
	}

	hellos := goClientHellos(t)
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			hello, err := ParseClientHello(hellos[test.name])
			require.NoError(t, err)

			assert.Equal(t, test.ja3, hello.JA3())
			assert.Equal(t, test.ja3Hash, hello.JA3Hash())
			assert.Equal(t, test.sni, hello.ServerName)
			assert.Equal(t, test.alpn, hello.ALPN)
			assert.Equal(t, test.versions, hello.SupportedVersions)
		})
	}
}

// refragment splits the handshake message of a single-record ClientHello into records of at most size bytes.
func refragment(t *testing.T, raw []byte, size int) []byte {
	t.Helper()

	msg, n, err := clientHelloMessage(raw)
	require.NoError(t, err)
	require.Len(t, raw, n)

	return refragmentMsg(msg, size)
}

func TestParseClientHello_fragmented(t *testing.T) {
	raw := goClientHellos(t)["go-tls13-sni-alpn"]
	whole, err := ParseClientHello(raw)
	require.NoError(t, err)

	for _, size := range []int{1, 3, 50, 1000} {
		fragmented := refragment(t, raw, size)
		// Bytes past the ClientHello, e.g. a ChangeCipherSpec record, are not part of it.
		wire := append(append([]byte{}, fragmented...), 0x14, 0x03, 0x03, 0x00, 0x01, 0x01)

		// Read one byte at a time, as the capture does with short reads.
		var got int
		for i := 1; i <= len(wire); i++ {
			n, ok := ClientHelloLen(wire[:i])
			if ok {
				got = n
				break
			}
		}
		assert.Equal(t, len(fragmented), got)

		hello, err := ParseClientHello(wire)
		require.NoError(t, err)
		assert.Equal(t, whole, hello)
	}
}

// bigClientHello returns a ClientHello carrying a padding extension that makes it longer than size bytes,
// split over records of the maximum plaintext length.
func bigClientHello(t *testing.T, size int) []byte {
	t.Helper()

	raw := goClientHellos(t)["go-tls12-no-sni"]
	msg, _, err := clientHelloMessage(raw)
	require.NoError(t, err)

	body := msg[tlsHandshakeHeaderLen:]
	// Locate the extensions block: version, random, session ID, cipher suites, compression methods.
	p := 2 + 32
	p += 1 + int(body[p])
	p += 2 + int(binary.BigEndian.Uint16(body[p:]))
	p += 1 + int(body[p])
	extLen := int(binary.BigEndian.Uint16(body[p:]))

	padding := make([]byte, 4+size)
	binary.BigEndian.PutUint16(padding, 21)
	binary.BigEndian.PutUint16(padding[2:], uint16(size))

	newBody := append([]byte{}, body[:p]...)
	newBody = binary.BigEndian.AppendUint16(newBody, uint16(extLen+len(padding)))
	newBody = append(newBody, body[p+2:]...)
	newBody = append(newBody, padding...)

	newMsg := []byte{tlsHandshakeClientHello, byte(len(newBody) >> 16), byte(len(newBody) >> 8), byte(len(newBody))}
	newMsg = append(newMsg, newBody...)

	return refragmentMsg(newMsg, 16384)
}

// refragmentMsg puts a handshake message in records of at most size bytes.
func refragmentMsg(msg []byte, size int) []byte {
	var out []byte
	for len(msg) > 0 {
		chunk := msg[:min(size, len(msg))]
		msg = msg[len(chunk):]
		out = append(out, tlsRecordTypeHandshake, 0x03, 0x01, byte(len(chunk)>>8), byte(len(chunk)))
		out = append(out, chunk...)
	}
	return out
}

func TestParseClientHello_longerThanCaptureLimit(t *testing.T) {
	const captureLimit = 16 << 10

	raw := bigClientHello(t, 20000)
	require.Greater(t, len(raw), captureLimit)

	_, ok := ClientHelloLen(raw[:captureLimit])
	assert.False(t, ok)

	_, err := ParseClientHello(raw[:captureLimit])
	require.Error(t, err)

	n, ok := ClientHelloLen(raw)
	assert.True(t, ok)
	assert.Equal(t, len(raw), n)

	hello, err := ParseClientHello(raw)
	require.NoError(t, err)
	assert.Equal(t, uint16(21), hello.Extensions[len(hello.Extensions)-1])
}

func TestParseClientHello_malformed(t *testing.T) {
	raw := goClientHellos(t)["go-tls12-no-sni"]

	testCases := []struct {
		desc string
		raw  []byte
	}{
		{desc: "empty"},
		{desc: "not handshake", raw: []byte{0x17, 0x03, 0x03, 0x00, 0x01, 0x00}},
		{desc: "empty record", raw: []byte{0x16, 0x03, 0x01, 0x00, 0x00}},
		{desc: "not a ClientHello", raw: []byte{0x16, 0x03, 0x01, 0x00, 0x04, 0x02, 0x00, 0x00, 0x00}},
		{desc: "truncated record", raw: raw[:len(raw)-1]},
		{desc: "SSLv2", raw: []byte{0x80, 0x2e, 0x01, 0x00, 0x02}},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			_, err := ParseClientHello(test.raw)
			assert.Error(t, err)
		})
	}

	// A message whose inner lengths overflow the message.
	msg := []byte{0x01, 0x00, 0x00, 0x26, 0x03, 0x03}
	msg = append(msg, make([]byte, 32)...)
	msg = append(msg, 0xff, 0x00)
	_, err := ParseClientHelloMessage(msg)
	assert.Error(t, err)
}

func TestClientHello_GREASE(t *testing.T) {
	hello := &ClientHello{
		LegacyVersion:       0x0303,
		CipherSuites:        []uint16{0x2a2a, 0x1301, 0xc02b},
		Extensions:          []uint16{0x0a0a, 0, 16, 43, 13, 10, 11, 0xfafa},
		ALPN:                []string{"h2"},
		SupportedGroups:     []uint16{0x4a4a, 29, 23},
		PointFormats:        []uint8{0},
		SignatureAlgorithms: []uint16{0x0403, 0x0804},
		SupportedVersions:   []uint16{0x3a3a, 0x0304, 0x0303},
	}

	assert.Equal(t, "771,4865-49195,0-16-43-13-10-11,29-23,0", hello.JA3())
	// 2 ciphers and 6 extensions once the GREASE values are removed, TLS 1.3 from the supported versions.
	assert.True(t, strings.HasPrefix(hello.JA4(false), "t13d0206h2_"), hello.JA4(false))
	assert.True(t, strings.HasPrefix(hello.JA4(true), "q13d0206h2_"))

	for _, v := range []uint16{0x0a0a, 0x1a1a, 0xfafa} {
		assert.True(t, isGREASE(v))
	}
	for _, v := range []uint16{0x0a1a, 0x0303, 0x1301} {
		assert.False(t, isGREASE(v))
	}
}

func TestClientHello_JA4ALPN(t *testing.T) {
	testCases := []struct {
		alpn     []string
		expected string
	}{
		{expected: "00"},
		{alpn: []string{"h2", "http/1.1"}, expected: "h2"},
		{alpn: []string{"http/1.1"}, expected: "h1"},
		{alpn: []string{"x"}, expected: "xx"},
		{alpn: []string{"\xe2\x82\xacb"}, expected: "9b"},
	}

	for _, test := range testCases {
		hello := &ClientHello{LegacyVersion: 0x0303, ALPN: test.alpn}
		assert.Equal(t, test.expected, hello.JA4(false)[8:10])
	}
}

func FuzzParseClientHello(f *testing.F) {
	data, err := os.ReadFile("testdata/clienthello-go.txt")
	require.NoError(f, err)
	for line := range strings.Lines(string(data)) {
		_, hexBytes, _ := strings.Cut(strings.TrimSpace(line), " ")
		raw, err := hex.DecodeString(hexBytes)
		require.NoError(f, err)
		f.Add(raw)
	}
	f.Add([]byte{0x16, 0x03, 0x01, 0x00, 0x05, 0x01, 0x00, 0x00, 0x01, 0x03})

	f.Fuzz(func(t *testing.T, raw []byte) {
		hello, err := ParseClientHello(raw)
		if err != nil {
			return
		}
		_ = hello.JA3Hash()
		_ = hello.JA4(false)
		_, _ = ClientHelloLen(raw)
	})
}
