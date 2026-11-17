package fingerprint

import (
	"strings"
)

// maxFieldValue bounds the exported value of a header.
const maxFieldValue = 512

// sensitiveFields carry credentials: they are only exported with allowSensitiveFields,
// so that a wildcard in the fields list never exports them by accident.
var sensitiveFields = []string{"cookie", "authorization", "proxy-authorization", "set-cookie"}

// Field is a request header kept with its value.
type Field struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type fieldMatcher struct {
	exact          []string
	prefixes       []string
	allowSensitive bool
}

func newFieldMatcher(patterns []string, allowSensitive bool) *fieldMatcher {
	m := &fieldMatcher{allowSensitive: allowSensitive}
	for _, p := range patterns {
		if prefix, ok := strings.CutSuffix(p, "*"); ok {
			m.prefixes = append(m.prefixes, prefix)
			continue
		}
		m.exact = append(m.exact, p)
	}

	return m
}

// sensitive returns the sensitive fields the patterns would export.
func (m *fieldMatcher) sensitive() []string {
	var names []string
	for _, name := range sensitiveFields {
		if m.matchPattern(name) {
			names = append(names, name)
		}
	}
	return names
}

func (m *fieldMatcher) match(name string) bool {
	if m == nil || !m.matchPattern(name) {
		return false
	}

	if !m.allowSensitive {
		for _, s := range sensitiveFields {
			if strings.EqualFold(name, s) {
				return false
			}
		}
	}

	return true
}

func (m *fieldMatcher) matchPattern(name string) bool {
	for _, e := range m.exact {
		if strings.EqualFold(name, e) {
			return true
		}
	}
	for _, p := range m.prefixes {
		if len(name) >= len(p) && strings.EqualFold(name[:len(p)], p) {
			return true
		}
	}

	return false
}

func truncateValue(v string) string {
	if len(v) > maxFieldValue {
		return v[:maxFieldValue]
	}
	return v
}
