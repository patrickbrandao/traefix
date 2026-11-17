package fingerprint

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
)

func TestAkamaiH2(t *testing.T) {
	testCases := []struct {
		desc     string
		frames   func(fr *http2.Framer) error
		pseudo   []string
		expected string
	}{
		{
			// Chrome, as reported by tls.peet.ws: the priority of its HEADERS frame is not part of the fingerprint.
			desc: "Chrome",
			frames: func(fr *http2.Framer) error {
				if err := fr.WriteSettings(
					http2.Setting{ID: http2.SettingHeaderTableSize, Val: 65536},
					http2.Setting{ID: http2.SettingEnablePush, Val: 0},
					http2.Setting{ID: http2.SettingInitialWindowSize, Val: 6291456},
					http2.Setting{ID: http2.SettingMaxHeaderListSize, Val: 262144},
				); err != nil {
					return err
				}
				if err := fr.WriteWindowUpdate(0, 15663105); err != nil {
					return err
				}
				return fr.WriteHeaders(http2.HeadersFrameParam{
					StreamID: 1, BlockFragment: []byte{0x82}, EndHeaders: true, EndStream: true,
					Priority: http2.PriorityParam{Exclusive: true, Weight: 255},
				})
			},
			pseudo:   []string{":method", ":authority", ":scheme", ":path"},
			expected: "1:65536;2:0;4:6291456;6:262144|15663105|0|m,a,s,p",
		},
		{
			// Firefox, example of the Akamai white paper (Black Hat EU 2017).
			desc: "Firefox",
			frames: func(fr *http2.Framer) error {
				if err := fr.WriteSettings(
					http2.Setting{ID: http2.SettingHeaderTableSize, Val: 65536},
					http2.Setting{ID: http2.SettingInitialWindowSize, Val: 131072},
					http2.Setting{ID: http2.SettingMaxFrameSize, Val: 16384},
				); err != nil {
					return err
				}
				if err := fr.WriteWindowUpdate(0, 12517377); err != nil {
					return err
				}
				for _, p := range []struct {
					stream, dep uint32
					weight      uint8
				}{{3, 0, 200}, {5, 0, 100}, {7, 0, 0}, {9, 7, 0}, {11, 3, 0}, {13, 0, 240}} {
					if err := fr.WritePriority(p.stream, http2.PriorityParam{StreamDep: p.dep, Weight: p.weight}); err != nil {
						return err
					}
				}
				return fr.WriteHeaders(http2.HeadersFrameParam{StreamID: 15, BlockFragment: []byte{0x82}, EndHeaders: true, EndStream: true})
			},
			pseudo:   []string{":method", ":path", ":authority", ":scheme"},
			expected: "1:65536;4:131072;5:16384|12517377|3:0:0:201,5:0:0:101,7:0:0:1,9:0:7:1,11:0:3:1,13:0:0:241|m,p,a,s",
		},
		{
			desc: "no connection window update and settings ack",
			frames: func(fr *http2.Framer) error {
				if err := fr.WriteSettings(http2.Setting{ID: http2.SettingMaxConcurrentStreams, Val: 100}); err != nil {
					return err
				}
				if err := fr.WriteSettingsAck(); err != nil {
					return err
				}
				// The window update of a stream is not the one of the connection.
				if err := fr.WriteWindowUpdate(1, 1000); err != nil {
					return err
				}
				return fr.WriteHeaders(http2.HeadersFrameParam{StreamID: 1, BlockFragment: []byte{0x82}, EndHeaders: true})
			},
			pseudo:   []string{":method", ":scheme", ":authority", ":path"},
			expected: "3:100|00|0|m,s,a,p",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			var buf bytes.Buffer
			require.NoError(t, test.frames(http2.NewFramer(&buf, nil)))

			assert.Equal(t, test.expected, AkamaiH2(buf.Bytes(), test.pseudo))
		})
	}
}

func TestAkamaiH2_malformed(t *testing.T) {
	// A SETTINGS payload whose length is not a multiple of 6.
	assert.Empty(t, AkamaiH2([]byte{0, 0, 7, 4, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7}, nil))
	assert.Empty(t, AkamaiH2([]byte{0, 0}, nil))
	assert.Equal(t, "|00|0|", AkamaiH2(nil, nil))
}

func FuzzAkamaiH2(f *testing.F) {
	f.Add([]byte{0, 0, 6, 4, 0, 0, 0, 0, 0, 0, 1, 0, 0, 16, 0})
	f.Fuzz(func(t *testing.T, prefix []byte) {
		_ = AkamaiH2(prefix, []string{":method", ":path"})
	})
}
