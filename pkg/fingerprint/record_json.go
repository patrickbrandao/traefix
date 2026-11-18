package fingerprint

import (
	"encoding/base64"
	"strconv"
	"time"
	"unicode/utf8"
)

// The records are encoded by hand, as encoding/json reflection costs several times more for each request,
// and the export runs for every request of the entry point. The output follows the json tags of the record types,
// which the tests decode with encoding/json.

type jsonAppender interface {
	appendJSON(b []byte) []byte
}

func (r *connectionRecord) appendJSON(b []byte) []byte {
	b = append(b, `{"type":`...)
	b = appendJSONString(b, r.Type)
	b = append(b, `,"connId":`...)
	b = appendJSONString(b, r.ConnID)
	b = append(b, `,"time":`...)
	b = appendJSONTime(b, r.Time)
	b = append(b, `,"entryPoint":`...)
	b = appendJSONString(b, r.EntryPoint)
	b = append(b, `,"src":`...)
	b = r.Src.appendJSON(b)
	b = append(b, `,"dst":`...)
	b = r.Dst.appendJSON(b)
	b = appendOptString(b, "proto", r.Proto)

	if t := r.TLS; t != nil {
		b = append(b, `,"tls":{`...)
		start := len(b)
		b = appendOptString(b, "sni", t.SNI)
		b = appendOptStrings(b, "alpnOffered", t.ALPNOffered)
		b = appendOptString(b, "alpn", t.ALPN)
		b = appendOptString(b, "version", t.Version)
		b = appendOptString(b, "cipher", t.Cipher)
		b = appendOptString(b, "ja3", t.JA3)
		b = appendOptString(b, "ja3String", t.JA3String)
		b = appendOptString(b, "ja4", t.JA4)
		b = appendOptBytes(b, "clientHello", t.ClientHello)
		b = appendOptBool(b, "clientHelloTruncated", t.ClientHelloTruncated)
		if len(t.QUICTransportParameters) > 0 {
			b = append(appendKey(b, "quicTransportParameters"), '[')
			for i, id := range t.QUICTransportParameters {
				if i > 0 {
					b = append(b, ',')
				}
				b = strconv.AppendUint(b, id, 10)
			}
			b = append(b, ']')
		}
		b = closeObject(b, start)
	}

	if h := r.HTTP2; h != nil {
		b = append(b, `,"http2":{`...)
		start := len(b)
		b = appendOptString(b, "akamai", h.Akamai)
		b = appendOptBytes(b, "prefix", h.Prefix)
		b = appendOptBool(b, "prefixTruncated", h.PrefixTruncated)
		b = closeObject(b, start)
	}

	if h := r.HTTP3; h != nil {
		b = append(b, `,"http3":{`...)
		start := len(b)
		b = appendOptString(b, "settings", h.Settings)
		b = closeObject(b, start)
	}

	return append(b, '}')
}

func (r *requestRecord) appendJSON(b []byte) []byte {
	b = append(b, `{"type":`...)
	b = appendJSONString(b, r.Type)
	b = appendOptString(b, "connId", r.ConnID)
	b = append(b, `,"start":`...)
	b = appendJSONTime(b, r.Start)
	b = append(b, `,"end":`...)
	b = appendJSONTime(b, r.End)
	b = append(b, `,"entryPoint":`...)
	b = appendJSONString(b, r.EntryPoint)
	b = append(b, `,"proto":`...)
	b = appendJSONString(b, r.Proto)
	b = append(b, `,"method":`...)
	b = appendJSONString(b, r.Method)
	b = appendOptString(b, "authority", r.Authority)
	b = append(b, `,"path":`...)
	b = appendJSONString(b, r.Path)
	b = append(b, `,"httpVersion":`...)
	b = appendJSONString(b, r.HTTPVersion)
	b = appendOptStrings(b, "fieldNames", r.FieldNames)

	if len(r.Fields) > 0 {
		b = append(b, `,"fields":[`...)
		for i, f := range r.Fields {
			if i > 0 {
				b = append(b, ',')
			}
			b = append(b, `{"name":`...)
			b = appendJSONString(b, f.Name)
			b = append(b, `,"value":`...)
			b = appendJSONString(b, f.Value)
			b = append(b, '}')
		}
		b = append(b, ']')
	}

	b = appendOptStrings(b, "cookieNames", r.CookieNames)
	b = appendOptString(b, "ja4h", r.JA4H)
	b = appendOptBool(b, "headersTruncated", r.HeadersTruncated)
	b = append(b, `,"status":`...)
	b = strconv.AppendInt(b, int64(r.Status), 10)
	b = appendOptString(b, "router", r.Router)
	b = appendOptString(b, "service", r.Service)

	return append(b, '}')
}

func (a addrRecord) appendJSON(b []byte) []byte {
	b = append(b, `{"ip":`...)
	b = appendJSONString(b, a.IP)
	b = append(b, `,"port":`...)
	b = strconv.AppendUint(b, uint64(a.Port), 10)
	return append(b, '}')
}

// closeObject ends an object opened at start, whose fields all start with a comma.
func closeObject(b []byte, start int) []byte {
	if len(b) > start {
		// The first field must not start with a comma.
		copy(b[start:], b[start+1:])
		b = b[:len(b)-1]
	}
	return append(b, '}')
}

func appendKey(b []byte, key string) []byte {
	b = append(b, ',', '"')
	b = append(b, key...)
	return append(b, '"', ':')
}

func appendOptString(b []byte, key, value string) []byte {
	if value == "" {
		return b
	}
	return appendJSONString(appendKey(b, key), value)
}

func appendOptStrings(b []byte, key string, values []string) []byte {
	if len(values) == 0 {
		return b
	}
	b = append(appendKey(b, key), '[')
	for i, v := range values {
		if i > 0 {
			b = append(b, ',')
		}
		b = appendJSONString(b, v)
	}
	return append(b, ']')
}

func appendOptBytes(b []byte, key string, value []byte) []byte {
	if len(value) == 0 {
		return b
	}
	b = append(appendKey(b, key), '"')
	b = base64.StdEncoding.AppendEncode(b, value)
	return append(b, '"')
}

func appendOptBool(b []byte, key string, value bool) []byte {
	if !value {
		return b
	}
	return append(appendKey(b, key), "true"...)
}

func appendJSONTime(b []byte, t time.Time) []byte {
	b = append(b, '"')
	b = t.AppendFormat(b, time.RFC3339Nano)
	return append(b, '"')
}

const hexDigits = "0123456789abcdef"

// appendJSONString appends s as a JSON string, replacing the invalid UTF-8 like encoding/json does,
// since the header values come from the clients.
func appendJSONString(b []byte, s string) []byte {
	b = append(b, '"')
	start := 0
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			if c >= 0x20 && c != '"' && c != '\\' && c != '<' && c != '>' && c != '&' {
				i++
				continue
			}
			b = append(b, s[start:i]...)
			switch c {
			case '"', '\\':
				b = append(b, '\\', c)
			case '\n':
				b = append(b, '\\', 'n')
			case '\r':
				b = append(b, '\\', 'r')
			case '\t':
				b = append(b, '\\', 't')
			default:
				b = append(b, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
			}
			i++
			start = i
			continue
		}

		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			b = append(b, s[start:i]...)
			b = append(b, `�`...)
			i++
			start = i
			continue
		}
		// U+2028 and U+2029 break JavaScript parsers reading JSON, as encoding/json notes.
		if r == ' ' || r == ' ' {
			b = append(b, s[start:i]...)
			b = append(b, '\\', 'u', '2', '0', '2', hexDigits[r&0xf])
			i += size
			start = i
			continue
		}
		i += size
	}
	b = append(b, s[start:]...)
	return append(b, '"')
}
