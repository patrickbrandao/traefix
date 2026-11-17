package accesslog

import (
	"net"
	"net/http"
	"strings"

	"github.com/traefik/traefik/v3/pkg/ip"
	otypes "github.com/traefik/traefik/v3/pkg/observability/types"
)

// maskClientAddr truncates the host part of a host:port ClientAddr value, keeping the port.
func maskClientAddr(value string, a *otypes.AccessLogAnonymization) string {
	host, port := silentSplitHostPort(value)
	masked := ip.MaskIP(host, a.IPv4Subnet, a.IPv6Subnet)
	if port == "-" {
		return masked
	}
	return net.JoinHostPort(masked, port)
}

// maskIPList truncates each address of a comma-separated list, such as an X-Forwarded-For value.
func maskIPList(value string, a *otypes.AccessLogAnonymization) string {
	parts := strings.Split(value, ",")
	for i, part := range parts {
		trimmed := strings.TrimSpace(part)
		masked := ip.MaskIP(trimmed, a.IPv4Subnet, a.IPv6Subnet)
		if masked == trimmed {
			// Not a bare IP, it may carry a port.
			masked = maskClientAddr(trimmed, a)
		}
		parts[i] = strings.Replace(part, trimmed, masked, 1)
	}
	return strings.Join(parts, ",")
}

// maskForwarded truncates the addresses of the for and by parameters of a Forwarded header value (RFC 7239).
func maskForwarded(value string, a *otypes.AccessLogAnonymization) string {
	elements := strings.Split(value, ",")
	for i, element := range elements {
		pairs := strings.Split(element, ";")
		for j, pair := range pairs {
			key, val, found := strings.Cut(pair, "=")
			if !found {
				continue
			}

			name := strings.ToLower(strings.TrimSpace(key))
			if name != "for" && name != "by" {
				continue
			}

			node := strings.Trim(strings.TrimSpace(val), `"`)
			masked := maskClientAddr(node, a)
			if strings.HasPrefix(masked, "[") || strings.Contains(masked, ":") {
				masked = `"` + masked + `"`
			}
			pairs[j] = key + "=" + masked
		}
		elements[i] = strings.Join(pairs, ";")
	}
	return strings.Join(elements, ",")
}

// anonymizedHeaderValue returns the header value with the client addresses truncated
// for the headers that carry them, and the value unchanged for any other header.
func anonymizedHeaderValue(name, value string, a *otypes.AccessLogAnonymization) string {
	switch http.CanonicalHeaderKey(name) {
	case "X-Forwarded-For", "X-Real-Ip", "X-Original-Forwarded-For", "True-Client-Ip", "Cf-Connecting-Ip":
		return maskIPList(value, a)
	case "Forwarded":
		return maskForwarded(value, a)
	default:
		return value
	}
}
