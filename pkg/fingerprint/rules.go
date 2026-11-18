package fingerprint

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	httpmuxer "github.com/traefik/traefik/v3/pkg/muxer/http"
)

// entryPoints holds the names of the entry points with the fingerprint option.
var entryPoints sync.Map

// EntryPointEnabled reports whether the entry point has the fingerprint option.
func EntryPointEnabled(name string) bool {
	_, ok := entryPoints.Load(name)
	return ok
}

const (
	matcherJA3  = "JA3"
	matcherJA4  = "JA4"
	matcherJA4H = "JA4H"
)

var ruleMatchers = []string{matcherJA3, matcherJA4, matcherJA4H}

func init() {
	httpmuxer.RegisterMatcher(matcherJA3, fingerprintMatcher(matcherJA3, func(st *requestState) string {
		if st.conn == nil {
			return ""
		}
		return st.conn.ja3
	}))
	httpmuxer.RegisterMatcher(matcherJA4, fingerprintMatcher(matcherJA4, func(st *requestState) string {
		if st.conn == nil {
			return ""
		}
		return st.conn.ja4
	}))
	httpmuxer.RegisterMatcher(matcherJA4H, fingerprintMatcher(matcherJA4H, func(st *requestState) string {
		if st.wire == nil {
			return ""
		}
		return st.wire.JA4H()
	}))
}

// fingerprintMatcher matches a fingerprint against a value, or against a prefix given with a trailing *.
// A request without the fingerprint, e.g. a request over plain HTTP for JA4, matches no value.
func fingerprintMatcher(name string, value func(*requestState) string) func(...string) (httpmuxer.MatcherFunc, error) {
	return func(params ...string) (httpmuxer.MatcherFunc, error) {
		if len(params) != 1 || params[0] == "" || params[0] == "*" {
			return nil, fmt.Errorf("the %s matcher expects one non-empty fingerprint", name)
		}

		expected, prefix := strings.CutSuffix(params[0], "*")

		return func(req *http.Request) bool {
			st := requestStateFrom(req)
			if st == nil {
				return false
			}

			v := value(st)
			if v == "" {
				return false
			}
			if prefix {
				return strings.HasPrefix(v, expected)
			}
			return v == expected
		}, nil
	}
}

// CheckRule refuses a rule using a fingerprint matcher on an entry point without the fingerprint option,
// where the matcher would never match.
func CheckRule(entryPoint, rule string) error {
	if EntryPointEnabled(entryPoint) {
		return nil
	}

	if name := usedMatcher(rule); name != "" {
		return fmt.Errorf("the %s matcher requires the fingerprint option on the entry point %s", name, entryPoint)
	}

	return nil
}

// usedMatcher returns the first fingerprint matcher called in the rule, skipping the quoted values,
// so that a value containing a matcher name does not count.
func usedMatcher(rule string) string {
	var quote byte
	start := -1

	for i := range len(rule) {
		c := rule[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}

		switch {
		case c == '`' || c == '"':
			quote = c
			start = -1
		case isIdentByte(c):
			if start < 0 {
				start = i
			}
		default:
			if c == '(' && start >= 0 {
				for _, name := range ruleMatchers {
					if strings.TrimSpace(rule[start:i]) == name {
						return name
					}
				}
			}
			if c != ' ' || start < 0 {
				start = -1
			}
		}
	}

	return ""
}

func isIdentByte(c byte) bool {
	return c == '_' || '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}
