package ingressnginx

import (
	"context"
	"net/http"
)

type originalURIContextKey struct{}

// WithOriginalURI stores the client URI for use by ingress-nginx middleware
// that needs it after the request path has been rewritten.
func WithOriginalURI(req *http.Request, uri string) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), originalURIContextKey{}, uri))
}

// OriginalURI returns the client URI saved before an ingress-nginx rewrite.
func OriginalURI(req *http.Request) string {
	uri, _ := req.Context().Value(originalURIContextKey{}).(string)
	return uri
}

// OriginalRequestURI returns the client URI saved before an ingress-nginx rewrite,
// or the current request URI when no rewrite happened.
func OriginalRequestURI(req *http.Request) string {
	if uri := OriginalURI(req); uri != "" {
		return uri
	}

	return req.URL.RequestURI()
}
