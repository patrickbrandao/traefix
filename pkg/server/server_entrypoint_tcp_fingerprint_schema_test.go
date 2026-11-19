//go:build !windows

package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kaptinlin/jsonschema"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v3/pkg/config/static"
	tcprouter "github.com/traefik/traefik/v3/pkg/server/router/tcp"
	traefiktls "github.com/traefik/traefik/v3/pkg/tls"
)

const fingerprintSchemaFile = "../../docs/content/reference/install-configuration/traefix-fingerprint-v1.schema.json"

// TestFingerprintExport_schema checks the batches an entry point exports, over HTTP/1, HTTP/2 and HTTP/3, with and
// without TLS, against the published JSON Schema, which the receiving services validate against. It also checks
// that the schema declares every field the batches carry, since the schema accepts unknown fields for the services
// running an older version.
func TestFingerprintExport_schema(t *testing.T) {
	schemaBytes, err := os.ReadFile(fingerprintSchemaFile)
	require.NoError(t, err)
	schema, err := jsonschema.NewCompiler().Compile(schemaBytes)
	require.NoError(t, err)
	require.False(t, schema.ValidateJSON([]byte(`{"schema":"traefix.fingerprint/v1","sender":"traefix/x","entryPoint":"ep","sentAt":"2026-11-18T00:00:00Z","records":[{"type":"connection","connId":"bad"}]}`)).IsValid())

	var mu sync.Mutex
	var batches [][]byte
	dest := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err == nil {
			mu.Lock()
			batches = append(batches, body)
			mu.Unlock()
		}
	}))
	t.Cleanup(dest.Close)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	// The tcp option only gives records on Linux, elsewhere the entry point logs a warning.
	fp := &static.Fingerprint{HeaderPrefix: "X-Edge-", HTTP3HeaderOrder: true, TCP: true, Export: &static.FingerprintExport{URL: dest.URL, RawBytes: true}}
	fp.Export.SetDefaults()
	fp.Export.FlushInterval = ptypes.Duration(20 * time.Millisecond)

	epConfig := &static.EntryPointsTransport{}
	epConfig.SetDefaults()
	entryPoint, err := NewTCPEntryPoint(t.Context(), "websecure", &static.EntryPoint{
		Address:          addr,
		Transport:        epConfig,
		ForwardedHeaders: &static.ForwardedHeaders{},
		HTTP2:            &static.HTTP2Config{},
		HTTP3:            &static.HTTP3Config{},
		Fingerprint:      fp,
	}, nil, nil)
	require.NoError(t, err)

	certContent, err := localhostCert.Read()
	require.NoError(t, err)
	keyContent, err := localhostKey.Read()
	require.NoError(t, err)
	tlsCert, err := tls.X509KeyPair(certContent, keyContent)
	require.NoError(t, err)

	router, err := tcprouter.NewRouter(nil)
	require.NoError(t, err)
	router.AddHTTPTLSConfig("example.com", &tls.Config{Certificates: []tls.Certificate{tlsCert}, NextProtos: []string{"h2", "http/1.1"}}, traefiktls.DefaultTLSConfigName)
	router.SetHTTPSHandler(http.HandlerFunc(echoFingerprintHeaders), nil)
	router.SetHTTPHandler(http.HandlerFunc(echoFingerprintHeaders))

	go entryPoint.Start(t.Context())
	entryPoint.SwitchRouter(router)
	t.Cleanup(func() { entryPoint.Shutdown(context.Background()) })

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(certContent)
	tlsConf := &tls.Config{RootCAs: certPool, ServerName: "example.com"}

	roundtrip := func(t *testing.T, rt http.RoundTripper, url string) {
		t.Helper()

		req, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err)
		req.Host = "example.com"
		req.Header.Set("User-Agent", "schema-test")
		req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9")
		req.Header.Set("Cookie", "a=1; b=2")
		resp, err := rt.RoundTrip(req)
		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	for _, proto := range []string{"h1", "h2"} {
		var protocols http.Protocols
		if proto == "h2" {
			protocols.SetHTTP2(true)
		} else {
			protocols.SetHTTP1(true)
		}
		transport := &http.Transport{TLSClientConfig: tlsConf.Clone(), Protocols: &protocols}
		roundtrip(t, transport, "https://"+addr+"/"+proto)
		transport.CloseIdleConnections()
	}

	plain := &http.Transport{}
	roundtrip(t, plain, "http://"+addr+"/plain")
	plain.CloseIdleConnections()

	h3 := &http3.Transport{TLSClientConfig: &tls.Config{RootCAs: certPool, ServerName: "example.com", NextProtos: []string{http3.NextProtoH3}}, QUICConfig: &quic.Config{}}
	t.Cleanup(func() { _ = h3.Close() })
	require.Eventually(t, func() bool {
		req, err := http.NewRequest(http.MethodGet, "https://"+addr+"/h3", nil)
		require.NoError(t, err)
		resp, err := h3.RoundTrip(req)
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return true
	}, 5*time.Second, 100*time.Millisecond)
	_ = h3.Close()

	// 4 connections and 4 requests.
	var all [][]byte
	require.Eventually(t, func() bool {
		mu.Lock()
		all = slices.Clone(batches)
		mu.Unlock()
		n := 0
		for _, b := range all {
			var batch struct {
				Records []json.RawMessage `json:"records"`
			}
			require.NoError(t, json.Unmarshal(b, &batch))
			n += len(batch.Records)
		}
		return n >= 8
	}, 5*time.Second, 20*time.Millisecond)

	declared := schemaFields(t, schemaBytes)
	seen := map[string]bool{}
	for _, b := range all {
		result := schema.ValidateJSON(b)
		if !assert.True(t, result.IsValid(), string(b)) {
			details, err := json.Marshal(result.ToList())
			require.NoError(t, err)
			t.Log(string(details))
		}

		var batch map[string]any
		require.NoError(t, json.Unmarshal(b, &batch))
		for _, path := range jsonFields("", batch) {
			seen[path] = true
		}
	}

	var undeclared []string
	for path := range seen {
		if !declared[path] {
			undeclared = append(undeclared, path)
		}
	}
	sort.Strings(undeclared)
	assert.Empty(t, undeclared)

	// The traffic above reaches the TLS, HTTP/2 and HTTP/3 parts of the records.
	expected := []string{"records.tls.ja4", "records.tls.ja4r", "records.tls.ja3n", "records.http2.akamai", "records.http3.settings", "records.ja4h", "records.udp.ttl", "records.udp.dscp"}
	if runtime.GOOS == "linux" {
		expected = append(expected, "records.tcp.ja4t", "records.tcp.p0f", "records.tcp.syn")
	}
	for _, path := range expected {
		assert.True(t, seen[path], path)
	}
	if t.Failed() {
		for _, b := range all {
			t.Log(string(b))
		}
	}
}

// jsonFields returns the paths of the object fields in v, the array items sharing the path of their array.
func jsonFields(prefix string, v any) []string {
	var paths []string
	switch v := v.(type) {
	case map[string]any:
		for k, child := range v {
			path := k
			if prefix != "" {
				path = prefix + "." + k
			}
			paths = append(paths, path)
			paths = append(paths, jsonFields(path, child)...)
		}
	case []any:
		for _, child := range v {
			paths = append(paths, jsonFields(prefix, child)...)
		}
	}
	return paths
}

// schemaFields returns the paths of the properties the schema declares, following its local references.
func schemaFields(t *testing.T, schemaBytes []byte) map[string]bool {
	t.Helper()

	var root map[string]any
	require.NoError(t, json.Unmarshal(schemaBytes, &root))

	resolve := func(node map[string]any) map[string]any {
		for {
			ref, ok := node["$ref"].(string)
			if !ok {
				return node
			}
			next := any(root)
			for part := range strings.SplitSeq(strings.TrimPrefix(ref, "#/"), "/") {
				next = next.(map[string]any)[part]
			}
			node = next.(map[string]any)
		}
	}

	out := map[string]bool{}
	var walk func(prefix string, node map[string]any)
	walk = func(prefix string, node map[string]any) {
		node = resolve(node)
		for _, key := range []string{"anyOf", "oneOf", "allOf"} {
			if list, ok := node[key].([]any); ok {
				for _, item := range list {
					walk(prefix, item.(map[string]any))
				}
			}
		}
		if items, ok := node["items"].(map[string]any); ok {
			walk(prefix, items)
		}
		props, _ := node["properties"].(map[string]any)
		for k, child := range props {
			path := k
			if prefix != "" {
				path = prefix + "." + k
			}
			out[path] = true
			walk(path, child.(map[string]any))
		}
	}
	walk("", root)

	return out
}
