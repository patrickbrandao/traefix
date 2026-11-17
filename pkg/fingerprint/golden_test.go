package fingerprint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGolden_FoxIO replays client byte streams extracted from the FoxIO sample captures
// (github.com/FoxIO-LLC/ja4, pcap/), and compares the fingerprints with the ones of the reference snapshots
// (rust/ja4/src/snapshots/). The streams are not part of the repository: FINGERPRINT_GOLDEN_DIR points at a
// directory holding <name>.client.bin files, the decrypted client side of a TCP stream,
// next to <name>.expected files, one JA4H per line.
func TestGolden_FoxIO(t *testing.T) {
	dir := os.Getenv("FINGERPRINT_GOLDEN_DIR")
	if dir == "" {
		t.Skip("FINGERPRINT_GOLDEN_DIR is not set")
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.client.bin"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".client.bin")
		t.Run(name, func(t *testing.T) {
			expectedRaw, err := os.ReadFile(filepath.Join(dir, name+".expected"))
			if os.IsNotExist(err) {
				t.Skip("no expected fingerprint")
			}
			require.NoError(t, err)

			wire, err := os.ReadFile(file)
			require.NoError(t, err)

			for _, chunk := range []int{0, 1} {
				c := newCapture(Config{})
				feedAll(c, wire, chunk)

				assert.Equal(t, strings.Fields(string(expectedRaw)), pendingJA4H(c))
			}
		})
	}
}
