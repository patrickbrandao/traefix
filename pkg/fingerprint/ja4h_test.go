package fingerprint

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkJA4H(b *testing.B) {
	r := &Request{Method: "GET", ProtoMajor: 2}
	for _, f := range ja4hVectors[6].fields {
		if f.name[0] == ':' {
			continue
		}
		r.Names = append(r.Names, f.name)
		switch f.name {
		case "cookie":
			r.Cookies = append(r.Cookies, f.value)
		case "accept-language":
			r.AcceptLanguage = f.value
		}
	}

	b.ReportAllocs()
	for b.Loop() {
		r.ja4h = ""
		_ = r.JA4H()
	}
}

// BenchmarkCapture measures the work added to each request: parsing its header block from the wire,
// handing it to the handler, and computing its JA4H.
func BenchmarkCapture(b *testing.B) {
	b.Run("h1", func(b *testing.B) {
		v := ja4hVectors[1]
		wire := v.h1Bytes()
		req := httptest.NewRequest(v.method, v.target, nil)
		req.RequestURI = v.target

		c := newCapture(Config{})
		b.ReportAllocs()
		b.SetBytes(int64(len(wire)))
		for b.Loop() {
			c.observe(wire)
			r := c.take(req)
			if r == nil {
				b.Fatal("request not captured")
			}
			_ = r.JA4H()
		}
	})

	b.Run("h2", func(b *testing.B) {
		v := ja4hVectors[6]
		req := httptest.NewRequest(v.method, v.target, nil)
		req.RequestURI = v.target
		req.Host = "example.com"

		// The requests after the first one come mostly from the HPACK dynamic table, as with a browser.
		w := newH2Writer()
		w.buf.Write(h2Preface)
		require.NoError(b, w.fr.WriteSettings())
		c := newCapture(Config{})
		c.observe(w.buf.Bytes())

		streamID := uint32(1)
		frames := make([][]byte, 0, 1024)
		for range cap(frames) {
			w.buf.Reset()
			w.request(b, streamID, v, h2Shape{})
			frames = append(frames, bytes.Clone(w.buf.Bytes()))
			streamID += 2
		}

		b.ReportAllocs()
		i := 0
		for b.Loop() {
			c.observe(frames[i%len(frames)])
			r := c.take(req)
			if r == nil {
				b.Fatal("request not captured")
			}
			_ = r.JA4H()
			i++
		}
	})
}
