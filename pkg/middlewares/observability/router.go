package observability

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/containous/alice"
	"github.com/traefik/traefik/v3/pkg/middlewares"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	"github.com/traefik/traefik/v3/pkg/observability/tracing"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	routerTypeName = "TracingRouter"
)

type routerTracing struct {
	router     string
	routerRule string
	route      string
	service    string
	next       http.Handler
}

// WrapRouterHandler Wraps tracing to alice.Constructor.
func WrapRouterHandler(ctx context.Context, router, routerRule, service string) alice.Constructor {
	return func(next http.Handler) (http.Handler, error) {
		return newRouter(ctx, router, routerRule, service, next), nil
	}
}

// newRouter creates a new tracing middleware that traces the internal requests.
func newRouter(ctx context.Context, router, routerRule, service string, next http.Handler) http.Handler {
	middlewares.GetLogger(ctx, "tracing", routerTypeName).
		Debug().Str(logs.RouterName, router).Str(logs.ServiceName, service).Msg("Added outgoing tracing middleware")

	return &routerTracing{
		router:     router,
		routerRule: routerRule,
		route:      extractRouteFromRule(routerRule),
		service:    service,
		next:       next,
	}
}

func (f *routerTracing) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	tracer := tracing.TracerFromContext(req.Context())
	if tracer == nil {
		f.next.ServeHTTP(rw, req)
		return
	}

	// Follow semantic conventions defined by OTEL: https://opentelemetry.io/docs/specs/semconv/http/http-spans/#name
	// The entrypoint names the server span after the method only, as the route is unknown there.
	// The server span is looked up by key because, with nested routers, the active span may be a parent Router span.
	if serverSpan := serverSpanFromContext(req.Context()); f.route != "" && serverSpan != nil && serverSpan.IsRecording() {
		serverSpan.SetAttributes(semconv.HTTPRoute(f.route))
		serverSpan.SetName(req.Method + " " + f.route)
	}

	if DetailedTracingEnabled(req.Context()) {
		tracingCtx, span := tracer.Start(req.Context(), "Router", trace.WithSpanKind(trace.SpanKindInternal))
		defer span.End()

		req = req.WithContext(tracingCtx)

		span.SetAttributes(attribute.String("traefik.service.name", f.service))
		span.SetAttributes(attribute.String("traefik.router.name", f.router))
		span.SetAttributes(semconv.HTTPRoute(f.routerRule))
	}

	f.next.ServeHTTP(rw, req)
}

// pathMatcher captures the argument of a Path or PathPrefix matcher, quoted with backticks or double quotes.
var pathMatcher = regexp.MustCompile("\\b(?:Path|PathPrefix)\\(\\s*(?:`([^`]*)`|\"([^\"\\\\]*)\")\\s*\\)")

// extractRouteFromRule returns the path of the single Path or PathPrefix matcher of a rule,
// or an empty string when the rule has no such unambiguous path.
// A rule with an OR or a negation can match requests outside of the matcher path,
// so naming the span after it would be misleading.
func extractRouteFromRule(rule string) string {
	if strings.Contains(rule, "||") || strings.Contains(rule, "!") {
		return ""
	}

	matches := pathMatcher.FindAllStringSubmatch(rule, -1)
	if len(matches) != 1 {
		return ""
	}

	if matches[0][1] != "" {
		return matches[0][1]
	}
	return matches[0][2]
}
