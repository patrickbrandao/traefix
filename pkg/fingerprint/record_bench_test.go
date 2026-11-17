package fingerprint

import (
	"encoding/json"
	"testing"
	"time"
)

func benchRequestRecord() *requestRecord {
	return &requestRecord{
		Type: recordRequest, ConnID: newULID(time.Now()), Start: time.Now(), End: time.Now(), EntryPoint: "websecure",
		Proto: "h1", Method: "GET", Authority: "example.com", Path: "/bench", HTTPVersion: "HTTP/1.1",
		FieldNames: []string{"Host", "User-Agent", "Sec-Ch-Ua", "Sec-Ch-Ua-Mobile", "Accept", "Sec-Fetch-Site", "Referer", "Accept-Encoding", "Accept-Language", "Cookie", "Priority"},
		Fields: []Field{
			{"User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"},
			{"Sec-Ch-Ua", `"Chromium";v="131", "Not_A Brand";v="24"`},
			{"Sec-Ch-Ua-Mobile", "?0"},
			{"Accept", "text/html"},
			{"Sec-Fetch-Site", "none"},
			{"Accept-Encoding", "gzip, deflate, br, zstd"},
			{"Accept-Language", "en-US,en;q=0.9"},
		},
		CookieNames: []string{"_ga", "sid"}, JA4H: "ge11cr08enus_308e4456ec31_b50932941748_1a7d4e15bf8f", Status: 200,
		Router: "app@file", Service: "app@file",
	}
}

func BenchmarkRequestRecordJSON(b *testing.B) {
	rec := benchRequestRecord()
	b.ReportAllocs()
	var n int
	for b.Loop() {
		out, err := json.Marshal(rec)
		if err != nil {
			b.Fatal(err)
		}
		n = len(out)
	}
	b.ReportMetric(float64(n), "bytes")
}
