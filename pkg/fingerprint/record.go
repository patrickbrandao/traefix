package fingerprint

import (
	"strings"
	"time"
)

// Schema identifies the format of the exported batches, documented with the entry point fingerprint option.
const Schema = "traefix.fingerprint/v1"

const (
	recordConnection = "connection"
	recordRequest    = "request"

	// maxRecordPath bounds the exported request path.
	maxRecordPath = 256
)

type addrRecord struct {
	IP   string `json:"ip"`
	Port uint16 `json:"port"`
}

type tlsRecord struct {
	SNI                  string   `json:"sni,omitempty"`
	ALPNOffered          []string `json:"alpnOffered,omitempty"`
	ALPN                 string   `json:"alpn,omitempty"`
	Version              string   `json:"version,omitempty"`
	Cipher               string   `json:"cipher,omitempty"`
	JA3                  string   `json:"ja3,omitempty"`
	JA3String            string   `json:"ja3String,omitempty"`
	JA4                  string   `json:"ja4,omitempty"`
	ClientHello          []byte   `json:"clientHello,omitempty"`
	ClientHelloTruncated bool     `json:"clientHelloTruncated,omitempty"`
}

type http2Record struct {
	Akamai          string `json:"akamai,omitempty"`
	Prefix          []byte `json:"prefix,omitempty"`
	PrefixTruncated bool   `json:"prefixTruncated,omitempty"`
}

type connectionRecord struct {
	Type       string       `json:"type"`
	ConnID     string       `json:"connId"`
	Time       time.Time    `json:"time"`
	EntryPoint string       `json:"entryPoint"`
	Src        addrRecord   `json:"src"`
	Dst        addrRecord   `json:"dst"`
	Proto      string       `json:"proto,omitempty"`
	TLS        *tlsRecord   `json:"tls,omitempty"`
	HTTP2      *http2Record `json:"http2,omitempty"`
}

func (r *connectionRecord) size() int {
	n := 256
	if r.TLS != nil {
		n += 256 + len(r.TLS.JA3String) + 4*len(r.TLS.ClientHello)/3
	}
	if r.HTTP2 != nil {
		n += 64 + 4*len(r.HTTP2.Prefix)/3
	}
	return n
}

type requestRecord struct {
	Type             string    `json:"type"`
	ConnID           string    `json:"connId,omitempty"`
	Start            time.Time `json:"start"`
	End              time.Time `json:"end"`
	EntryPoint       string    `json:"entryPoint"`
	Proto            string    `json:"proto"`
	Method           string    `json:"method"`
	Authority        string    `json:"authority,omitempty"`
	Path             string    `json:"path"`
	HTTPVersion      string    `json:"httpVersion"`
	FieldNames       []string  `json:"fieldNames,omitempty"`
	Fields           []Field   `json:"fields,omitempty"`
	CookieNames      []string  `json:"cookieNames,omitempty"`
	JA4H             string    `json:"ja4h,omitempty"`
	HeadersTruncated bool      `json:"headersTruncated,omitempty"`
	Status           int       `json:"status"`
	Router           string    `json:"router,omitempty"`
	Service          string    `json:"service,omitempty"`
}

func (r *requestRecord) size() int {
	n := 320 + len(r.Authority) + len(r.Path) + len(r.Router) + len(r.Service)
	for _, name := range r.FieldNames {
		n += len(name) + 3
	}
	for _, f := range r.Fields {
		n += len(f.Name) + len(f.Value) + 24
	}
	for _, name := range r.CookieNames {
		n += len(name) + 3
	}
	return n
}

func recordPath(path string) string {
	if len(path) > maxRecordPath {
		return path[:maxRecordPath]
	}
	return path
}

// cookieNames returns the names of the cookies in the received order, without their values.
func cookieNames(cookies []string) []string {
	var names []string
	for _, cookie := range cookies {
		for crumb := range strings.SplitSeq(cookie, ";") {
			name, _, _ := strings.Cut(strings.TrimSpace(crumb), "=")
			if name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}
