package coraza

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	corazawaf "github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/experimental"
	"github.com/corazawaf/coraza/v3/types"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/middlewares"
	"github.com/traefik/traefik/v3/pkg/middlewares/observability"
)

const typeName = "Coraza"

const (
	resultAllowed  = "allowed"
	resultDetected = "detected"
	resultBlocked  = "blocked"
	resultError    = "error"
)

// wafs shares the compiled WAFs between the middlewares of the same configuration: a middleware is built once
// per router and entry point, and compiling the Core Rule Set takes time and memory.
var wafs = struct {
	sync.Mutex

	byKey map[string]*cachedWAF
}{byKey: map[string]*cachedWAF{}}

type cachedWAF struct {
	waf      corazawaf.WAF
	lastUsed time.Time
}

// wafCacheTTL bounds the life of the WAFs no configuration builds anymore.
const wafCacheTTL = time.Hour

type waf struct {
	next     http.Handler
	name     string
	waf      corazawaf.WAF
	failOpen bool
}

// New builds the native Coraza WAF middleware.
func New(ctx context.Context, next http.Handler, config dynamic.Coraza, name string) (http.Handler, error) {
	logger := middlewares.GetLogger(ctx, name, typeName)
	logger.Debug().Msg("Creating middleware")

	if len(config.Directives) == 0 {
		return nil, errors.New("no directives")
	}

	w, err := getWAF(name, config)
	if err != nil {
		return nil, err
	}

	return &waf{next: next, name: name, waf: w, failOpen: config.FailOpen}, nil
}

func getWAF(name string, config dynamic.Coraza) (corazawaf.WAF, error) {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%t\x00", config.CRSEnabled)
	for _, d := range config.Directives {
		_, _ = io.WriteString(h, d)
		_, _ = h.Write([]byte{0})
	}
	key := name + "\x00" + hex.EncodeToString(h.Sum(nil))

	wafs.Lock()
	defer wafs.Unlock()

	now := time.Now()
	for k, c := range wafs.byKey {
		if now.Sub(c.lastUsed) > wafCacheTTL {
			delete(wafs.byKey, k)
		}
	}

	if c, ok := wafs.byKey[key]; ok {
		c.lastUsed = now
		return c.waf, nil
	}

	cfg := corazawaf.NewWAFConfig().
		WithRootFS(newRulesFS(config.CRSEnabled)).
		WithErrorCallback(func(rule types.MatchedRule) {
			onLoggedMatch(name, rule)
		})
	for _, d := range config.Directives {
		cfg = cfg.WithDirectives(d)
	}

	w, err := corazawaf.NewWAF(cfg)
	if err != nil {
		return nil, fmt.Errorf("compiling the directives: %w", err)
	}

	wafs.byKey[key] = &cachedWAF{waf: w, lastUsed: now}

	return w, nil
}

func (w *waf) GetTracingInformation() (string, string) {
	return w.name, typeName
}

func (w *waf) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var tx types.Transaction
	if ctxWAF, ok := w.waf.(experimental.WAFWithOptions); ok {
		tx = ctxWAF.NewTransactionWithOptions(experimental.Options{Context: req.Context()})
	} else {
		tx = w.waf.NewTransaction()
	}

	m := &meter{}
	result := resultAllowed
	defer func() {
		// Phase 5 runs the logging rules and writes the audit log.
		tx.ProcessLogging()
		w.record(tx, result, m.elapsed)
		if err := tx.Close(); err != nil {
			middlewares.GetLogger(req.Context(), w.name, typeName).Debug().Err(err).Msg("Closing the WAF transaction")
		}
	}()

	if tx.IsRuleEngineOff() {
		w.next.ServeHTTP(rw, req)
		return
	}

	start := time.Now()
	it, err := processRequest(tx, req)
	m.add(start)

	switch {
	case err != nil:
		result = resultError
		middlewares.GetLogger(req.Context(), w.name, typeName).Error().Err(err).Msg("Processing the request in the WAF")
		if w.failOpen {
			w.next.ServeHTTP(rw, req)
			return
		}
		observability.SetStatusErrorf(req.Context(), "WAF error: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return

	case it != nil:
		result = resultBlocked
		observability.SetStatusErrorf(req.Context(), "WAF interruption by rule %d", it.RuleID)
		rw.WriteHeader(statusFromInterruption(it, http.StatusForbidden))
		return
	}

	ww := newResponseWriter(rw, tx, req.Proto, m)
	w.next.ServeHTTP(ww, req)

	start = time.Now()
	err = ww.finish()
	m.add(start)

	switch {
	case err != nil:
		result = resultError
		middlewares.GetLogger(req.Context(), w.name, typeName).Error().Err(err).Msg("Processing the response in the WAF")
	case tx.IsInterrupted():
		result = resultBlocked
	}
}

// record counts the request. Without an interruption, a logged rule match is a detection,
// e.g. with SecRuleEngine DetectionOnly or below the anomaly score threshold.
func (w *waf) record(tx types.Transaction, result string, elapsed time.Duration) {
	if _, matched := loggedMatches.LoadAndDelete(tx.ID()); matched && result == resultAllowed {
		result = resultDetected
	}

	requestsMetric().With("middleware", w.name, "result", result).Add(1)
	durationMetric().With("middleware", w.name).ObserveFromStart(time.Now().Add(-elapsed))
}

// loggedMatches holds the transactions with a logged rule match, until they are recorded.
var loggedMatches sync.Map

// onLoggedMatch is called by the WAF for the rules matched with the log action, which excludes the setup rules
// of the Core Rule Set, marked nolog.
func onLoggedMatch(name string, rule types.MatchedRule) {
	id := rule.Rule().ID()
	if id == 0 {
		return
	}

	loggedMatches.Store(rule.TransactionID(), struct{}{})
	ruleMatchesMetric().With("middleware", name, "rule_id", strconv.Itoa(id)).Add(1)
	logMatchedRule(name, rule)
}

// meter adds up the time spent in the WAF, apart from the next handlers.
type meter struct {
	mu      sync.Mutex
	elapsed time.Duration
}

func (m *meter) add(start time.Time) {
	m.mu.Lock()
	m.elapsed += time.Since(start)
	m.mu.Unlock()
}

func statusFromInterruption(it *types.Interruption, defaultStatus int) int {
	if it.Action == "deny" && it.Status != 0 {
		return it.Status
	}
	if it.Action == "deny" {
		return http.StatusForbidden
	}
	return defaultStatus
}

// processRequest runs the phases 1 and 2, from the connection and the request.
func processRequest(tx types.Transaction, req *http.Request) (*types.Interruption, error) {
	client, clientPort := splitHostPort(req.RemoteAddr)

	var server string
	var serverPort int
	if addr, ok := req.Context().Value(http.LocalAddrContextKey).(net.Addr); ok {
		server, serverPort = splitHostPort(addr.String())
	}

	tx.ProcessConnection(client, clientPort, server, serverPort)

	// The request URI after the path normalizations of the entry point, as the backend gets it.
	tx.ProcessURI(req.URL.RequestURI(), req.Method, req.Proto)

	for name, values := range req.Header {
		for _, v := range values {
			tx.AddRequestHeader(name, v)
		}
	}
	// net/http moves the Host and Transfer-Encoding headers out of req.Header: the rules expect them,
	// e.g. CRS rule 920171 to detect request smuggling.
	if req.Host != "" {
		tx.AddRequestHeader("Host", req.Host)
		tx.SetServerName(hostWithoutPort(req.Host))
	}
	for _, te := range req.TransferEncoding {
		tx.AddRequestHeader("Transfer-Encoding", te)
	}

	if it := tx.ProcessRequestHeaders(); it != nil {
		return it, nil
	}

	if tx.IsRequestBodyAccessible() && req.Body != nil && req.Body != http.NoBody {
		it, _, err := tx.ReadRequestBodyFrom(req.Body)
		if err != nil {
			return nil, fmt.Errorf("reading the request body: %w", err)
		}
		if it != nil {
			return it, nil
		}

		buffered, err := tx.RequestBodyReader()
		if err != nil {
			return nil, fmt.Errorf("reading the buffered request body: %w", err)
		}
		// The bytes past the WAF limit are still in the original body, after the buffered ones.
		req.Body = readCloser{Reader: io.MultiReader(buffered, req.Body), Closer: req.Body}
	}

	return tx.ProcessRequestBody()
}

type readCloser struct {
	io.Reader
	io.Closer
}

func splitHostPort(addr string) (string, int) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, 0
	}
	p, _ := strconv.Atoi(port)
	return host, p
}

func hostWithoutPort(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return strings.Trim(host, "[]")
}
