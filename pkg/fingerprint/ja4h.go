package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The JA4H rules below follow the FoxIO reference implementation (github.com/FoxIO-LLC/ja4, rust/ja4/src/http.rs),
// including its handling of cookies, so that the fingerprints computed here equal the ones it computes.

var ja4hMethods = map[string]string{
	"CONNECT": "co",
	"DELETE":  "de",
	"GET":     "ge",
	"HEAD":    "he",
	"OPTIONS": "op",
	"PATCH":   "pa",
	"POST":    "po",
	"PUT":     "pu",
	"TRACE":   "tr",
}

// JA4H returns the JA4H fingerprint of the request.
func (r *Request) JA4H() string {
	if r.ja4h == "" {
		r.ja4h = r.computeJA4H()
	}

	return r.ja4h
}

func (r *Request) computeJA4H() string {
	// The fields are joined in a stack buffer, which avoids an allocation for the usual header blocks.
	var scratch [1024]byte

	out := make([]byte, 0, 51)
	out = append(out, ja4hMethod(r.Method)...)

	if r.ProtoMajor == 2 {
		out = append(out, "20"...)
	} else {
		out = strconv.AppendInt(out, int64(r.ProtoMajor), 10)
		out = strconv.AppendInt(out, int64(r.ProtoMinor), 10)
	}

	var hasCookie, hasReferer bool
	count := 0
	names := scratch[:0]
	for _, name := range r.Names {
		switch {
		case strings.EqualFold(name, "cookie"):
			hasCookie = true
		case strings.EqualFold(name, "referer"):
			hasReferer = true
		default:
			if count > 0 {
				names = append(names, ',')
			}
			names = append(names, name...)
			count++
		}
	}

	out = append(out, marker(hasCookie, 'c'), marker(hasReferer, 'r'))

	count = min(count, 99)
	out = append(out, byte('0'+count/10), byte('0'+count%10))
	out = append(out, ja4hLanguage(r.AcceptLanguage)...)

	out = append(out, '_')
	out = appendHash12(out, names)

	cookieNames, cookiePairs := r.ja4hCookies(names[len(names):])
	out = append(out, '_')
	out = appendHash12(out, cookieNames)
	out = append(out, '_')
	out = appendHash12(out, cookiePairs)

	return string(out)
}

func ja4hMethod(method string) string {
	if code, ok := ja4hMethods[method]; ok {
		return code
	}

	// The reference implementation yields no fingerprint for an unknown method,
	// while its Python counterpart uses the first two lowercased characters, which keeps the fingerprint usable.
	method = strings.ToLower(method)
	if len(method) > 2 {
		method = method[:2]
	}

	return method
}

func marker(set bool, c byte) byte {
	if set {
		return c
	}

	return 'n'
}

func ja4hLanguage(value string) string {
	primary, _, _ := strings.Cut(strings.TrimLeft(value, " \t"), ",")
	primary = strings.ToLower(strings.ReplaceAll(primary, "-", ""))

	n := utf8.RuneCountInString(primary)
	switch {
	case n > 4:
		i := 0
		for range 4 {
			_, size := utf8.DecodeRuneInString(primary[i:])
			i += size
		}
		return primary[:i]
	case n < 4:
		return primary + strings.Repeat("0", 4-n)
	default:
		return primary
	}
}

type cookiePair struct {
	name     string
	value    string
	hasValue bool
}

// ja4hCookies returns the sorted cookie names and pairs, joined by commas, appended to buf.
// An HTTP/1 request contributes the crumbs of its first Cookie field, split on "; ",
// and an HTTP/2 request contributes each cookie field as a single crumb, as the reference implementation does.
func (r *Request) ja4hCookies(buf []byte) ([]byte, []byte) {
	if len(r.Cookies) == 0 {
		return nil, nil
	}

	var pairs []cookiePair
	addCrumb := func(crumb string) {
		name, value, ok := strings.Cut(crumb, "=")
		pairs = append(pairs, cookiePair{name: name, value: value, hasValue: ok})
	}

	if r.ProtoMajor >= 2 {
		pairs = make([]cookiePair, 0, len(r.Cookies))
		for _, crumb := range r.Cookies {
			addCrumb(crumb)
		}
	} else {
		cookie := r.Cookies[0]
		pairs = make([]cookiePair, 0, strings.Count(cookie, "; ")+1)
		for {
			crumb, rest, found := strings.Cut(cookie, "; ")
			addCrumb(crumb)
			if !found {
				break
			}
			cookie = rest
		}
	}

	slices.SortFunc(pairs, func(a, b cookiePair) int {
		if c := strings.Compare(a.name, b.name); c != 0 {
			return c
		}
		if a.hasValue != b.hasValue {
			if a.hasValue {
				return 1
			}
			return -1
		}
		return strings.Compare(a.value, b.value)
	})

	names := buf
	for i, p := range pairs {
		if i > 0 {
			names = append(names, ',')
		}
		names = append(names, p.name...)
	}

	joined := names[len(names):]
	for i, p := range pairs {
		if i > 0 {
			joined = append(joined, ',')
		}
		joined = append(joined, p.name...)
		if p.hasValue {
			joined = append(joined, '=')
			joined = append(joined, p.value...)
		}
	}

	return names, joined
}

func appendHash12(out, s []byte) []byte {
	if len(s) == 0 {
		return append(out, "000000000000"...)
	}

	sum := sha256.Sum256(s)

	return hex.AppendEncode(out, sum[:6])
}
