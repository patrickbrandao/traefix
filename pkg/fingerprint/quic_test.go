package fingerprint

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/quic-go/quic-go"
	h3qlog "github.com/quic-go/quic-go/http3/qlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/static"
)

// TestQUIC_clientHelloSources checks that the ClientHello the QUIC library gives to the TLS callback and the one
// decrypted from the Initial packets give the same fingerprints.
func TestQUIC_clientHelloSources(t *testing.T) {
	for _, version := range []quic.Version{quic.Version1, quic.Version2} {
		t.Run(version.String(), func(t *testing.T) {
			ep, err := NewEntryPoint(t.Context(), "quic", &static.Fingerprint{
				HeaderPrefix: "X-Traefix-",
				Export:       &static.FingerprintExport{URL: "http://127.0.0.1:1", RawBytes: true, QueueSize: 1, QueueMaxBytes: 1, BatchSize: 1, FlushInterval: 1, SendTimeout: 1},
			}, nil, nil)
			require.NoError(t, err)
			t.Cleanup(func() { ep.Close(context.Background()) })

			pc, err := net.ListenPacket("udp", "127.0.0.1:0")
			require.NoError(t, err)
			t.Cleanup(func() { _ = pc.Close() })

			serverTLS := &tls.Config{GetConfigForClient: ep.QUICGetConfigForClient(func(*tls.ClientHelloInfo) (*tls.Config, error) {
				return &tls.Config{Certificates: []tls.Certificate{mustCert(t)}, NextProtos: []string{"h3"}}, nil
			})}
			ln, err := quic.Listen(ep.WrapPacketConn(pc), serverTLS, &quic.Config{Versions: []quic.Version{version}})
			require.NoError(t, err)
			t.Cleanup(func() { _ = ln.Close() })

			go func() {
				c, err := ln.Accept(context.Background())
				if err == nil {
					_ = c.CloseWithError(0, "")
				}
			}()

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			conn, err := quic.DialAddr(ctx, pc.LocalAddr().String(),
				&tls.Config{InsecureSkipVerify: true, ServerName: "example.com", NextProtos: []string{"h3"}},
				&quic.Config{Versions: []quic.Version{version}})
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.CloseWithError(0, "") })

			// The client listens on all addresses, the server sees it from 127.0.0.1: the single entry is the client.
			ep.quic.mu.Lock()
			require.Len(t, ep.quic.byKey, 1)
			var h *quicHello
			for _, v := range ep.quic.byKey {
				h = v
			}
			ep.quic.mu.Unlock()
			require.NotNil(t, h.fromInfo)
			require.NotNil(t, h.fromRaw)

			assert.Equal(t, h.fromRaw.JA4(true), h.fromInfo.JA4(true))
			assert.Equal(t, h.fromRaw.JA3(), h.fromInfo.JA3())
			assert.Regexp(t, `^q13d\d{4}h3_`, h.fromRaw.JA4(true))
			assert.Contains(t, h.params, uint64(0x04))
		})
	}
}

func TestQUICTrace_settings(t *testing.T) {
	testCases := []struct {
		desc     string
		frame    h3qlog.SettingsFrame
		expected string
	}{
		{
			desc: "wire order, GREASE included",
			frame: h3qlog.SettingsFrame{
				MaxFieldSectionSize: 16384, Datagram: -1, ExtendedConnect: 1,
				Other: map[uint64]uint64{0x1: 65536, 0x7: 100, 0x1f*7 + 0x21: 99},
				Order: []uint64{0x1f*7 + 0x21, 0x8, 0x1, 0x6, 0x7},
			},
			expected: "250:99;8:1;1:65536;6:16384;7:100",
		},
		{
			desc: "no order",
			frame: h3qlog.SettingsFrame{
				MaxFieldSectionSize: 16384, Datagram: 1, ExtendedConnect: -1,
				Other: map[uint64]uint64{0x7: 100, 0x1: 65536},
			},
			expected: "1:65536;6:16384;7:100;51:1",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			tr := &quicTrace{}
			tr.setSettings(test.frame)
			assert.Equal(t, test.expected, tr.Settings())
		})
	}
}
