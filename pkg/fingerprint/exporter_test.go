package fingerprint

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	gokitmetrics "github.com/go-kit/kit/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/observability/metrics"
)

// testCounter records the counts by label values.
type testCounter struct {
	mu     *sync.Mutex
	values map[string]float64
	labels string
}

func newTestCounter() *testCounter {
	return &testCounter{mu: &sync.Mutex{}, values: map[string]float64{}}
}

func (c *testCounter) With(labelValues ...string) gokitmetrics.Counter {
	return &testCounter{mu: c.mu, values: c.values, labels: strings.Join(labelValues, ",")}
}

func (c *testCounter) Add(delta float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[c.labels] += delta
}

func (c *testCounter) get(labels string) float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.values[labels]
}

type batch struct {
	Schema     string            `json:"schema"`
	Sender     string            `json:"sender"`
	EntryPoint string            `json:"entryPoint"`
	Records    []json.RawMessage `json:"records"`
}

// collector is an export destination recording the batches it receives.
type collector struct {
	mu      sync.Mutex
	batches []batch
	headers []http.Header
	status  int
	delay   time.Duration
}

func (c *collector) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if c.delay > 0 {
		select {
		case <-time.After(c.delay):
		case <-req.Context().Done():
			return
		}
	}

	body, _ := io.ReadAll(req.Body)
	var b batch
	_ = json.Unmarshal(body, &b)

	c.mu.Lock()
	c.batches = append(c.batches, b)
	c.headers = append(c.headers, req.Header.Clone())
	status := c.status
	c.mu.Unlock()

	if status != 0 {
		rw.WriteHeader(status)
	}
}

func (c *collector) snapshot() []batch {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]batch(nil), c.batches...)
}

func (c *collector) records() int {
	n := 0
	for _, b := range c.snapshot() {
		n += len(b.Records)
	}
	return n
}

type testExporter struct {
	*exporter

	dropped *testCounter
	batches *testCounter
}

func newTestExporter(t *testing.T, url string, change func(*exporterConfig)) *testExporter {
	t.Helper()

	cfg := exporterConfig{
		EntryPoint:    "ep",
		URL:           url,
		Headers:       http.Header{"Authorization": []string{"Bearer secret"}},
		QueueSize:     4096,
		QueueMaxBytes: 16 << 20,
		BatchSize:     64,
		FlushInterval: time.Hour,
		SendTimeout:   2 * time.Second,
	}
	if change != nil {
		change(&cfg)
	}

	x := newExporter(cfg, metrics.NewVoidRegistry())
	tx := &testExporter{exporter: x, dropped: newTestCounter(), batches: newTestCounter()}
	x.dropped = tx.dropped
	x.queueFull = tx.dropped.With("entrypoint", "ep", "reason", dropQueueFull)
	x.batches = tx.batches
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		x.close(ctx)
	})

	return tx
}

func testRecord(i int) *requestRecord {
	return &requestRecord{Type: recordRequest, Method: "GET", Path: "/" + strings.Repeat("x", i%10), Status: 200}
}

func TestExporter_batchBySize(t *testing.T) {
	c := &collector{}
	srv := httptest.NewServer(c)
	t.Cleanup(srv.Close)

	x := newTestExporter(t, srv.URL, func(cfg *exporterConfig) { cfg.BatchSize = 3 })
	for i := range 3 {
		x.enqueue(recordRequest, testRecord(i), 100)
	}

	require.Eventually(t, func() bool { return c.records() == 3 }, 5*time.Second, 5*time.Millisecond)

	batches := c.snapshot()
	require.Len(t, batches, 1)
	assert.Equal(t, Schema, batches[0].Schema)
	assert.Equal(t, "ep", batches[0].EntryPoint)
	assert.True(t, strings.HasPrefix(batches[0].Sender, "traefix/"))

	c.mu.Lock()
	assert.Equal(t, "Bearer secret", c.headers[0].Get("Authorization"))
	assert.Equal(t, "application/json", c.headers[0].Get("Content-Type"))
	c.mu.Unlock()
	assert.InDelta(t, 1, x.batches.get("entrypoint,ep,code,200"), 0)
}

func TestExporter_batchByTime(t *testing.T) {
	c := &collector{}
	srv := httptest.NewServer(c)
	t.Cleanup(srv.Close)

	x := newTestExporter(t, srv.URL, func(cfg *exporterConfig) { cfg.FlushInterval = 50 * time.Millisecond })
	x.enqueue(recordRequest, testRecord(0), 100)

	require.Eventually(t, func() bool { return c.records() == 1 }, 5*time.Second, 5*time.Millisecond)
}

func TestExporter_batchLimit(t *testing.T) {
	c := &collector{}
	srv := httptest.NewServer(c)
	t.Cleanup(srv.Close)

	x := newTestExporter(t, srv.URL, func(cfg *exporterConfig) { cfg.BatchSize = maxBatchRecords })
	for i := range maxBatchRecords + 10 {
		x.enqueue(recordRequest, testRecord(i), 100)
	}

	// The first batch stops at the record limit; the rest leaves on close.
	require.Eventually(t, func() bool { return c.records() == maxBatchRecords }, 5*time.Second, 5*time.Millisecond)
	assert.Len(t, c.snapshot()[0].Records, maxBatchRecords)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	x.close(ctx)
	assert.Equal(t, maxBatchRecords+10, c.records())
}

func TestExporter_queueFull(t *testing.T) {
	x := newTestExporter(t, "http://127.0.0.1:1", func(cfg *exporterConfig) {
		cfg.QueueSize = 2
		cfg.BatchSize = 100
	})

	for i := range 5 {
		x.enqueue(recordRequest, testRecord(i), 100)
	}

	assert.Equal(t, 2, x.queued())
	assert.InDelta(t, 3, x.dropped.get("entrypoint,ep,reason,queue_full"), 0)
}

func TestExporter_queueMaxBytes(t *testing.T) {
	x := newTestExporter(t, "http://127.0.0.1:1", func(cfg *exporterConfig) {
		cfg.QueueMaxBytes = 250
		cfg.BatchSize = 100
	})

	x.enqueue(recordRequest, testRecord(0), 100)
	x.enqueue(recordRequest, testRecord(1), 100)
	x.enqueue(recordRequest, testRecord(2), 100)
	x.enqueue(recordRequest, testRecord(3), 300)

	assert.Equal(t, 2, x.queued())
	assert.InDelta(t, 1, x.dropped.get("entrypoint,ep,reason,queue_full"), 0)
	assert.InDelta(t, 1, x.dropped.get("entrypoint,ep,reason,too_large"), 0)
}

func TestExporter_destinationErrors(t *testing.T) {
	testCases := []struct {
		desc   string
		status int
		reason string
		code   string
	}{
		{desc: "413", status: http.StatusRequestEntityTooLarge, reason: dropHTTP4xx, code: "413"},
		{desc: "429", status: http.StatusTooManyRequests, reason: dropHTTP4xx, code: "429"},
		{desc: "500", status: http.StatusInternalServerError, reason: dropHTTP5xx, code: "500"},
		{desc: "503", status: http.StatusServiceUnavailable, reason: dropHTTP5xx, code: "503"},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			c := &collector{status: test.status}
			srv := httptest.NewServer(c)
			t.Cleanup(srv.Close)

			x := newTestExporter(t, srv.URL, func(cfg *exporterConfig) { cfg.BatchSize = 2 })
			x.enqueue(recordRequest, testRecord(0), 100)
			x.enqueue(recordRequest, testRecord(1), 100)

			require.Eventually(t, func() bool {
				return x.dropped.get("entrypoint,ep,reason,"+test.reason) == 2
			}, 5*time.Second, 5*time.Millisecond)
			assert.InDelta(t, 1, x.batches.get("entrypoint,ep,code,"+test.code), 0)
			// A single attempt per batch.
			assert.Len(t, c.snapshot(), 1)
		})
	}
}

func TestExporter_destinationDown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	x := newTestExporter(t, "http://"+addr, func(cfg *exporterConfig) { cfg.BatchSize = 1 })
	x.enqueue(recordRequest, testRecord(0), 100)

	require.Eventually(t, func() bool {
		return x.dropped.get("entrypoint,ep,reason,send_error") == 1
	}, 5*time.Second, 5*time.Millisecond)
	assert.InDelta(t, 1, x.batches.get("entrypoint,ep,code,error"), 0)
}

func TestExporter_destinationSlow(t *testing.T) {
	c := &collector{delay: time.Second}
	srv := httptest.NewServer(c)
	t.Cleanup(srv.Close)

	x := newTestExporter(t, srv.URL, func(cfg *exporterConfig) {
		cfg.BatchSize = 1
		cfg.SendTimeout = 100 * time.Millisecond
	})

	start := time.Now()
	for i := range 100 {
		x.enqueue(recordRequest, testRecord(i), 100)
	}
	// Queuing never waits on the destination.
	assert.Less(t, time.Since(start), 50*time.Millisecond)

	require.Eventually(t, func() bool {
		return x.dropped.get("entrypoint,ep,reason,send_error") >= 1
	}, 5*time.Second, 5*time.Millisecond)
}

func TestExporter_noRedirect(t *testing.T) {
	target := &collector{}
	targetSrv := httptest.NewServer(target)
	t.Cleanup(targetSrv.Close)

	redirect := httptest.NewServer(http.RedirectHandler(targetSrv.URL, http.StatusTemporaryRedirect))
	t.Cleanup(redirect.Close)

	x := newTestExporter(t, redirect.URL, func(cfg *exporterConfig) { cfg.BatchSize = 1 })
	x.enqueue(recordRequest, testRecord(0), 100)

	require.Eventually(t, func() bool {
		return x.batches.get("entrypoint,ep,code,307") == 1
	}, 5*time.Second, 5*time.Millisecond)
	assert.Empty(t, target.snapshot())
}
