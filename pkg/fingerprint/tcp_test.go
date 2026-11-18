package fingerprint

import (
	"bufio"
	"encoding/json"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tcpHeader builds a TCP header with the given flags, window and options, padded to a multiple of 4 bytes.
func tcpHeader(flags uint8, window uint16, opts ...byte) []byte {
	for len(opts)%4 != 0 {
		opts = append(opts, 0)
	}
	h := make([]byte, 20, 20+len(opts))
	h[12] = byte((20+len(opts))/4) << 4
	h[13] = flags
	h[14], h[15] = byte(window>>8), byte(window)
	return append(h, opts...)
}

// ipv4Packet wraps a TCP header in an IPv4 header with the DF flag and a non-zero ID, as Linux sends a SYN.
func ipv4Packet(ttl uint8, tcp []byte) []byte {
	ip := make([]byte, 20, 20+len(tcp))
	ip[0] = 0x45
	total := 20 + len(tcp)
	ip[2], ip[3] = byte(total>>8), byte(total)
	ip[4], ip[5] = 0x12, 0x34
	ip[6] = 0x40
	ip[8] = ttl
	ip[9] = 6
	copy(ip[12:16], []byte{192, 0, 2, 1})
	copy(ip[16:20], []byte{192, 0, 2, 2})
	return append(ip, tcp...)
}

// The expected values follow the FoxIO tool reading the same options through tshark.
func TestJA4T(t *testing.T) {
	testCases := []struct {
		desc     string
		tcp      []byte
		expected string
	}{
		{desc: "Linux", tcp: tcpHeader(0x02, 64240, 2, 4, 5, 0xb4, 4, 2, 8, 10, 0, 0, 0, 1, 0, 0, 0, 0, 1, 3, 3, 7), expected: "64240_2-4-8-1-3_1460_7"},
		{desc: "Windows", tcp: tcpHeader(0x02, 64240, 2, 4, 5, 0xb4, 1, 3, 3, 8, 1, 1, 4, 2), expected: "64240_2-1-3-1-1-4_1460_8"},
		{desc: "no option", tcp: tcpHeader(0x02, 8192), expected: "8192_00_00_00"},
		{desc: "zero window scale", tcp: tcpHeader(0x02, 5744, 2, 4, 0x05, 0x9c, 4, 2, 8, 10, 0, 0, 0, 1, 0, 0, 0, 0, 1, 3, 3, 0), expected: "5744_2-4-8-1-3_1436_00"},
		{desc: "one digit MSS", tcp: tcpHeader(0x02, 8192, 2, 4, 0, 9), expected: "8192_2_09_00"},
		{desc: "EOL and zeros", tcp: tcpHeader(0x02, 65535, 2, 4, 5, 0xb4, 4, 2, 0, 0), expected: "65535_2-4-0-0_1460_00"},
		{desc: "EOL and garbage", tcp: tcpHeader(0x02, 1, 0, 1, 3, 3, 7, 0, 0, 0), expected: "1_0_00_00"},
		{desc: "MSS with a wrong length", tcp: tcpHeader(0x02, 1, 2, 3, 5, 1), expected: "1_2-1_00_00"},
		{desc: "zero length", tcp: tcpHeader(0x02, 1, 2, 4, 5, 0xb4, 30, 0, 1, 1), expected: "1_2_1460_00"},
		{desc: "past the end", tcp: tcpHeader(0x02, 1, 2, 4, 5, 0xb4, 1, 3, 9, 7), expected: "1_2-1_1460_00"},
		{desc: "last MSS wins", tcp: tcpHeader(0x02, 1, 2, 4, 5, 0xb4, 2, 4, 2, 0x18), expected: "1_2-2_536_00"},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			assert.Equal(t, test.expected, ja4t(test.tcp))
		})
	}
}

func TestParseSYN(t *testing.T) {
	linux := tcpHeader(0x02, 64240, 2, 4, 5, 0xb4, 4, 2, 8, 10, 0, 0, 0, 1, 0, 0, 0, 0, 1, 3, 3, 7)
	linux[7] = 1
	syn, err := ParseSYN(ipv4Packet(57, linux))
	require.NoError(t, err)

	assert.Equal(t, 4, syn.IPVersion)
	assert.Equal(t, 57, syn.TTL)
	assert.Equal(t, []byte{192, 0, 2, 1}, syn.Src)
	assert.Equal(t, "64240_2-4-8-1-3_1460_7", syn.JA4T)
	assert.Equal(t, "4:57+7:0:1460:mss*44,7:mss,sok,ts,nop,ws:df,id+:0", syn.P0F)

	_, err = ParseSYN(ipv4Packet(64, tcpHeader(0x12, 1)))
	require.ErrorIs(t, err, errSYNNotSYN)
	_, err = ParseSYN([]byte{0x45, 0})
	require.ErrorIs(t, err, errSYNShort)

	udp := ipv4Packet(64, tcpHeader(0x02, 1))
	udp[9] = 17
	_, err = ParseSYN(udp)
	require.ErrorIs(t, err, errSYNNotTCP)
}

// TestGolden_SYN compares the JA4T and the p0f signature of each client SYN of the captures with the outputs of
// the FoxIO tool and of p0f on the full captures.
func TestGolden_SYN(t *testing.T) {
	files, err := filepath.Glob("testdata/golden/tcp/*.pcap")
	require.NoError(t, err)
	require.Len(t, files, 10)

	var ja4tChecked, p0fChecked int
	for _, file := range files {
		name := strings.TrimSuffix(file, ".pcap")
		t.Run(filepath.Base(name), func(t *testing.T) {
			ja4ts := goldenJA4T(t, name+".ja4.jsonl")
			sigs := goldenP0F(t, name+".p0f.txt")

			seen := map[netip.AddrPort]bool{}
			for _, p := range readPcap(t, file) {
				if p.flags&0x12 != 0x02 || seen[p.src] {
					continue
				}
				seen[p.src] = true

				syn, err := ParseSYN(p.ip)
				require.NoError(t, err, p.src.String())
				assert.Equal(t, p.src.Addr().AsSlice(), syn.Src)

				if expected, ok := ja4ts[p.src]; ok {
					assert.Equal(t, expected, syn.JA4T, p.src.String())
					ja4tChecked++
				}
				if expected, ok := sigs[p.src]; ok {
					assert.Equal(t, expected, syn.P0F, p.src.String())
					p0fChecked++
				}
			}
		})
	}
	t.Logf("%d JA4T and %d p0f signatures checked", ja4tChecked, p0fChecked)
	assert.GreaterOrEqual(t, ja4tChecked, 15)
	assert.GreaterOrEqual(t, p0fChecked, 15)
}

func goldenJA4T(t *testing.T, file string) map[netip.AddrPort]string {
	t.Helper()

	f, err := os.Open(file)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	out := map[netip.AddrPort]string{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(nil, 1<<20)
	for scanner.Scan() {
		var line struct {
			Src     string `json:"src"`
			SrcPort uint16 `json:"src_port"`
			JA4T    string `json:"ja4t"`
		}
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &line))
		if line.JA4T == "" {
			continue
		}
		addr, err := netip.ParseAddr(line.Src)
		require.NoError(t, err)
		out[netip.AddrPortFrom(addr, line.SrcPort)] = line.JA4T
	}
	require.NoError(t, scanner.Err())

	return out
}

// goldenP0F reads the raw_sig of the first syn observation of each client in the output of p0f -r.
func goldenP0F(t *testing.T, file string) map[netip.AddrPort]string {
	t.Helper()

	f, err := os.Open(file)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	out := map[netip.AddrPort]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := map[string]string{}
		_, rest, _ := strings.Cut(scanner.Text(), "] ")
		for field := range strings.SplitSeq(rest, "|") {
			k, v, _ := strings.Cut(field, "=")
			fields[k] = v
		}
		if fields["mod"] != "syn" {
			continue
		}
		host, port, ok := strings.Cut(fields["cli"], "/")
		require.True(t, ok)
		addr, err := netip.ParseAddr(host)
		require.NoError(t, err)
		n, err := strconv.ParseUint(port, 10, 16)
		require.NoError(t, err)
		key := netip.AddrPortFrom(addr, uint16(n))
		if _, ok := out[key]; !ok {
			out[key] = fields["raw_sig"]
		}
	}
	require.NoError(t, scanner.Err())

	return out
}
