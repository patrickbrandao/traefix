package fingerprint

import (
	"encoding/binary"
	"strconv"
	"strings"
)

const (
	h2FrameSettingsType     = 0x4
	h2FramePriorityType     = 0x2
	h2FrameWindowUpdateType = 0x8
	h2FlagAck               = 0x1
	h2FrameHeaderLength     = 9
)

// AkamaiH2 returns the Akamai HTTP/2 fingerprint (SETTINGS|WINDOW_UPDATE|PRIORITY|pseudo-header order) from the
// client bytes after the preface up to the end of the first header block (Capture.H2Prefix), and the pseudo-header
// names of the first request (Request.PseudoNames, e.g. ":method"). Unparseable prefix → "".
//
// The format is the one of the Akamai white paper (Shuster and Gapanovich, "Passive Fingerprinting of HTTP/2
// Clients", Black Hat EU 2017), as computed by tls.peet.ws and curl-impersonate: the priority part only lists the
// PRIORITY frames, not the priority fields of the HEADERS frame, so that Chrome yields
// "1:65536;2:0;4:6291456;6:262144|15663105|0|m,a,s,p" although its HEADERS frame carries a priority.
// A PRIORITY weight is written as the wire value plus one, the effective weight of RFC 9113 section 5.3.2,
// as in the Firefox example of the paper.
func AkamaiH2(prefix []byte, pseudoNames []string) string {
	var settings, priorities []string
	windowUpdate := "00"
	seenWindowUpdate := false

	for len(prefix) > 0 {
		if len(prefix) < h2FrameHeaderLength {
			return ""
		}
		length := int(prefix[0])<<16 | int(prefix[1])<<8 | int(prefix[2])
		typ, flags := prefix[3], prefix[4]
		streamID := binary.BigEndian.Uint32(prefix[5:9]) & 0x7fffffff
		if len(prefix)-h2FrameHeaderLength < length {
			// A truncated capture still tells the frames before the last one.
			break
		}
		payload := prefix[h2FrameHeaderLength : h2FrameHeaderLength+length]
		prefix = prefix[h2FrameHeaderLength+length:]

		switch typ {
		case h2FrameSettingsType:
			if flags&h2FlagAck != 0 {
				continue
			}
			if len(payload)%6 != 0 {
				return ""
			}
			for i := 0; i < len(payload); i += 6 {
				id := binary.BigEndian.Uint16(payload[i:])
				value := binary.BigEndian.Uint32(payload[i+2:])
				settings = append(settings, strconv.Itoa(int(id))+":"+strconv.FormatUint(uint64(value), 10))
			}

		case h2FrameWindowUpdateType:
			if streamID != 0 || seenWindowUpdate {
				continue
			}
			if len(payload) != 4 {
				return ""
			}
			windowUpdate = strconv.FormatUint(uint64(binary.BigEndian.Uint32(payload)&0x7fffffff), 10)
			seenWindowUpdate = true

		case h2FramePriorityType:
			if len(payload) != 5 {
				return ""
			}
			dep := binary.BigEndian.Uint32(payload)
			exclusive := dep >> 31
			priorities = append(priorities, strconv.FormatUint(uint64(streamID), 10)+":"+
				strconv.FormatUint(uint64(exclusive), 10)+":"+
				strconv.FormatUint(uint64(dep&0x7fffffff), 10)+":"+
				strconv.Itoa(int(payload[4])+1))
		}
	}

	priority := "0"
	if len(priorities) > 0 {
		priority = strings.Join(priorities, ",")
	}

	pseudo := make([]string, 0, len(pseudoNames))
	for _, name := range pseudoNames {
		if len(name) > 1 && name[0] == ':' {
			pseudo = append(pseudo, name[1:2])
		}
	}

	return strings.Join(settings, ";") + "|" + windowUpdate + "|" + priority + "|" + strings.Join(pseudo, ",")
}

// H2HeadersPriority returns the priority fields of the first HEADERS frame in the client bytes after the preface,
// in the form of a priority entry of the Akamai fingerprint (stream:exclusive:dependency:weight), which leaves them
// out, e.g. "1:1:0:256" for Chrome. It returns "" when the frame has no priority fields, or is not there.
func H2HeadersPriority(prefix []byte) string {
	for len(prefix) >= h2FrameHeaderLength {
		length := int(prefix[0])<<16 | int(prefix[1])<<8 | int(prefix[2])
		typ, flags := prefix[3], prefix[4]
		streamID := binary.BigEndian.Uint32(prefix[5:9]) & 0x7fffffff
		if len(prefix)-h2FrameHeaderLength < length {
			return ""
		}
		payload := prefix[h2FrameHeaderLength : h2FrameHeaderLength+length]
		prefix = prefix[h2FrameHeaderLength+length:]

		if typ != h2FrameHeaders {
			continue
		}
		if flags&h2FlagPriority == 0 {
			return ""
		}
		if flags&h2FlagPadded != 0 {
			if len(payload) == 0 {
				return ""
			}
			payload = payload[1:]
		}
		if len(payload) < 5 {
			return ""
		}
		dep := binary.BigEndian.Uint32(payload)
		return strconv.FormatUint(uint64(streamID), 10) + ":" +
			strconv.FormatUint(uint64(dep>>31), 10) + ":" +
			strconv.FormatUint(uint64(dep&0x7fffffff), 10) + ":" +
			strconv.Itoa(int(payload[4])+1)
	}

	return ""
}

// firstHeaders returns the flags of the first HEADERS frame in the client bytes after the preface,
// and its priority fields as H2HeadersPriority formats them.
func firstHeaders(prefix []byte) (byte, string, bool) {
	for b := prefix; len(b) >= h2FrameHeaderLength; {
		length := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
		if len(b)-h2FrameHeaderLength < length {
			return 0, "", false
		}
		if b[3] == h2FrameHeaders {
			return b[4], H2HeadersPriority(b), true
		}
		b = b[h2FrameHeaderLength+length:]
	}
	return 0, "", false
}
