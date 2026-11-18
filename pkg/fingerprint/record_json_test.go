package fingerprint

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendJSON_matchesEncodingJSON(t *testing.T) {
	hostile := "a\"b\\c\n\r\t\x00\x1f<>&  é\xff\xfe日本"
	now := time.Date(2026, 11, 17, 21, 30, 0, 123456789, time.UTC)

	records := []jsonAppender{
		&requestRecord{Type: recordRequest, Start: now, End: now, Proto: "h3", Method: "GET", Path: "/", HTTPVersion: "HTTP/3.0"},
		benchRequestRecord(),
		&requestRecord{
			Type: recordRequest, ConnID: "id", Start: now, End: now.Add(time.Second), EntryPoint: hostile, Proto: "h1",
			Method: hostile, Authority: hostile, Path: hostile, HTTPVersion: "HTTP/1.1", FieldNames: []string{hostile, ""},
			Fields: []Field{{Name: hostile, Value: hostile}}, CookieNames: []string{hostile}, JA4H: hostile,
			HeadersTruncated: true, Status: 599, Router: hostile, Service: hostile,
		},
		&connectionRecord{Type: recordConnection, ConnID: "id", Time: now, EntryPoint: "ep", Src: addrRecord{IP: "::1", Port: 1}, Dst: addrRecord{IP: "10.0.0.1", Port: 65535}},
		&connectionRecord{
			Type: recordConnection, ConnID: "id", Time: now, EntryPoint: "ep", Proto: "h2",
			TLS:   &tlsRecord{ClientHelloTruncated: true},
			HTTP2: &http2Record{},
		},
		&connectionRecord{
			Type: recordConnection, ConnID: "id", Time: now, EntryPoint: "ep", Proto: "h2",
			Src: addrRecord{IP: "1.2.3.4", Port: 5555}, Dst: addrRecord{IP: "5.6.7.8", Port: 443},
			TLS: &tlsRecord{
				SNI: hostile, ALPNOffered: []string{"h2", "http/1.1"}, ALPN: "h2", Version: "TLS 1.3", Cipher: "TLS_AES_128_GCM_SHA256",
				JA3: "abc", JA3String: "771,4865,0-23,29,0", JA3N: "def", JA3NString: "771,4865,0-23,29,0",
				JA4: "t13d", JA4R: "t13d_1301_000a", JA4O: "t13d_o", ClientHello: []byte{0x16, 3, 1, 0, 0xff},
			},
			HTTP2: &http2Record{Akamai: "1:65536|0|0|m,a,s,p", HeadersPriority: "1:1:0:256", Prefix: []byte("prefix"), PrefixTruncated: true},
		},
		&connectionRecord{
			Type: recordConnection, ConnID: "id", Time: now, EntryPoint: "ep", Proto: "h3",
			TLS:   &tlsRecord{JA4: "q13d", QUICTransportParameters: []uint64{4, 5, 0x0f, 0x1a2a3a4a}},
			HTTP3: &http3Record{Settings: "1:0;6:16384"},
		},
	}

	for _, r := range records {
		expected, err := json.Marshal(r)
		require.NoError(t, err)

		got := r.appendJSON(nil)
		assert.JSONEq(t, string(expected), string(got))
		assert.True(t, json.Valid(got))
	}
}

func BenchmarkRequestRecordAppendJSON(b *testing.B) {
	rec := benchRequestRecord()
	b.ReportAllocs()
	var buf []byte
	for b.Loop() {
		buf = rec.appendJSON(buf[:0])
	}
}
