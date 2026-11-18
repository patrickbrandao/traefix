package coraza

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/corazawaf/coraza/v3/types"
)

// responseWriter runs the phases 3 and 4 on the response. While the WAF inspects the response body, it holds back
// the status code and the body, and only sends them once the WAF let them through: a flush before that would commit
// the status code.
type responseWriter struct {
	w     http.ResponseWriter
	tx    types.Transaction
	proto string
	meter *meter

	status        int
	wroteHeader   bool
	headerFlushed bool
	bodyReleased  bool
	allowFlush    bool
	hijacked      bool
}

func newResponseWriter(w http.ResponseWriter, tx types.Transaction, proto string, m *meter) *responseWriter {
	return &responseWriter{w: w, tx: tx, proto: proto, meter: m, status: http.StatusOK}
}

func (r *responseWriter) Header() http.Header {
	return r.w.Header()
}

func (r *responseWriter) Unwrap() http.ResponseWriter {
	return r.w
}

func (r *responseWriter) WriteHeader(status int) {
	// The informational responses go through, and the final response follows.
	if status >= http.StatusContinue && status < http.StatusOK && status != http.StatusSwitchingProtocols {
		r.w.WriteHeader(status)
		return
	}
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true
	r.status = status

	start := time.Now()
	for name, values := range r.w.Header() {
		for _, v := range values {
			r.tx.AddResponseHeader(name, v)
		}
	}
	it := r.tx.ProcessResponseHeaders(status, r.proto)
	r.meter.add(start)

	if it != nil {
		r.deny(it)
		return
	}

	// A switch of protocol has no body, and the connection is taken over right after.
	if status == http.StatusSwitchingProtocols {
		r.flushHeader()
	}
	if !r.inspectsBody() {
		r.allowFlush = true
	}
}

func (r *responseWriter) Write(b []byte) (int, error) {
	if r.tx.IsInterrupted() {
		// The interruption already answered the client: the next handlers write into the void.
		return len(b), nil
	}
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
		if r.tx.IsInterrupted() {
			return len(b), nil
		}
	}

	if !r.inspectsBody() || r.bodyReleased {
		r.flushHeader()
		return r.w.Write(b)
	}

	start := time.Now()
	it, n, err := r.tx.WriteResponseBody(b)
	r.meter.add(start)

	if it != nil {
		r.deny(it)
		return len(b), nil
	}
	if err != nil || n == len(b) {
		return n, err
	}

	// The body outgrew the WAF limit without an interruption: the WAF processes what it kept,
	// and the rest goes straight to the client, as SecResponseBodyLimitAction ProcessPartial does.
	if err := r.releaseBody(); err != nil {
		return n, err
	}
	n2, err := r.w.Write(b[n:])
	return n + n2, err
}

// FlushError only flushes once the WAF let the response through, as a flush commits the status code.
func (r *responseWriter) FlushError() error {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	if !r.allowFlush || !r.headerFlushed {
		return nil
	}
	return http.NewResponseController(r.w).Flush()
}

// Hijack hands over the connection, e.g. for a WebSocket, after which the response is not inspected anymore.
func (r *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, brw, err := http.NewResponseController(r.w).Hijack()
	if err == nil {
		r.hijacked = true
	}
	return conn, brw, err
}

func (r *responseWriter) inspectsBody() bool {
	return r.tx.IsResponseBodyAccessible() && r.tx.IsResponseBodyProcessable()
}

func (r *responseWriter) deny(it *types.Interruption) {
	for name := range r.w.Header() {
		r.w.Header().Del(name)
	}
	r.w.Header().Set("Content-Length", "0")
	r.status = statusFromInterruption(it, r.status)
	r.flushHeader()
}

func (r *responseWriter) flushHeader() {
	if !r.headerFlushed {
		r.headerFlushed = true
		r.w.WriteHeader(r.status)
	}
}

func (r *responseWriter) releaseBody() error {
	if r.bodyReleased {
		return nil
	}
	r.bodyReleased = true
	r.allowFlush = true

	reader, err := r.tx.ResponseBodyReader()
	if err != nil {
		r.status = http.StatusInternalServerError
		r.flushHeader()
		return fmt.Errorf("reading the buffered response body: %w", err)
	}

	r.flushHeader()
	if _, err := io.Copy(r.w, reader); err != nil {
		return fmt.Errorf("writing the response body: %w", err)
	}
	return nil
}

// finish runs the phase 4 on the buffered response body, then sends the response if the WAF let it through.
func (r *responseWriter) finish() error {
	if r.hijacked {
		return nil
	}
	// A handler writing nothing still answers 200, whose headers the WAF inspects.
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	if r.tx.IsInterrupted() {
		return nil
	}

	if !r.inspectsBody() || r.bodyReleased {
		r.allowFlush = true
		r.flushHeader()
		return nil
	}

	start := time.Now()
	it, err := r.tx.ProcessResponseBody()
	r.meter.add(start)

	switch {
	case err != nil:
		r.status = http.StatusInternalServerError
		r.flushHeader()
		return fmt.Errorf("processing the response body: %w", err)
	case it != nil:
		r.deny(it)
		return nil
	}

	return r.releaseBody()
}
