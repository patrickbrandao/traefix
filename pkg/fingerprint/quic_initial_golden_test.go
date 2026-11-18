package fingerprint

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoldenQUIC_FoxIO decrypts the client datagrams extracted from the FoxIO QUIC sample captures
// (github.com/FoxIO-LLC/ja4, pcap/), and compares the JA4 of their ClientHello with the reference snapshots
// (rust/ja4/src/snapshots/). The datagrams are not part of the repository: FINGERPRINT_GOLDEN_QUIC_DIR points at a
// directory holding <name>.datagrams files, one hex datagram per line in capture order, next to <name>.ja4 files.
func TestGoldenQUIC_FoxIO(t *testing.T) {
	dir := os.Getenv("FINGERPRINT_GOLDEN_QUIC_DIR")
	if dir == "" {
		t.Skip("FINGERPRINT_GOLDEN_QUIC_DIR is not set")
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.datagrams"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".datagrams")
		t.Run(name, func(t *testing.T) {
			expected, err := os.ReadFile(filepath.Join(dir, name+".ja4"))
			require.NoError(t, err)
			content, err := os.ReadFile(file)
			require.NoError(t, err)

			var a QUICClientHelloAssembler
			var msg []byte
			for line := range strings.Lines(string(content)) {
				datagram, err := hex.DecodeString(strings.TrimSpace(line))
				require.NoError(t, err)

				_, frames, _ := DecryptQUICInitials(datagram)
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
			assert.Equal(t, strings.TrimSpace(string(expected)), hello.JA4(true))

			ids, err := QUICTransportParameterIDs(msg)
			require.NoError(t, err)
			assert.NotEmpty(t, ids)
			t.Logf("SNI %s, transport parameters %v", hello.ServerName, ids)
		})
	}
}
