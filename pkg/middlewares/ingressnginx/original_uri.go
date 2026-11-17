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

type originalPathContextKey struct{}

// WithOriginalPath stores the request path before an ingress-nginx rewrite.
// It preserves the first value when more than one rewrite middleware is used.
func WithOriginalPath(req *http.Request, path string) *http.Request {
	if _, ok := req.Context().Value(originalPathContextKey{}).(string); ok {
		return req
	}

	return req.WithContext(context.WithValue(req.Context(), originalPathContextKey{}, path))
}

// OriginalPath returns the path before an ingress-nginx rewrite, if one ran.
func OriginalPath(req *http.Request) string {
	if path, ok := req.Context().Value(originalPathContextKey{}).(string); ok {
		return path
	}

	return req.URL.Path
}
