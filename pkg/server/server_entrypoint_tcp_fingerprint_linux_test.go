//go:build linux

package server

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v3/pkg/config/static"
	tcprouter "github.com/traefik/traefik/v3/pkg/server/router/tcp"
)

// TestFingerprintEntryPoint_tcp reads the SYN the kernel saves for each connection, over IPv4 and IPv6 loopback,
// and checks the JA4T header and the TCP part of the connection record.
func TestFingerprintEntryPoint_tcp(t *testing.T) {
	for _, network := range []struct{ name, host string }{{"ipv4", "127.0.0.1"}, {"ipv6", "::1"}} {
		t.Run(network.name, func(t *testing.T) {
			ln, err := net.Listen("tcp", net.JoinHostPort(network.host, "0"))
			if err != nil {
				t.Skipf("no %s loopback: %v", network.name, err)
			}
			addr := ln.Addr().String()
			require.NoError(t, ln.Close())

			var mu sync.Mutex
			var conns []map[string]any
			dest := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
				var b struct {
					Records []map[string]any `json:"records"`
				}
				if err := json.NewDecoder(req.Body).Decode(&b); err == nil {
					mu.Lock()
					for _, r := range b.Records {
						if r["type"] == "connection" {
							conns = append(conns, r)
						}
					}
					mu.Unlock()
				}
			}))
			t.Cleanup(dest.Close)

			fp := &static.Fingerprint{HeaderPrefix: "X-Edge-", TCP: true, Export: &static.FingerprintExport{URL: dest.URL, RawBytes: true}}
			fp.Export.SetDefaults()
			fp.Export.FlushInterval = ptypes.Duration(20 * time.Millisecond)

			epConfig := &static.EntryPointsTransport{}
			epConfig.SetDefaults()
			entryPoint, err := NewTCPEntryPoint(t.Context(), "web", &static.EntryPoint{
				Address:          addr,
				Transport:        epConfig,
				ForwardedHeaders: &static.ForwardedHeaders{},
				HTTP2:            &static.HTTP2Config{},
				Fingerprint:      fp,
			}, nil, nil)
			require.NoError(t, err)

			router, err := tcprouter.NewRouter(nil)
			require.NoError(t, err)
			router.SetHTTPHandler(http.HandlerFunc(echoFingerprintHeaders))

			go entryPoint.Start(t.Context())
			entryPoint.SwitchRouter(router)
			t.Cleanup(func() { entryPoint.Shutdown(t.Context()) })

			transport := &http.Transport{}
			t.Cleanup(transport.CloseIdleConnections)
			resp, err := transport.RoundTrip(httptest.NewRequest(http.MethodGet, "http://"+addr+"/", nil))
			require.NoError(t, err)
			var got map[string]string
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()

			// Linux loopback: window 65495 (MSS 65483 for IPv4, 65476 for IPv6), options MSS, SACK permitted,
			// timestamps, NOP, window scale.
			assert.Regexp(t, `^\d+_2-4-8-1-3_\d+_\d+$`, got["X-Edge-Ja4t"])

			var rec map[string]any
			require.Eventually(t, func() bool {
				mu.Lock()
				defer mu.Unlock()
				if len(conns) == 0 {
					return false
				}
				rec = conns[0]
				return true
			}, 5*time.Second, 20*time.Millisecond)

			tcp, ok := rec["tcp"].(map[string]any)
			require.True(t, ok, rec)
			assert.Equal(t, got["X-Edge-Ja4t"], tcp["ja4t"])
			assert.InDelta(t, 64, tcp["ttl"], 0)
			assert.NotEmpty(t, tcp["syn"])
			p0f, _ := tcp["p0f"].(string)
			version := "4"
			if network.name == "ipv6" {
				version = "6"
			}
			assert.True(t, strings.HasPrefix(p0f, version+":64+0:0:"), p0f)
			assert.Contains(t, p0f, ":mss,sok,ts,nop,ws:")
		})
	}
}
