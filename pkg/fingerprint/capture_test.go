package fingerprint

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
)

// feedAll feeds the capture in chunks of the given size, a size of 0 meaning in one go.
func feedAll(c *Capture, b []byte, chunk int) {
	if chunk == 0 {
		c.observe(b)
		return
	}
	for len(b) > 0 {
		n := min(chunk, len(b))
		c.observe(b[:n])
		b = b[n:]
	}
}

func pendingJA4H(c *Capture) []string {
	var got []string
	for _, r := range c.pending {
		got = append(got, r.JA4H())
	}
	return got
}

func TestCapture_h1(t *testing.T) {
	vectors := vectorsFor("h1")

	var wire []byte
	var expected []string
	for _, v := range vectors {
		wire = append(wire, v.h1Bytes()...)
		expected = append(expected, v.expected)
	}

	// A chunked body, whose bytes must not be taken for a header block.
	wire = append(wire, "POST /chunked HTTP/1.1\r\nHost: h\r\nTransfer-Encoding: chunked\r\n\r\n4;ext=1\r\nGET \r\n3\r\n/ H\r\n0\r\nTrailer: x\r\n\r\n"...)
	expected = append(expected, "po11nn020000_37ec0c17f899_000000000000_000000000000")

	for _, chunk := range []int{0, 1, 7} {
		c := newCapture(Config{})
		feedAll(c, wire, chunk)

		assert.Equal(t, "h1", c.Proto())
		assert.Equal(t, expected, pendingJA4H(c))
	}
}

func TestCapture_h1Order(t *testing.T) {
	c := newCapture(Config{})
	c.observe([]byte("GET / HTTP/1.1\r\nX-B: 1\r\nhost: h\r\nX-A: 2\r\nx-b: 3\r\n\r\n"))

	require.Len(t, c.pending, 1)
	assert.Equal(t, []string{"X-B", "host", "X-A", "x-b"}, c.pending[0].Names)
}

func TestCapture_h2(t *testing.T) {
	vectors := vectorsFor("h2")
	shapes := []h2Shape{{}, {continuation: true}, {padded: true, priority: true}}

	w := newH2Writer()
	w.buf.Write(h2Preface)
	require.NoError(t, w.fr.WriteSettings(http2.Setting{ID: http2.SettingInitialWindowSize, Val: 6291456}, http2.Setting{ID: http2.SettingHeaderTableSize, Val: 65536}))
	require.NoError(t, w.fr.WriteWindowUpdate(0, 15663105))
	prefixStart := len(h2Preface)

	var expected []string
	var firstBlockEnd int
	streamID := uint32(1)
	// Each vector is sent twice, the second time mostly from the dynamic table.
	for range 2 {
		for i, v := range vectors {
			w.request(t, streamID, v, shapes[i%len(shapes)])
			if streamID == 1 {
				firstBlockEnd = w.buf.Len()
				if v.body != "" {
					t.Fatal("the first vector must not have a body, for the prefix to end at its header block")
				}
			}
			expected = append(expected, v.expected)
			streamID += 2
		}
	}
	// A trailer section, which is not a request, and the reset of an unknown stream.
	w.hb.Reset()
	require.NoError(t, w.enc.WriteField(hpackField("x-trailer", "1")))
	require.NoError(t, w.fr.WriteHeaders(http2.HeadersFrameParam{StreamID: 3, BlockFragment: w.hb.Bytes(), EndStream: true, EndHeaders: true}))
	require.NoError(t, w.fr.WriteRSTStream(streamID+2, http2.ErrCodeCancel))

	wire := w.buf.Bytes()

	for _, chunk := range []int{0, 1, 5} {
		c := newCapture(Config{})
		feedAll(c, wire, chunk)

		assert.Equal(t, "h2", c.Proto())
		assert.Equal(t, expected, pendingJA4H(c))

		prefix, truncated := c.H2Prefix()
		assert.False(t, truncated)
		assert.Equal(t, wire[prefixStart:firstBlockEnd], prefix)

		require.NotEmpty(t, c.pending)
		assert.Equal(t, []string{":method", ":authority", ":scheme", ":path"}, c.pending[0].PseudoNames)
	}
}

func TestCapture_h2ResetStream(t *testing.T) {
	v := vectorsFor("h2")[0]

	w := newH2Writer()
	w.buf.Write(h2Preface)
	require.NoError(t, w.fr.WriteSettings())
	w.request(t, 1, v, h2Shape{})
	w.request(t, 3, v, h2Shape{})
	require.NoError(t, w.fr.WriteRSTStream(1, http2.ErrCodeCancel))

	c := newCapture(Config{})
	c.observe(w.buf.Bytes())

	require.Len(t, c.pending, 1)
	assert.Equal(t, uint32(3), c.pending[0].streamID)
}

func TestCapture_h2PrefixTruncated(t *testing.T) {
	w := newH2Writer()
	w.buf.Write(h2Preface)
	require.NoError(t, w.fr.WriteSettings())
	// PING frames before the first request push its header block past the prefix limit.
	for range maxH2Prefix/17 + 1 {
		require.NoError(t, w.fr.WritePing(false, [8]byte{}))
	}
	w.request(t, 1, vectorsFor("h2")[0], h2Shape{})

	c := newCapture(Config{})
	feedAll(c, w.buf.Bytes(), 1000)

	prefix, truncated := c.H2Prefix()
	assert.True(t, truncated)
	assert.Len(t, prefix, maxH2Prefix)
	assert.Len(t, c.pending, 1)
}

func TestCapture_h2HeaderTableSize(t *testing.T) {
	v := vectorsFor("h2")[0]

	// A client allowed a larger table indexes more fields: the capture must decode with the same table size.
	w := newH2Writer()
	w.enc.SetMaxDynamicTableSizeLimit(65536)
	w.enc.SetMaxDynamicTableSize(65536)
	w.buf.Write(h2Preface)
	require.NoError(t, w.fr.WriteSettings())
	for i := range 20 {
		vv := v
		vv.fields = append(slices.Clone(v.fields), field{"x-filler", strings.Repeat("a", 300+i)})
		w.request(t, uint32(1+2*i), vv, h2Shape{})
	}

	c := newCapture(Config{MaxDecoderHeaderTableSize: 65536})
	c.observe(w.buf.Bytes())
	assert.Len(t, c.pending, 20)
}

func TestCapture_garbageStops(t *testing.T) {
	c := newCapture(Config{})
	c.observe([]byte("\x16\x03\x01garbage without line end"))
	c.observe([]byte("\n"))

	assert.Empty(t, c.pending)
	assert.Equal(t, stateStopped, c.state)
}
