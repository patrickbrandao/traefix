package fingerprint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The captures of testdata/golden come from the finger-sniffer reference set (test/golden/tls and test/golden/pcaps
// of github.com/patrickbrandao/finger-sniffer): real curl, Go, Chrome and Firefox clients, and the outputs of
// tshark 4.4, of the FoxIO ja4 tool 0.18.6 and of p0f 3.09b on the same files. testdata/golden/README.md tells
// how they were made. The tests below compare the Traefix fingerprints with these outputs, so that a divergence
// between Traefix and the services reading its records shows here.

type goldenTLSRef struct {
	Case         string `json:"case"`
	ClientHellos []struct {
		TCPStream int      `json:"tcp_stream"`
		JA3       string   `json:"ja3"`
		JA3Full   string   `json:"ja3_full"`
		JA4       string   `json:"ja4_tool"`
		JA4R      string   `json:"ja4_r_tool"`
		SNI       string   `json:"sni"`
		ALPN      []string `json:"alpn"`
	} `json:"clientHellos"`
	Akamai []struct {
		TCPStream       int    `json:"tcp_stream"`
		Akamai          string `json:"akamai"`
		HeadersPriority *struct {
			Stream    int `json:"stream"`
			Exclusive int `json:"exclusive"`
			DependsOn int `json:"depends_on"`
			Weight    int `json:"weight"`
		} `json:"headersPriority"`
	} `json:"akamai"`
	JA4H []struct {
		TCPStream int    `json:"tcp_stream"`
		JA4H      string `json:"ja4h"`
	} `json:"ja4h"`
	ClientH2TCPStream *int `json:"clientH2TcpStream"`
}

func readGoldenTLSRef(t *testing.T, file string) goldenTLSRef {
	t.Helper()

	raw, err := os.ReadFile(file)
	require.NoError(t, err)

	var ref goldenTLSRef
	require.NoError(t, json.Unmarshal(raw, &ref))

	return ref
}

// TestGolden_clientHello parses the ClientHello of each TLS connection of the captures, and compares JA3 with the
// output of tshark, and JA4 and JA4_r with the output of the FoxIO tool.
func TestGolden_clientHello(t *testing.T) {
	files, err := filepath.Glob("testdata/golden/tls/*.ref.json")
	require.NoError(t, err)
	require.Len(t, files, 10)

	for _, file := range files {
		ref := readGoldenTLSRef(t, file)
		t.Run(ref.Case, func(t *testing.T) {
			streams := pcapStreams(readPcap(t, strings.TrimSuffix(file, ".ref.json")+".pcap"))
			require.NotEmpty(t, ref.ClientHellos)

			for _, expected := range ref.ClientHellos {
				require.Less(t, expected.TCPStream, len(streams))
				stream := streams[expected.TCPStream]

				wire := stream.clientBytes()
				n, ok := ClientHelloLen(wire)
				require.True(t, ok, stream.String())
				hello, err := ParseClientHello(wire[:n])
				require.NoError(t, err, stream.String())

				assert.Equal(t, expected.JA3Full, hello.JA3(), stream.String())
				assert.Equal(t, expected.JA3, hello.JA3Hash(), stream.String())
				v := hello.JA4Variants(false)
				assert.Equal(t, expected.JA4, v.JA4, stream.String())
				assert.Equal(t, expected.JA4R, v.R, stream.String())
				assert.Equal(t, expected.SNI, hello.ServerName, stream.String())
				if len(expected.ALPN) == 0 {
					assert.Empty(t, hello.ALPN, stream.String())
				} else {
					assert.Equal(t, expected.ALPN, hello.ALPN, stream.String())
				}
			}
		})
	}
}

// TestGolden_http2 replays the decrypted client side of the main HTTP/2 connection of each capture, and compares
// the Akamai fingerprint and the priority of the first HEADERS frame with the frames that tshark decoded,
// and the JA4H of each request with the output of the FoxIO tool.
func TestGolden_http2(t *testing.T) {
	files, err := filepath.Glob("testdata/golden/tls/*.client.h2")
	require.NoError(t, err)
	require.Len(t, files, 5)

	for _, file := range files {
		ref := readGoldenTLSRef(t, strings.TrimSuffix(file, ".client.h2")+".ref.json")
		t.Run(ref.Case, func(t *testing.T) {
			require.NotNil(t, ref.ClientH2TCPStream)
			stream := *ref.ClientH2TCPStream

			wire, err := os.ReadFile(file)
			require.NoError(t, err)

			for _, chunk := range []int{0, 1, 13} {
				c := newCapture(Config{})
				feedAll(c, wire, chunk)
				require.Equal(t, "h2", c.Proto())

				prefix, truncated := c.H2Prefix()
				require.NotNil(t, prefix)
				require.False(t, truncated)

				var akamai, priority string
				for _, a := range ref.Akamai {
					if a.TCPStream != stream {
						continue
					}
					akamai = a.Akamai
					if p := a.HeadersPriority; p != nil {
						// tshark shows the effective weight, from 1 to 256.
						priority = strconv.Itoa(p.Stream) + ":" + strconv.Itoa(p.Exclusive) + ":" + strconv.Itoa(p.DependsOn) + ":" + strconv.Itoa(p.Weight)
					}
				}
				require.NotEmpty(t, akamai)
				assert.Equal(t, akamai, AkamaiH2(prefix, c.FirstPseudoNames()))
				assert.Equal(t, priority, H2HeadersPriority(prefix))

				var expected []string
				for _, r := range ref.JA4H {
					if r.TCPStream == stream {
						expected = append(expected, r.JA4H)
					}
				}
				// The FoxIO tool prints no JA4H for some requests, e.g. the ones of curl with OpenSSL, and the
				// decrypted bytes may stop before the last requests of the connection.
				got := pendingJA4H(c)
				if len(expected) > 0 {
					require.NotEmpty(t, got)
					assert.Equal(t, expected[:min(len(got), len(expected))], got)
				}
			}
		})
	}
}
