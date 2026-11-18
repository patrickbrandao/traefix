package fingerprint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/middlewares/accesslog"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestWrapRouterHandler_observability(t *testing.T) {
	enabled.Store(true)

	recorder := tracetest.NewSpanRecorder()
	tracer := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)).Tracer("test")

	handler, err := WrapRouterHandler("app@file", "app@file")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	require.NoError(t, err)

	f := Fingerprints{JA3: "ja3hash", JA4: "t13d", JA4H: "ge11", HTTP2: "1:65536|0|0|m,a,s,p"}

	ctx, span := tracer.Start(context.Background(), "router")
	ctx = ContextWithFingerprints(ctx, f)
	requestStateFrom(httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)).conn.id = "01ARYZ6S41TSV4RRFFQ69G5FAV"

	data := &accesslog.LogData{Core: accesslog.CoreLogData{}}
	ctx = context.WithValue(ctx, accesslog.DataTableKey, data)

	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	req.ProtoMajor = 2
	handler.ServeHTTP(httptest.NewRecorder(), req)
	span.End()

	assert.Equal(t, accesslog.CoreLogData{
		AccessLogConnID: "01ARYZ6S41TSV4RRFFQ69G5FAV",
		AccessLogJA3:    "ja3hash",
		AccessLogJA4:    "t13d",
		AccessLogH2:     "1:65536|0|0|m,a,s,p",
		AccessLogJA4H:   "ge11",
	}, data.Core)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	assert.ElementsMatch(t, []attribute.KeyValue{
		traceConnID.String("01ARYZ6S41TSV4RRFFQ69G5FAV"),
		traceJA3.String("ja3hash"),
		traceJA4.String("t13d"),
		traceH2.String("1:65536|0|0|m,a,s,p"),
		traceJA4H.String("ge11"),
	}, spans[0].Attributes())
}
