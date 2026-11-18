package fingerprint

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net/netip"
	"os"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// pcapPacket is a TCP segment read from a classic pcap file with Ethernet frames.
type pcapPacket struct {
	src, dst netip.AddrPort
	// ip holds the IP packet from its header, as the kernel saves a SYN.
	ip []byte
	// tcp holds the TCP header, options included.
	tcp     []byte
	flags   byte
	seq     uint32
	payload []byte
}

// readPcap returns the TCP segments of a pcap file, the other packets left out.
func readPcap(t *testing.T, file string) []pcapPacket {
	t.Helper()

	data, err := os.ReadFile(file)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(data), 24)

	var order binary.ByteOrder
	switch {
	case bytes.Equal(data[:4], []byte{0xd4, 0xc3, 0xb2, 0xa1}), bytes.Equal(data[:4], []byte{0x4d, 0x3c, 0xb2, 0xa1}):
		order = binary.LittleEndian
	case bytes.Equal(data[:4], []byte{0xa1, 0xb2, 0xc3, 0xd4}), bytes.Equal(data[:4], []byte{0xa1, 0xb2, 0x3c, 0x4d}):
		order = binary.BigEndian
	default:
		t.Fatalf("%s: not a classic pcap file", file)
	}
	require.Equal(t, uint32(1), order.Uint32(data[20:24]), "Ethernet link type")

	var packets []pcapPacket
	for rest := data[24:]; len(rest) >= 16; {
		n := int(order.Uint32(rest[8:12]))
		require.LessOrEqual(t, 16+n, len(rest))
		frame := rest[16 : 16+n]
		rest = rest[16+n:]

		if p, err := parseEthernet(frame); err == nil {
			packets = append(packets, p)
		}
	}

	return packets
}

func parseEthernet(frame []byte) (pcapPacket, error) {
	if len(frame) < 14 {
		return pcapPacket{}, errors.New("short frame")
	}
	etherType := binary.BigEndian.Uint16(frame[12:14])
	ip := frame[14:]
	for etherType == 0x8100 && len(ip) >= 4 {
		etherType = binary.BigEndian.Uint16(ip[2:4])
		ip = ip[4:]
	}

	var p pcapPacket
	var tcp []byte
	switch etherType {
	case 0x0800:
		if len(ip) < 20 || ip[9] != 6 {
			return p, errors.New("not TCP over IPv4")
		}
		hl := int(ip[0]&0x0f) * 4
		total := int(binary.BigEndian.Uint16(ip[2:4]))
		if total > len(ip) || hl > total {
			return p, errors.New("truncated IPv4")
		}
		ip = ip[:total]
		src, _ := netip.AddrFromSlice(ip[12:16])
		dst, _ := netip.AddrFromSlice(ip[16:20])
		p.src, p.dst = netip.AddrPortFrom(src, 0), netip.AddrPortFrom(dst, 0)
		tcp = ip[hl:]
	case 0x86dd:
		if len(ip) < 40 || ip[6] != 6 {
			return p, errors.New("not TCP over IPv6")
		}
		total := 40 + int(binary.BigEndian.Uint16(ip[4:6]))
		if total > len(ip) {
			return p, errors.New("truncated IPv6")
		}
		ip = ip[:total]
		src, _ := netip.AddrFromSlice(ip[8:24])
		dst, _ := netip.AddrFromSlice(ip[24:40])
		p.src, p.dst = netip.AddrPortFrom(src, 0), netip.AddrPortFrom(dst, 0)
		tcp = ip[40:]
	default:
		return p, errors.New("not IP")
	}

	if len(tcp) < 20 {
		return p, errors.New("short TCP header")
	}
	doff := int(tcp[12]>>4) * 4
	if doff < 20 || doff > len(tcp) {
		return p, errors.New("bad TCP data offset")
	}

	p.ip = ip
	p.src = netip.AddrPortFrom(p.src.Addr(), binary.BigEndian.Uint16(tcp[0:2]))
	p.dst = netip.AddrPortFrom(p.dst.Addr(), binary.BigEndian.Uint16(tcp[2:4]))
	p.tcp = tcp[:doff]
	p.flags = tcp[13]
	p.seq = binary.BigEndian.Uint32(tcp[4:8])
	p.payload = tcp[doff:]

	return p, nil
}

// pcapStream is a TCP connection of a capture, numbered like the tcp.stream field of Wireshark.
type pcapStream struct {
	index          int
	client, server netip.AddrPort
	syn            *pcapPacket
	segments       map[uint32][]byte
}

func (s *pcapStream) String() string {
	return strconv.Itoa(s.index) + " " + s.client.String() + " > " + s.server.String()
}

// clientBytes returns the client payload in sequence order, the retransmissions counted once.
func (s *pcapStream) clientBytes() []byte {
	seqs := make([]uint32, 0, len(s.segments))
	for seq := range s.segments {
		seqs = append(seqs, seq)
	}
	slices.Sort(seqs)

	var out []byte
	var next uint32
	for i, seq := range seqs {
		seg := s.segments[seq]
		if i > 0 && seq != next {
			if seq < next {
				// An overlapping retransmission: keep the bytes after what is already there.
				if skip := next - seq; int(skip) < len(seg) {
					out = append(out, seg[skip:]...)
					next = seq + uint32(len(seg))
				}
				continue
			}
			break
		}
		out = append(out, seg...)
		next = seq + uint32(len(seg))
	}

	return out
}

// pcapStreams groups the segments by connection, in the order of their first packet.
func pcapStreams(packets []pcapPacket) []*pcapStream {
	byKey := map[string]*pcapStream{}
	var streams []*pcapStream
	key := func(a, b netip.AddrPort) string {
		if a.String() > b.String() {
			a, b = b, a
		}
		return a.String() + "|" + b.String()
	}

	for i := range packets {
		p := &packets[i]
		k := key(p.src, p.dst)
		s := byKey[k]
		if s == nil {
			s = &pcapStream{index: len(streams), client: p.src, server: p.dst, segments: map[uint32][]byte{}}
			if p.flags&0x12 == 0x12 {
				// The capture starts with the SYN-ACK: the sender is the server.
				s.client, s.server = p.dst, p.src
			}
			byKey[k] = s
			streams = append(streams, s)
		}

		if p.flags&0x12 == 0x02 && p.src == s.client && s.syn == nil {
			s.syn = p
		}
		if p.src == s.client && len(p.payload) > 0 {
			if _, ok := s.segments[p.seq]; !ok {
				s.segments[p.seq] = p.payload
			}
		}
	}

	return streams
}
