package acme

import (
	"sync/atomic"

	"github.com/traefik/traefik/v3/pkg/observability/metrics"
)

const (
	electionWon     = "won"
	electionLost    = "lost"
	electionSkipped = "skipped"

	issuanceObtain = "obtain"
	issuanceRenew  = "renew"

	lookupLocal   = "local"
	lookupRedis   = "redis"
	lookupMesh    = "mesh"
	lookupMiss    = "miss"
	lookupInvalid = "invalid"
	lookupError   = "error"

	resultSuccess = "success"
	resultFailure = "failure"

	metricLabelResolver = "resolver"
	metricLabelResult   = "result"
)

// The stores and the HTTP-01 challenge are created before the metrics registries, which reach them afterwards.
var metricsRegistry atomic.Pointer[metrics.Registry]

// SetMetricsRegistry sets the registry of the metrics of the certificate stores and of the cluster (Traefix).
func SetMetricsRegistry(registry metrics.Registry) {
	if registry != nil {
		metricsRegistry.Store(&registry)
	}
}

func countElection(resolverName, result string) {
	if registry := metricsRegistry.Load(); registry != nil {
		(*registry).ACMEClusterElectionsCounter().With(metricLabelResolver, resolverName, metricLabelResult, result).Add(1)
	}
}

// countIssuance counts a request to the ACME server. Nothing is counted when nothing was asked.
func countIssuance(resolverName, kind string, obtained bool, err error) {
	if !obtained && err == nil {
		return
	}

	if registry := metricsRegistry.Load(); registry != nil {
		(*registry).ACMEClusterIssuancesCounter().
			With(metricLabelResolver, resolverName, "kind", kind, metricLabelResult, metricResult(err)).Add(1)
	}
}

func countChallengeLookup(result string) {
	if registry := metricsRegistry.Load(); registry != nil {
		(*registry).ACMEClusterChallengeLookupsCounter().With(metricLabelResult, result).Add(1)
	}
}

func countStoreWrite(resolverName, target string, err error) {
	if registry := metricsRegistry.Load(); registry != nil {
		(*registry).ACMEStoreWritesCounter().
			With(metricLabelResolver, resolverName, "target", target, metricLabelResult, metricResult(err)).Add(1)
	}
}

func setConfigMismatch(resolverName string, mismatch bool) {
	registry := metricsRegistry.Load()
	if registry == nil {
		return
	}

	value := 0.0
	if mismatch {
		value = 1
	}

	(*registry).ACMEClusterConfigMismatchGauge().With(metricLabelResolver, resolverName).Set(value)
}

func metricResult(err error) string {
	if err != nil {
		return resultFailure
	}

	return resultSuccess
}

// storeTarget names a store in the metrics.
func storeTarget(store Store) string {
	switch store.(type) {
	case *RedisStore:
		return StoreRedis
	case *DirStore:
		return StoreDir
	default:
		return StoreFile
	}
}
