//go:build linux

package server

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
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
			// The loopback SYN carries the default TTL or hop limit of the host, which a sysctl may change.
			ttlFile := "/proc/sys/net/ipv4/ip_default_ttl"
			if network.name == "ipv6" {
				ttlFile = "/proc/sys/net/ipv6/conf/lo/hop_limit"
			}
			ttlBytes, err := os.ReadFile(ttlFile)
			require.NoError(t, err)
			ttl := strings.TrimSpace(string(ttlBytes))
			ttlValue, err := strconv.Atoi(ttl)
			require.NoError(t, err)

			assert.Equal(t, ttl, got["X-Edge-Ttl"])
			assert.Equal(t, network.host, got["X-Edge-Server-Ip"])

			kv := map[string]string{}
			for pair := range strings.SplitSeq(got["X-Edge-Transport"], ",") {
				k, v, ok := strings.Cut(pair, "=")
				require.True(t, ok, pair)
				kv[k] = v
			}
			ipv := "4"
			if network.name == "ipv6" {
				ipv = "6"
			}
			assert.Equal(t, "tcp", kv["proto"])
			assert.Equal(t, ipv, kv["ipv"])
			assert.Equal(t, ttl, kv["ttl"])
			assert.Equal(t, "0", kv["hops"])
			assert.Equal(t, "2-4-8-1-3", kv["syn_opts"])
			assert.Equal(t, "1", kv["syn_sack"])
			assert.Equal(t, "1", kv["opt_ts"])
			assert.NotEmpty(t, kv["syn_mss"])
			assert.NotEmpty(t, kv["mss_snd"])
			assert.NotEmpty(t, kv["pmtu"])
			assert.NotEmpty(t, kv["wscale_snd"])
			assert.NotEmpty(t, kv["cc"])
			assert.Equal(t, "1", kv["req_seq"])
			t.Log(got["X-Edge-Transport"])

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
			assert.InDelta(t, ttlValue, tcp["ttl"], 0)
			assert.NotEmpty(t, tcp["syn"])
			p0f, _ := tcp["p0f"].(string)
			version := "4"
			if network.name == "ipv6" {
				version = "6"
			}
			assert.True(t, strings.HasPrefix(p0f, version+":"+ttl+"+0:0:"), p0f)
			assert.Contains(t, p0f, ":mss,sok,ts,nop,ws:")
		})
	}
}
