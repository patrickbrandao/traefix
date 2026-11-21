package fingerprint

import (
	"net/http"

	"github.com/traefik/traefik/v3/pkg/middlewares/accesslog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// The access log fields of the fingerprints.
const (
	AccessLogConnID = "FingerprintConnId"
	AccessLogJA3    = "FingerprintJA3"
	AccessLogJA4    = "FingerprintJA4"
	AccessLogH2     = "FingerprintH2"
	AccessLogJA4H   = "FingerprintJA4H"
	AccessLogJA4T   = "FingerprintJA4T"
)

// The trace attributes of the fingerprints. tls.client.ja3 is the OpenTelemetry semantic convention,
// which has no name for the others.
var (
	traceConnID = attribute.Key("traefix.connection.id")
	traceJA3    = attribute.Key("tls.client.ja3")
	traceJA4    = attribute.Key("traefix.tls.client.ja4")
	traceH2     = attribute.Key("traefix.http2.akamai")
	traceJA4H   = attribute.Key("traefix.http.request.ja4h")
	traceJA4T   = attribute.Key("traefix.tcp.client.ja4t")
)

// observe adds the fingerprints to the access log and to the span of the router serving the request.
// It runs in the router chain, where both the access log data and the router span exist.
func (st *requestState) observe(req *http.Request) {
	var connID, ja3, ja4, h2, ja4h, ja4t string

	switch {
	case st.forwarded != nil:
		f := st.forwarded.Fingerprints
		connID, ja3, ja4, h2, ja4h, ja4t = st.forwarded.ConnID, f.JA3, f.JA4, f.HTTP2, f.JA4H, f.JA4T

	default:
		if c := st.conn; c != nil {
			connID, ja3, ja4, ja4t = c.id, c.ja3, c.ja4, c.ja4t
			if req.ProtoMajor == 2 {
				h2 = c.akamai
			}
		}
		if st.wire != nil {
			ja4h = st.wire.JA4H()
		}
	}

	if data := accesslog.GetLogData(req); data != nil {
		setField(data.Core, AccessLogConnID, connID)
		setField(data.Core, AccessLogJA3, ja3)
		setField(data.Core, AccessLogJA4, ja4)
		setField(data.Core, AccessLogH2, h2)
		setField(data.Core, AccessLogJA4H, ja4h)
		setField(data.Core, AccessLogJA4T, ja4t)
	}

	if span := trace.SpanFromContext(req.Context()); span.IsRecording() {
		attrs := make([]attribute.KeyValue, 0, 6)
		attrs = appendAttr(attrs, traceConnID, connID)
		attrs = appendAttr(attrs, traceJA3, ja3)
		attrs = appendAttr(attrs, traceJA4, ja4)
		attrs = appendAttr(attrs, traceH2, h2)
		attrs = appendAttr(attrs, traceJA4H, ja4h)
		attrs = appendAttr(attrs, traceJA4T, ja4t)
		span.SetAttributes(attrs...)
	}
}

func setField(core accesslog.CoreLogData, name, value string) {
	if value != "" {
		core[name] = value
	}
}

func appendAttr(attrs []attribute.KeyValue, key attribute.Key, value string) []attribute.KeyValue {
	if value == "" {
		return attrs
	}
	return append(attrs, key.String(value))
}
