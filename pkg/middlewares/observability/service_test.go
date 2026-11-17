package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/traefik/traefik/v3/pkg/observability/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TestNewService_serviceNameInContext(t *testing.T) {
	testCases := []struct {
		desc           string
		tracingEnabled bool
		expected       string
	}{
		{desc: "tracing enabled", tracingEnabled: true, expected: "myService"},
		{desc: "tracing disabled", tracingEnabled: false},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			tracer := tracing.NewTracer(&mockTracer{}, nil, nil, nil)
			ctx, span := tracer.Start(t.Context(), "entry", trace.WithSpanKind(trace.SpanKindServer))
			defer span.End()

			ctx = WithObservability(ctx, Observability{TracingEnabled: test.tracingEnabled})
			req := httptest.NewRequest(http.MethodGet, "http://www.test.com/", nil).WithContext(ctx)

			var got string
			next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got, _ = ServiceNameFromContext(r.Context())
			})

			NewService(t.Context(), "myService", next).ServeHTTP(httptest.NewRecorder(), req)

			assert.Equal(t, test.expected, got)
		})
	}
}

func TestNewService(t *testing.T) {
	type expected struct {
		attributes []attribute.KeyValue
		name       string
	}

	testCases := []struct {
		desc     string
		service  string
		expected []expected
	}{
		{
			desc:    "base",
			service: "myService",
			expected: []expected{
				{
					name: "GET",
					attributes: []attribute.KeyValue{
						attribute.String("span.kind", "server"),
					},
				},
				{
					name: "Service",
					attributes: []attribute.KeyValue{
						attribute.String("span.kind", "internal"),
						attribute.String("http.request.method", "GET"),
						attribute.Int64("http.response.status_code", int64(404)),
						attribute.String("network.protocol.version", "1.1"),
						attribute.String("server.address", "www.test.com"),
						attribute.Int64("server.port", int64(80)),
						attribute.String("url.full", "http://www.test.com/traces?p=OpenTelemetry"),
						attribute.String("url.scheme", "http"),
						attribute.String("traefik.service.name", "myService"),
						attribute.String("user_agent.original", "service-test"),
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://www.test.com/traces?p=OpenTelemetry", nil)
			req.RemoteAddr = "10.0.0.1:1234"
			req.Header.Set("User-Agent", "service-test")

			tracer := &mockTracer{}
			tracingCtx, entryPointSpan := tracer.Start(req.Context(), http.MethodGet, trace.WithSpanKind(trace.SpanKindServer))
			defer entryPointSpan.End()

			req = req.WithContext(tracingCtx)

			rw := httptest.NewRecorder()
			next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				rw.WriteHeader(http.StatusNotFound)
			})

			handler := NewService(t.Context(), test.service, next)
			handler.ServeHTTP(rw, req)

			for i, span := range tracer.spans {
				assert.Equal(t, test.expected[i].name, span.name)
				assert.Equal(t, test.expected[i].attributes, span.attributes)
			}
		})
	}
}
