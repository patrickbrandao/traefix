package coraza

import (
	"sync/atomic"
	"time"

	"github.com/corazawaf/coraza/v3/types"
	gokitmetrics "github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	"github.com/traefik/traefik/v3/pkg/observability/metrics"
)

var (
	requestsCounter  atomic.Value
	durationHisto    atomic.Value
	ruleMatchCounter atomic.Value
)

// SetMetricsRegistry sets the registry of the Coraza middleware metrics.
func SetMetricsRegistry(registry metrics.Registry) {
	if registry == nil {
		return
	}
	if c := registry.WAFRequestsCounter(); c != nil {
		requestsCounter.Store(c)
	}
	if h := registry.WAFDurationHistogram(); h != nil {
		durationHisto.Store(h)
	}
	if c := registry.WAFRuleMatchesCounter(); c != nil {
		ruleMatchCounter.Store(c)
	}
}

func requestsMetric() gokitmetrics.Counter {
	if c, ok := requestsCounter.Load().(gokitmetrics.Counter); ok {
		return c
	}
	return discard.NewCounter()
}

func ruleMatchesMetric() gokitmetrics.Counter {
	if c, ok := ruleMatchCounter.Load().(gokitmetrics.Counter); ok {
		return c
	}
	return discard.NewCounter()
}

// durationMetric returns the histogram of the WAF time, observed with ObserveFromStart so that each backend
// gets its own unit (seconds or milliseconds).
func durationMetric() metrics.ScalableHistogram {
	if h, ok := durationHisto.Load().(metrics.ScalableHistogram); ok {
		return h
	}
	return discardHistogram{}
}

type discardHistogram struct{}

func (d discardHistogram) With(...string) metrics.ScalableHistogram { return d }
func (discardHistogram) Observe(float64)                            {}
func (discardHistogram) ObserveFromStart(time.Time)                 {}

// logMatchedRule logs the rules that matched with a message, at the debug level: the audit log
// (SecAuditEngine) is the place of the detections in production.
func logMatchedRule(name string, rule types.MatchedRule) {
	log.Debug().Str(logs.MiddlewareName, name).Str(logs.MiddlewareType, typeName).
		Int("ruleId", rule.Rule().ID()).Str("severity", rule.Rule().Severity().String()).
		Msg(rule.ErrorLog())
}
