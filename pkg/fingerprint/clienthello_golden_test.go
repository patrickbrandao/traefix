package fingerprint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoldenTLS_FoxIO parses the ClientHello of client byte streams extracted from the FoxIO sample captures
// (github.com/FoxIO-LLC/ja4, pcap/), and compares its JA4 with the one of the reference snapshots
// (rust/ja4/src/snapshots/). The streams are not part of the repository: FINGERPRINT_GOLDEN_TLS_DIR points at a
// directory holding <name>.client.bin files, the raw client side of a TCP stream, next to <name>.ja4 files.
func TestGoldenTLS_FoxIO(t *testing.T) {
	dir := os.Getenv("FINGERPRINT_GOLDEN_TLS_DIR")
	if dir == "" {
		t.Skip("FINGERPRINT_GOLDEN_TLS_DIR is not set")
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.client.bin"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".client.bin")
		t.Run(name, func(t *testing.T) {
			expected, err := os.ReadFile(filepath.Join(dir, name+".ja4"))
			require.NoError(t, err)

			raw, err := os.ReadFile(file)
			require.NoError(t, err)

			n, ok := ClientHelloLen(raw)
			require.True(t, ok)

			hello, err := ParseClientHello(raw[:n])
			require.NoError(t, err)

			assert.Equal(t, strings.TrimSpace(string(expected)), hello.JA4(false))
		})
	}
}
