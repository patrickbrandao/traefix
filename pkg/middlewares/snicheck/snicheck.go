package snicheck

import (
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/tcp"
)

// SNICheck is an HTTP handler that checks whether the TLS configuration for the server name is the same as for the host header.
type SNICheck struct {
	next           http.Handler
	routerName     string
	tlsOptionsName string
}

// New creates a new SNICheck.
func New(routerName, tlsOptionsName string, next http.Handler) *SNICheck {
	return &SNICheck{
		next:           next,
		routerName:     routerName,
		tlsOptionsName: tlsOptionsName,
	}
}

func (s SNICheck) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.TLS == nil {
		s.next.ServeHTTP(rw, req)
		return
	}

	tlsOptionsNameUsed := tcp.GetTLSOptionsName(req.Context())
	if s.tlsOptionsName != tlsOptionsNameUsed {
		log.Debug().
			Str("routerName", s.routerName).
			Str("req.Host", req.Host).
			Str("req.TLS.ServerName", req.TLS.ServerName).
			Msgf("TLS options difference: SNI:%s, Header:%s", tlsOptionsNameUsed, s.tlsOptionsName)

		// The TLS options name of the connection is resolved once, at handshake time:
		// after a configuration change it can stay stale, and every request on the connection would get a 421.
		// Closing the connection makes the client handshake again against the current configuration.
		// "Connection: close" closes HTTP/1.1 connections and sends a GOAWAY on HTTP/2,
		// but has no meaning on HTTP/3, where the QUIC connection is closed once the response is flushed.
		rw.Header().Set("Connection", "close")
		http.Error(rw, http.StatusText(http.StatusMisdirectedRequest), http.StatusMisdirectedRequest)
		_ = http.NewResponseController(rw).Flush()
		tcp.CloseConn(req.Context())
		return
	}

	s.next.ServeHTTP(rw, req)
}
