package fingerprint

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

type field struct {
	name  string
	value string
}

type vector struct {
	name    string
	proto   string
	method  string
	target  string
	version string
	fields  []field
	body    string
	// expected is the JA4H computed by the FoxIO Python reference (python/ja4h.py, to_ja4h) for the same fields.
	expected string
}

// ja4hVectors avoid the inputs on which the FoxIO Rust and Python implementations disagree,
// e.g. an Accept-Language value with a parameter before the first comma, or several cookies in one HTTP/2 field.
var ja4hVectors = []vector{
	{
		name: "h1-curl", proto: "h1", method: "GET", target: "/", version: "HTTP/1.1",
		fields:   []field{{"Host", "example.com"}, {"User-Agent", "curl/8.7.1"}, {"Accept", "*/*"}},
		expected: "ge11nn030000_fe444ad14866_000000000000_000000000000",
	},
	{
		name: "h1-browser", proto: "h1", method: "GET", target: "/index.html?q=1", version: "HTTP/1.1",
		fields: []field{
			{"Host", "example.com"},
			{"Connection", "keep-alive"},
			{"sec-ch-ua", `"Chromium";v="131"`},
			{"User-Agent", "Mozilla/5.0"},
			{"Accept", "text/html"},
			{"Referer", "https://example.com/"},
			{"Accept-Encoding", "gzip, deflate, br"},
			{"Accept-Language", "en-US,en;q=0.9"},
			{"Cookie", "sid=abc; _ga=GA1.2; theme=dark"},
		},
		expected: "ge11cr07enus_3c666a2bb02e_86aa86fc6cd2_760a8ef12d1d",
	},
	{
		name: "h1-post-mixed-case", proto: "h1", method: "POST", target: "/api", version: "HTTP/1.1",
		fields: []field{
			{"host", "example.com"},
			{"content-type", "application/json"},
			{"X-Custom-Header", "1"},
			{"CONTENT-LENGTH", "2"},
			{"accept-language", "pt-BR,pt;q=0.9"},
		},
		body:     "{}",
		expected: "po11nn05ptbr_b98b1bca4055_000000000000_000000000000",
	},
	{
		name: "h1-http10", proto: "h1", method: "HEAD", target: "/", version: "HTTP/1.0",
		fields:   []field{{"User-Agent", "x"}},
		expected: "he10nn010000_b8bcd45ac095_000000000000_000000000000",
	},
	{
		name: "h1-lang-da", proto: "h1", method: "GET", target: "/", version: "HTTP/1.1",
		fields: []field{
			{"Host", "localhost:8000"},
			{"User-Agent", "curl/8.1.2"},
			{"Accept", "*/*"},
			{"Referer", "https://fake.example"},
			{"Cookie", "yummy_cookie=choco; tasty_cookie=strawberry"},
			{"Accept-Language", "da, en-GB;q=0.8, en;q=0.7"},
		},
		expected: "ge11cr04da00_8ddaef5d77af_280f366eaa04_c2fb0fe53442",
	},
	{
		name: "h1-cookie-sort", proto: "h1", method: "PUT", target: "/c", version: "HTTP/1.1",
		fields:   []field{{"Host", "h"}, {"Cookie", "z=1; a=2; m=3; flag"}},
		expected: "pu11cn010000_4a823118b9ba_c999e971302e_005e6e545f35",
	},
	{
		name: "h2-chrome", proto: "h2", method: "GET", target: "/",
		fields: []field{
			{":method", "GET"},
			{":authority", "example.com"},
			{":scheme", "https"},
			{":path", "/"},
			{"sec-ch-ua", `"Chromium";v="131"`},
			{"sec-ch-ua-mobile", "?0"},
			{"user-agent", "Mozilla/5.0"},
			{"accept", "text/html"},
			{"sec-fetch-site", "none"},
			{"referer", "https://example.com/"},
			{"accept-encoding", "gzip, deflate, br, zstd"},
			{"accept-language", "en-US,en;q=0.9"},
			{"cookie", "_ga=GA1.2"},
			{"cookie", "sid=xyz"},
			{"priority", "u=0, i"},
		},
		expected: "ge20cr08enus_308e4456ec31_b50932941748_1a7d4e15bf8f",
	},
	{
		name: "h2-single-cookie", proto: "h2", method: "POST", target: "/api",
		fields: []field{
			{":method", "POST"},
			{":path", "/api"},
			{":scheme", "https"},
			{":authority", "example.com"},
			{"content-type", "application/json"},
			{"accept-language", "pt-BR,pt;q=0.9"},
			{"cookie", "a=1"},
		},
		body:     "{}",
		expected: "po20cn02ptbr_75fe3c69c692_ca978112ca1b_c22fea5d7428",
	},
	{
		name: "h2-no-cookie", proto: "h2", method: "DELETE", target: "/x",
		fields: []field{
			{":method", "DELETE"},
			{":scheme", "http"},
			{":authority", "h"},
			{":path", "/x"},
			{"x-a", "1"},
			{"x-b", "2"},
		},
		expected: "de20nn020000_0ae542561bfc_000000000000_000000000000",
	},
}

func vectorsFor(proto string) []vector {
	var vs []vector
	for _, v := range ja4hVectors {
		if v.proto == proto {
			vs = append(vs, v)
		}
	}
	return vs
}

func (v vector) h1Bytes() []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s %s\r\n", v.method, v.target, v.version)
	for _, f := range v.fields {
		fmt.Fprintf(&b, "%s: %s\r\n", f.name, f.value)
	}
	b.WriteString("\r\n")
	b.WriteString(v.body)
	return []byte(b.String())
}

// h2Writer writes HTTP/2 frames sharing one HPACK encoder, so that the requests after the first one
// reference its dynamic table like a real client does.
type h2Writer struct {
	buf bytes.Buffer
	fr  *http2.Framer
	enc *hpack.Encoder
	hb  bytes.Buffer
}

func newH2Writer() *h2Writer {
	w := &h2Writer{}
	w.fr = http2.NewFramer(&w.buf, nil)
	w.enc = hpack.NewEncoder(&w.hb)
	return w
}

type h2Shape struct {
	padded       bool
	priority     bool
	continuation bool
}

func (w *h2Writer) request(tb testing.TB, streamID uint32, v vector, shape h2Shape) {
	tb.Helper()

	w.hb.Reset()
	for _, f := range v.fields {
		require.NoError(tb, w.enc.WriteField(hpack.HeaderField{Name: f.name, Value: f.value}))
	}
	block := bytes.Clone(w.hb.Bytes())

	first := block
	var rest []byte
	if shape.continuation && len(block) > 2 {
		first, rest = block[:len(block)/2], block[len(block)/2:]
	}

	param := http2.HeadersFrameParam{
		StreamID:      streamID,
		BlockFragment: first,
		EndStream:     v.body == "",
		EndHeaders:    rest == nil,
	}
	if shape.padded {
		param.PadLength = 7
	}
	if shape.priority {
		param.Priority = http2.PriorityParam{StreamDep: 0, Weight: 41, Exclusive: true}
	}
	require.NoError(tb, w.fr.WriteHeaders(param))

	if rest != nil {
		// Split the rest in two CONTINUATION frames to cover a block spanning several of them.
		mid := len(rest) / 2
		require.NoError(tb, w.fr.WriteContinuation(streamID, false, rest[:mid]))
		require.NoError(tb, w.fr.WriteContinuation(streamID, true, rest[mid:]))
	}

	if v.body != "" {
		require.NoError(tb, w.fr.WriteData(streamID, true, []byte(v.body)))
	}
}

func hpackField(name, value string) hpack.HeaderField {
	return hpack.HeaderField{Name: name, Value: value}
}
