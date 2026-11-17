package server

import (
	"context"
	"fmt"

	"github.com/traefik/traefik/v3/pkg/config/static"
	"github.com/traefik/traefik/v3/pkg/fingerprint"
	"github.com/traefik/traefik/v3/pkg/observability/metrics"
)

// The options below carry the Traefix additions to the entry point constructors,
// so that the upstream signatures and their callers stay unchanged.

type tcpEntryPointOptions struct {
	metricsRegistry metrics.Registry
}

type tcpEntryPointOption func(*tcpEntryPointOptions)

func withMetricsRegistry(registry metrics.Registry) tcpEntryPointOption {
	return func(o *tcpEntryPointOptions) {
		o.metricsRegistry = registry
	}
}

type httpServerOptions struct {
	fingerprint *fingerprint.EntryPoint
}

type httpServerOption func(*httpServerOptions)

func withFingerprint(fp *fingerprint.EntryPoint) httpServerOption {
	return func(o *httpServerOptions) {
		o.fingerprint = fp
	}
}

func newFingerprint(ctx context.Context, name string, config *static.EntryPoint, opts []tcpEntryPointOption) (*fingerprint.EntryPoint, error) {
	if config.Fingerprint == nil {
		return nil, nil
	}

	var options tcpEntryPointOptions
	for _, opt := range opts {
		opt(&options)
	}

	fp, err := fingerprint.NewEntryPoint(ctx, name, config.Fingerprint, config.HTTP2, options.metricsRegistry)
	if err != nil {
		return nil, fmt.Errorf("creating fingerprint: %w", err)
	}

	return fp, nil
}

// closeFingerprint stops the export of an entry point that failed to start.
func closeFingerprint(fp *fingerprint.EntryPoint) {
	if fp != nil {
		fp.Close(context.Background())
	}
}
