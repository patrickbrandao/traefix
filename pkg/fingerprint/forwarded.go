package fingerprint

import (
	"context"
	"net/http"
	"time"
)

// Forwarded are the connection identifier and the fingerprints of a request, as the entry point that received it
// measured them, for an instance that serves the request without having received it: the client connected to another
// instance, which forwarded the request through the mesh.
type Forwarded struct {
	ConnID       string
	Fingerprints Fingerprints
}

// ForwardedFromRequest returns what the entry point measured for req, and false when its entry point has no fingerprint option.
func ForwardedFromRequest(req *http.Request) (Forwarded, bool) {
	st := requestStateFrom(req)
	if st == nil {
		return Forwarded{}, false
	}

	if st.forwarded != nil {
		return *st.forwarded, true
	}

	fingerprints, _ := FingerprintsFromRequest(req)

	forwarded := Forwarded{Fingerprints: fingerprints}
	if st.conn != nil {
		forwarded.ConnID = st.conn.id
	}

	return forwarded, true
}

// ContextWithForwarded returns a context carrying what another instance measured for a request,
// for the middlewares, the access log and the traces of the instance that serves it.
func ContextWithForwarded(ctx context.Context, f Forwarded) context.Context {
	return context.WithValue(ctx, requestStateKey{}, &requestState{start: time.Now(), forwarded: &f})
}

// EnableForwarded makes the routers record the fingerprints of the requests in the access log and in the traces,
// on an instance where no entry point has the fingerprint option but which serves requests that another instance received.
func EnableForwarded() {
	enabled.Store(true)
}
