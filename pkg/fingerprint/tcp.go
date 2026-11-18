package fingerprint

import (
	"encoding/binary"
	"errors"
	"strconv"
	"strings"
)

// The TCP view of a connection comes from its SYN, as the Linux kernel saves it with TCP_SAVE_SYN: the IP header and
// the TCP header with its options. JA4T follows the FoxIO tool (rust/ja4/src/tcp.rs), which reads the tshark fields,
// and the p0f signature follows p0f 3.09b (process.c, fp_tcp.c). Both readings were checked against the outputs of
// these tools on the captures of testdata/golden/tcp.

const (
	tcpFlagFIN = 0x01
	tcpFlagSYN = 0x02
	tcpFlagRST = 0x04
	tcpFlagPSH = 0x08
	tcpFlagACK = 0x10
	tcpFlagURG = 0x20
	tcpFlagECE = 0x40
	tcpFlagCWR = 0x80

	tcpOptEOL    = 0
	tcpOptNOP    = 1
	tcpOptMSS    = 2
	tcpOptWScale = 3
	tcpOptSACKOK = 4
	tcpOptSACK   = 5
	tcpOptTS     = 8

	// p0fMaxOptions and p0fMaxDist are the MAX_TCP_OPT and MAX_DIST of p0f.
	p0fMaxOptions = 24
	p0fMaxDist    = 35

	minIPv4TCP = 40
	minIPv6TCP = 60
)

// The p0f quirks, in the order of the signature.
const (
	quirkDF uint32 = 1 << iota
	quirkNZID
	quirkZeroID
	quirkECN
	quirkNZMBZ
	quirkFlow
	quirkZeroSeq
	quirkNZAck
	quirkZeroAck
	quirkNZURG
	quirkURG
	quirkPush
	quirkZeroTS1
	quirkNZTS2
	quirkOptEOLNZ
	quirkOptExWS
	quirkOptBad
)

var quirkNames = []string{
	"df", "id+", "id-", "ecn", "0+", "flow", "seq-", "ack+", "ack-", "uptr+", "urgf+", "pushf+", "ts1-", "ts2+",
	"opt+", "exws", "bad",
}

var (
	errSYNShort   = errors.New("short or inconsistent SYN packet")
	errSYNNotTCP  = errors.New("not a TCP packet")
	errSYNNotSYN  = errors.New("not an initial SYN")
	errSYNVersion = errors.New("unknown IP version")
)

// SYN is what Traefix reads from the SYN of a connection.
type SYN struct {
	// IPVersion is 4 or 6.
	IPVersion int
	// Src is the client address of the IP header.
	Src []byte
	// TTL is the TTL or hop limit as received.
	TTL int
	// JA4T is the FoxIO JA4T fingerprint, e.g. 64240_2-4-8-1-3_1460_7.
	JA4T string
	// P0F is the p0f v3 signature of the SYN, e.g. 4:64+0:0:1460:mss*44,7:mss,sok,ts,nop,ws:df,id+:0.
	P0F string
}

// ParseSYN reads the SYN packet, from its IP header.
func ParseSYN(packet []byte) (*SYN, error) {
	p, err := parseP0FPacket(packet)
	if err != nil {
		return nil, err
	}
	if p.flags&(tcpFlagSYN|tcpFlagACK) != tcpFlagSYN {
		return nil, errSYNNotSYN
	}

	return &SYN{
		IPVersion: p.ipVersion,
		Src:       p.src,
		TTL:       int(p.ttl),
		JA4T:      ja4t(p.tcp),
		P0F:       p.signature(),
	}, nil
}

// ja4t formats the window, the option kinds, the MSS and the window scale of a TCP header, as the FoxIO tool does
// with the tshark fields: no option gives "00", the MSS has two digits at least, a missing or zero window scale
// gives "00".
func ja4t(tcp []byte) string {
	window := binary.BigEndian.Uint16(tcp[14:16])
	end := int(tcp[12]>>4) * 4
	have := len(tcp)

	mss, wscale := -1, -1
	var kinds []byte
	i := 20
options:
	for i < end && i < have {
		k := tcp[i]
		switch k {
		case tcpOptEOL:
			// tshark lists the EOL and each zero byte after it as another EOL, and stops at the first other byte.
			kinds = append(kinds, tcpOptEOL)
			i++
			for i < end && i < have && tcp[i] == 0 {
				kinds = append(kinds, tcpOptEOL)
				i++
			}
			break options
		case tcpOptNOP:
			kinds = append(kinds, tcpOptNOP)
			i++
			continue
		}

		// An option without its length byte, with a length below 2, or past the end of the options stops tshark
		// without listing it.
		if i+1 >= end || i+1 >= have {
			break
		}
		l := int(tcp[i+1])
		if l < 2 || i+l > end {
			break
		}
		kinds = append(kinds, k)
		if i+l > have {
			break
		}
		switch {
		case k == tcpOptMSS && l == 4:
			mss = int(binary.BigEndian.Uint16(tcp[i+2 : i+4]))
		case k == tcpOptWScale && l == 3:
			wscale = int(tcp[i+2])
		}
		i += l
	}

	var b strings.Builder
	b.Grow(32)
	b.WriteString(strconv.Itoa(int(window)))
	b.WriteByte('_')
	if len(kinds) == 0 {
		b.WriteString("00")
	}
	for i, k := range kinds {
		if i > 0 {
			b.WriteByte('-')
		}
		b.WriteString(strconv.Itoa(int(k)))
	}
	b.WriteByte('_')
	mss = max(mss, 0)
	if mss < 10 {
		b.WriteByte('0')
	}
	b.WriteString(strconv.Itoa(mss))
	b.WriteByte('_')
	if wscale <= 0 {
		b.WriteString("00")
	} else {
		b.WriteString(strconv.Itoa(wscale))
	}

	return b.String()
}

// p0fPacket holds the fields of a packet that p0f reads (struct packet_data).
type p0fPacket struct {
	ipVersion int
	src       []byte
	ttl       uint8
	ipOptLen  uint8
	totHdr    int
	quirks    uint32
	flags     uint8
	window    uint16
	seq       uint32
	layout    []uint8
	eolPad    uint8
	mss       uint16
	wscale    uint8
	ts1       uint32
	payLen    int
	tcp       []byte
}

// parseP0FPacket reads a packet from its IP header, as parse_packet of p0f does.
func parseP0FPacket(ip []byte) (*p0fPacket, error) {
	n := min(len(ip), 65535)
	if n < minIPv4TCP {
		return nil, errSYNShort
	}

	p := &p0fPacket{}
	var tcp []byte
	switch ip[0] >> 4 {
	case 4:
		hl := int(ip[0]&0x0f) * 4
		flagsOff := binary.BigEndian.Uint16(ip[6:8])
		total := int(binary.BigEndian.Uint16(ip[2:4]))
		n = min(n, total)
		if n < minIPv4TCP || hl < 20 || total > n || hl+20 > n {
			return nil, errSYNShort
		}
		if ip[9] != 6 {
			return nil, errSYNNotTCP
		}
		p.ipVersion = 4
		p.src = ip[12:16]
		p.ipOptLen = uint8(hl - 20)
		p.ttl = ip[8]
		if ip[1]&0x03 != 0 {
			p.quirks |= quirkECN
		}
		if flagsOff&0x8000 != 0 {
			p.quirks |= quirkNZMBZ
		}
		id := binary.BigEndian.Uint16(ip[4:6])
		if flagsOff&0x4000 != 0 {
			p.quirks |= quirkDF
			if id != 0 {
				p.quirks |= quirkNZID
			}
		} else if id == 0 {
			p.quirks |= quirkZeroID
		}
		p.totHdr = hl
		tcp = ip[hl:n]
	case 6:
		verTOS := binary.BigEndian.Uint32(ip[0:4])
		total := int(binary.BigEndian.Uint16(ip[4:6])) + 40
		n = min(n, total)
		if n < minIPv6TCP || total > n {
			return nil, errSYNShort
		}
		// As p0f, the extension headers are not followed.
		if ip[6] != 6 {
			return nil, errSYNNotTCP
		}
		p.ipVersion = 6
		p.src = ip[8:24]
		p.ttl = ip[7]
		if verTOS&0xfffff != 0 {
			p.quirks |= quirkFlow
		}
		if (verTOS>>20)&0x03 != 0 {
			p.quirks |= quirkECN
		}
		p.totHdr = 40
		tcp = ip[40:n]
	default:
		return nil, errSYNVersion
	}

	doff := int(tcp[12]>>4) * 4
	if doff < 20 || doff > len(tcp) {
		return nil, errSYNShort
	}
	p.tcp = tcp[:doff]
	p.totHdr += doff

	flags := tcp[13]
	p.flags = flags
	p.window = binary.BigEndian.Uint16(tcp[14:16])
	p.seq = binary.BigEndian.Uint32(tcp[4:8])
	if flags&(tcpFlagECE|tcpFlagCWR) != 0 || tcp[12]&0x01 != 0 {
		p.quirks |= quirkECN
	}
	if p.seq == 0 {
		p.quirks |= quirkZeroSeq
	}
	ack := tcp[8:12]
	if flags&tcpFlagACK != 0 {
		if binary.BigEndian.Uint32(ack) == 0 {
			p.quirks |= quirkZeroAck
		}
	} else if ack[0]&0x01 != 0 && flags&tcpFlagRST == 0 {
		// p0f tests RD32(tcp->ack) & !(flags & RST), a bitwise AND with 0 or 1 of the value read in host order
		// (little-endian): the low bit of the first byte of the field. Kept so that the signature is the same.
		p.quirks |= quirkNZAck
	}
	if flags&tcpFlagURG != 0 {
		p.quirks |= quirkURG
	} else if binary.BigEndian.Uint16(tcp[18:20]) != 0 {
		p.quirks |= quirkNZURG
	}
	if flags&tcpFlagPSH != 0 {
		p.quirks |= quirkPush
	}
	p.payLen = len(tcp) - doff
	p.parseOptions(tcp[20:doff])

	return p, nil
}

// parseOptions follows the option loop of p0f, including the cases that set the bad and opt+ quirks.
func (p *p0fPacket) parseOptions(opt []byte) {
	end := len(opt)
	i := 0
	for i < end && len(p.layout) < p0fMaxOptions {
		k := opt[i]
		p.layout = append(p.layout, k)
		i++
		switch k {
		case tcpOptEOL:
			// As p0f (while (d < end && !*d++)), the cursor also moves past the first non-zero byte: when it is the
			// last byte, opt+ is not set.
			p.eolPad = uint8(end - i)
			for i < end {
				b := opt[i]
				i++
				if b != 0 {
					break
				}
			}
			if i != end {
				p.quirks |= quirkOptEOLNZ
				i = end
			}
		case tcpOptNOP:
		case tcpOptMSS:
			if i+3 > end {
				p.quirks |= quirkOptBad
				return
			}
			if opt[i] != 4 {
				p.quirks |= quirkOptBad
			}
			p.mss = binary.BigEndian.Uint16(opt[i+1 : i+3])
			i += 3
		case tcpOptWScale:
			if i+2 > end {
				p.quirks |= quirkOptBad
				return
			}
			if opt[i] != 3 {
				p.quirks |= quirkOptBad
			}
			p.wscale = opt[i+1]
			if p.wscale > 14 {
				p.quirks |= quirkOptExWS
			}
			i += 2
		case tcpOptSACKOK:
			if i+1 > end {
				p.quirks |= quirkOptBad
				return
			}
			if opt[i] != 2 {
				p.quirks |= quirkOptBad
			}
			i++
		case tcpOptSACK:
			if i == end || opt[i] < 10 || opt[i] > 34 || i-1+int(opt[i]) > end {
				p.quirks |= quirkOptBad
				return
			}
			i += int(opt[i]) - 1
		case tcpOptTS:
			if i+9 > end {
				p.quirks |= quirkOptBad
				return
			}
			if opt[i] != 10 {
				p.quirks |= quirkOptBad
			}
			p.ts1 = binary.BigEndian.Uint32(opt[i+1 : i+5])
			if p.ts1 == 0 {
				p.quirks |= quirkZeroTS1
			}
			if p.flags&(tcpFlagSYN|tcpFlagACK|tcpFlagFIN|tcpFlagRST) == tcpFlagSYN && binary.BigEndian.Uint32(opt[i+5:i+9]) != 0 {
				p.quirks |= quirkNZTS2
			}
			i += 9
		default:
			if i == end || opt[i] < 2 || opt[i] > 40 || i-1+int(opt[i]) > end {
				p.quirks |= quirkOptBad
				return
			}
			i += int(opt[i]) - 1
		}
	}
	if i != end {
		p.quirks |= quirkOptBad
	}
}

// guessDist returns the distance to the closest usual initial TTL: 32, 64, 128 or 255.
func guessDist(ttl uint8) int {
	switch {
	case ttl <= 32:
		return 32 - int(ttl)
	case ttl <= 64:
		return 64 - int(ttl)
	case ttl <= 128:
		return 128 - int(ttl)
	}
	return 255 - int(ttl)
}

// windowMultiple tells whether the window is a multiple of the MSS or of the MTU, as detect_win_multi of p0f does
// for a SYN: it returns the multiplier, or -1.
func (p *p0fPacket) windowMultiple() (multi int, mtu bool) {
	win := int(p.window)
	mss := int(p.mss)
	if win == 0 || mss < 100 {
		return -1, false
	}

	try := func(div int, isMTU bool) bool {
		if div != 0 && win%div == 0 {
			// As in C, the quotient is a signed 16-bit value.
			multi, mtu = int(int16(win/div)), isMTU //nolint:gosec // Same conversion as p0f.
			return true
		}
		return false
	}
	switch {
	case try(mss, false):
	case p.ts1 != 0 && try(mss-12, false):
	case try(1500-minIPv4TCP, false):
	case try(1500-minIPv4TCP-12, false):
	case p.ipVersion == 6 && try(1500-minIPv6TCP, false):
	case p.ipVersion == 6 && try(1500-minIPv6TCP-12, false):
	case try(mss+minIPv4TCP, true):
	case try(mss+p.totHdr, true):
	case p.ipVersion == 6 && try(mss+minIPv6TCP, true):
	case try(1500, true):
	default:
		return -1, false
	}

	return multi, mtu
}

// signature writes the packet signature in the p0f.fp format, as dump_sig of p0f does.
func (p *p0fPacket) signature() string {
	var b strings.Builder
	b.Grow(96)
	u := func(v int) { b.WriteString(strconv.Itoa(v)) }

	u(p.ipVersion)
	b.WriteByte(':')
	u(int(p.ttl))
	b.WriteByte('+')
	if d := guessDist(p.ttl); d > p0fMaxDist {
		b.WriteByte('?')
	} else {
		u(d)
	}
	b.WriteByte(':')
	u(int(p.ipOptLen))
	b.WriteByte(':')
	u(int(p.mss))
	b.WriteByte(':')
	if multi, mtu := p.windowMultiple(); multi > 0 {
		if mtu {
			b.WriteString("mtu*")
		} else {
			b.WriteString("mss*")
		}
		u(multi)
	} else {
		u(int(p.window))
	}
	b.WriteByte(',')
	u(int(p.wscale))
	b.WriteByte(':')
	for i, k := range p.layout {
		if i > 0 {
			b.WriteByte(',')
		}
		switch k {
		case tcpOptEOL:
			b.WriteString("eol+")
			u(int(p.eolPad))
		case tcpOptNOP:
			b.WriteString("nop")
		case tcpOptMSS:
			b.WriteString("mss")
		case tcpOptWScale:
			b.WriteString("ws")
		case tcpOptSACKOK:
			b.WriteString("sok")
		case tcpOptSACK:
			b.WriteString("sack")
		case tcpOptTS:
			b.WriteString("ts")
		default:
			b.WriteByte('?')
			u(int(k))
		}
	}
	b.WriteByte(':')
	first := true
	for i, name := range quirkNames {
		if p.quirks&(1<<i) == 0 {
			continue
		}
		if !first {
			b.WriteByte(',')
		}
		first = false
		b.WriteString(name)
	}
	if p.payLen > 0 {
		b.WriteString(":+")
	} else {
		b.WriteString(":0")
	}

	return b.String()
}
