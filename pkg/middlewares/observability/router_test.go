package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/traefik/traefik/v3/pkg/observability/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TestNewRouter(t *testing.T) {
	type expected struct {
		attributes []attribute.KeyValue
		name       string
	}

	testCases := []struct {
		desc       string
		service    string
		router     string
		routerRule string
		expected   []expected
	}{
		{
			desc:       "Path rule",
			service:    "myService",
			router:     "myRouter",
			routerRule: "Path(`/`)",
			expected: []expected{
				{
					// Root server span: name updated from "GET" to "GET /"
					// and http.route attribute added by the router middleware.
					name: "GET /",
					attributes: []attribute.KeyValue{
						attribute.String("span.kind", "server"),
						attribute.String("http.route", "/"),
					},
				},
				{
					// Router internal span: keeps the raw rule as http.route
					// (the extracted path is only set on the root server span).
					name: "Router",
					attributes: []attribute.KeyValue{
						attribute.String("span.kind", "internal"),
						attribute.String("traefik.service.name", "myService"),
						attribute.String("traefik.router.name", "myRouter"),
						attribute.String("http.route", "Path(`/`)"),
					},
				},
			},
		},
		{
			desc:       "PathPrefix rule",
			service:    "myService",
			router:     "myRouter",
			routerRule: "PathPrefix(`/api/v1/ml-scribe`)",
			expected: []expected{
				{
					name: "GET /api/v1/ml-scribe",
					attributes: []attribute.KeyValue{
						attribute.String("span.kind", "server"),
						attribute.String("http.route", "/api/v1/ml-scribe"),
					},
				},
				{
					name: "Router",
					attributes: []attribute.KeyValue{
						attribute.String("span.kind", "internal"),
						attribute.String("traefik.service.name", "myService"),
						attribute.String("traefik.router.name", "myRouter"),
						attribute.String("http.route", "PathPrefix(`/api/v1/ml-scribe`)"),
					},
				},
			},
		},
		{
			desc:       "complex rule with Host and PathPrefix",
			service:    "myService",
			router:     "myRouter",
			routerRule: "Host(`example.com`) && PathPrefix(`/grafana`)",
			expected: []expected{
				{
					name: "GET /grafana",
					attributes: []attribute.KeyValue{
						attribute.String("span.kind", "server"),
						attribute.String("http.route", "/grafana"),
					},
				},
				{
					name: "Router",
					attributes: []attribute.KeyValue{
						attribute.String("span.kind", "internal"),
						attribute.String("traefik.service.name", "myService"),
						attribute.String("traefik.router.name", "myRouter"),
						attribute.String("http.route", "Host(`example.com`) && PathPrefix(`/grafana`)"),
					},
				},
			},
		},
		{
			desc:       "Host-only rule",
			service:    "myService",
			router:     "myRouter",
			routerRule: "Host(`example.com`)",
			expected: []expected{
				{
					name: "GET",
					attributes: []attribute.KeyValue{
						attribute.String("span.kind", "server"),
					},
				},
				{
					name: "Router",
					attributes: []attribute.KeyValue{
						attribute.String("span.kind", "internal"),
						attribute.String("traefik.service.name", "myService"),
						attribute.String("traefik.router.name", "myRouter"),
						attribute.String("http.route", "Host(`example.com`)"),
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://www.test.com/traces?p=OpenTelemetry", nil)
			req.RemoteAddr = "10.0.0.1:1234"
			req.Header.Set("User-Agent", "router-test")

			mockTracer := &mockTracer{}
			tracer := tracing.NewTracer(mockTracer, nil, nil, nil)
			tracingCtx, entryPointSpan := tracer.Start(req.Context(), http.MethodGet, trace.WithSpanKind(trace.SpanKindServer))
			defer entryPointSpan.End()

			tracingCtx = context.WithValue(tracingCtx, serverSpanKey{}, entryPointSpan)
			tracingCtx = WithObservability(tracingCtx, Observability{
				TracingEnabled:         true,
				DetailedTracingEnabled: true,
			})

			req = req.WithContext(tracingCtx)

			rw := httptest.NewRecorder()
			next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				rw.WriteHeader(http.StatusNotFound)
			})

			handler := newRouter(t.Context(), test.router, test.routerRule, test.service, next)
			handler.ServeHTTP(rw, req)

			for i, span := range mockTracer.spans {
				assert.Equal(t, test.expected[i].name, span.name)
				assert.Equal(t, test.expected[i].attributes, span.attributes)
			}
		})
	}
}

func TestExtractRouteFromRule(t *testing.T) {
	testCases := []struct {
		rule     string
		expected string
	}{
		{rule: "Path(`/healthz`)", expected: "/healthz"},
		{rule: "PathPrefix(`/api`)", expected: "/api"},
		{rule: `PathPrefix("/api")`, expected: "/api"},
		{rule: "Host(`example.com`) && PathPrefix(`/grafana`)", expected: "/grafana"},
		{rule: "Host(`example.com`)"},
		{rule: "PathRegexp(`^/api/.*`)"},
		{rule: "Path(`/a`) || Path(`/b`)"},
		{rule: "!PathPrefix(`/admin`)"},
		{rule: "PathPrefix(`/a`) && PathPrefix(`/a/b`)"},
		{rule: ""},
	}

	for _, test := range testCases {
		t.Run(test.rule, func(t *testing.T) {
			assert.Equal(t, test.expected, extractRouteFromRule(test.rule))
		})
	}
}
