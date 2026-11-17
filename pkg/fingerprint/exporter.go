package fingerprint

import (
	"bytes"
	"context"
	"io"
	"maps"
	"net/http"
	"strconv"
	"sync"
	"time"

	gokitmetrics "github.com/go-kit/kit/metrics"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/observability/metrics"
	"github.com/traefik/traefik/v3/pkg/version"
)

const (
	maxBatchRecords = 1024
	maxBatchBytes   = 4 << 20

	dropQueueFull = "queue_full"
	dropSendError = "send_error"
	dropHTTP4xx   = "http_4xx"
	dropHTTP5xx   = "http_5xx"
	dropTooLarge  = "too_large"

	// logInterval bounds the drop logs to one line per interval, so that a destination down does not flood the logs.
	logInterval = time.Minute
)

type exporterConfig struct {
	EntryPoint    string
	URL           string
	Headers       http.Header
	QueueSize     int
	QueueMaxBytes int
	BatchSize     int
	FlushInterval time.Duration
	SendTimeout   time.Duration
}

type queuedRecord struct {
	record jsonAppender
	size   int
}

// exporter sends the records in batches from its own goroutine, so that a request never waits on the destination:
// the request side only appends to a bounded queue, and drops the record when the queue is full.
type exporter struct {
	cfg    exporterConfig
	client *http.Client

	mu         sync.Mutex
	queue      []queuedRecord
	queueBytes int

	wake     chan struct{}
	stop     chan struct{}
	stopOnce sync.Once
	done     chan struct{}

	// The labeled metrics of the request path are resolved once, as resolving labels allocates.
	connRecords gokitmetrics.Counter
	reqRecords  gokitmetrics.Counter
	queueFull   gokitmetrics.Counter
	dropped     gokitmetrics.Counter
	batches     gokitmetrics.Counter
	queueSize   gokitmetrics.Gauge
	send        metrics.ScalableHistogram

	dropLog rateLog
}

func newExporter(cfg exporterConfig, registry metrics.Registry) *exporter {
	x := &exporter{
		cfg: cfg,
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     90 * time.Second,
				ForceAttemptHTTP2:   true,
			},
			// A redirection would send the batch, and its headers, somewhere the operator did not configure.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		wake:        make(chan struct{}, 1),
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
		connRecords: registry.FingerprintExportRecordsCounter().With("entrypoint", cfg.EntryPoint, "type", recordConnection),
		reqRecords:  registry.FingerprintExportRecordsCounter().With("entrypoint", cfg.EntryPoint, "type", recordRequest),
		queueFull:   registry.FingerprintExportDroppedCounter().With("entrypoint", cfg.EntryPoint, "reason", dropQueueFull),
		dropped:     registry.FingerprintExportDroppedCounter(),
		batches:     registry.FingerprintExportBatchesCounter(),
		queueSize:   registry.FingerprintExportQueueRecordsGauge().With("entrypoint", cfg.EntryPoint),
		send:        registry.FingerprintExportSendHistogram(),
	}
	x.dropLog.entryPoint = cfg.EntryPoint

	go x.run()

	return x
}

// enqueue queues a record without ever blocking on the destination.
func (x *exporter) enqueue(recordType string, record jsonAppender, size int) {
	if size > x.cfg.QueueMaxBytes || size > maxBatchBytes {
		x.drop(dropTooLarge, 1)
		return
	}

	x.mu.Lock()
	if len(x.queue) >= x.cfg.QueueSize || x.queueBytes+size > x.cfg.QueueMaxBytes {
		x.mu.Unlock()
		x.queueFull.Add(1)
		x.dropLog.add(dropQueueFull, 1)
		return
	}
	x.queue = append(x.queue, queuedRecord{record: record, size: size})
	x.queueBytes += size
	n := len(x.queue)
	x.mu.Unlock()

	if recordType == recordConnection {
		x.connRecords.Add(1)
	} else {
		x.reqRecords.Add(1)
	}

	if n >= x.cfg.BatchSize {
		select {
		case x.wake <- struct{}{}:
		default:
		}
	}
}

func (x *exporter) drop(reason string, n int) {
	x.dropped.With("entrypoint", x.cfg.EntryPoint, "reason", reason).Add(float64(n))
	x.dropLog.add(reason, n)
}

func (x *exporter) queued() int {
	x.mu.Lock()
	defer x.mu.Unlock()
	return len(x.queue)
}

func (x *exporter) run() {
	defer close(x.done)

	ticker := time.NewTicker(x.cfg.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-x.stop:
			// The records left are sent once, then the remainder is dropped with the process.
			for x.queued() > 0 {
				x.sendBatch()
			}
			return
		case <-x.wake:
			for x.queued() >= x.cfg.BatchSize {
				x.sendBatch()
			}
		case <-ticker.C:
			x.queueSize.Set(float64(x.queued()))
			if x.queued() > 0 {
				x.sendBatch()
			}
			for x.queued() >= x.cfg.BatchSize {
				x.sendBatch()
			}
		}
	}
}

// sendBatch sends the oldest records, up to the batch limits, in a single attempt.
func (x *exporter) sendBatch() {
	x.mu.Lock()
	n := min(len(x.queue), maxBatchRecords)
	batch := make([]queuedRecord, n)
	copy(batch, x.queue[:n])
	clear(x.queue[:n])
	x.queue = x.queue[n:]
	for _, r := range batch {
		x.queueBytes -= r.size
	}
	left := len(x.queue)
	x.mu.Unlock()
	x.queueSize.Set(float64(left))

	body, count, rest := x.encode(batch)
	if len(rest) > 0 {
		// The records past the byte limit go back to the front of the queue, for the next batch.
		x.requeue(rest)
	}
	if count == 0 {
		return
	}

	start := time.Now()
	code, err := x.post(body)
	x.send.With("entrypoint", x.cfg.EntryPoint).ObserveFromStart(start)

	switch {
	case err != nil:
		x.batches.With("entrypoint", x.cfg.EntryPoint, "code", "error").Add(1)
		x.drop(dropSendError, count)
		x.dropLog.setError(err)
	case code >= http.StatusInternalServerError:
		x.batches.With("entrypoint", x.cfg.EntryPoint, "code", strconv.Itoa(code)).Add(1)
		x.drop(dropHTTP5xx, count)
	case code >= http.StatusBadRequest:
		x.batches.With("entrypoint", x.cfg.EntryPoint, "code", strconv.Itoa(code)).Add(1)
		x.drop(dropHTTP4xx, count)
	default:
		x.batches.With("entrypoint", x.cfg.EntryPoint, "code", strconv.Itoa(code)).Add(1)
	}
}

func (x *exporter) requeue(records []queuedRecord) {
	x.mu.Lock()
	x.queue = append(records, x.queue...)
	for _, r := range records {
		x.queueBytes += r.size
	}
	n := len(x.queue)
	x.mu.Unlock()
	x.queueSize.Set(float64(n))
}

// encode writes the batch JSON, and returns the records that did not fit in the batch byte limit.
func (x *exporter) encode(batch []queuedRecord) ([]byte, int, []queuedRecord) {
	head := []byte(`{"schema":`)
	head = appendJSONString(head, Schema)
	head = append(head, `,"sender":`...)
	head = appendJSONString(head, "traefix/"+version.Version)
	head = append(head, `,"entryPoint":`...)
	head = appendJSONString(head, x.cfg.EntryPoint)
	head = append(head, `,"sentAt":`...)
	head = appendJSONTime(head, time.Now().UTC())

	var buf bytes.Buffer
	buf.Write(head)
	buf.WriteString(`,"records":[`)

	count := 0
	var b []byte
	for i, r := range batch {
		b = r.record.appendJSON(b[:0])
		if len(b) > maxBatchBytes {
			x.drop(dropTooLarge, 1)
			continue
		}
		if buf.Len()+len(b)+2 > maxBatchBytes {
			return x.closeBatch(&buf), count, batch[i:]
		}
		if count > 0 {
			buf.WriteByte(',')
		}
		buf.Write(b)
		count++
	}

	return x.closeBatch(&buf), count, nil
}

func (x *exporter) closeBatch(buf *bytes.Buffer) []byte {
	buf.WriteString("]}")
	return buf.Bytes()
}

func (x *exporter) post(body []byte) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), x.cfg.SendTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, x.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	maps.Copy(req.Header, x.cfg.Headers)
	req.Header.Set("Content-Type", "application/json")

	resp, err := x.client.Do(req)
	if err != nil {
		return 0, err
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	_ = resp.Body.Close()

	return resp.StatusCode, nil
}

// close stops the exporter after a last attempt to send the queued records, bounded by ctx.
func (x *exporter) close(ctx context.Context) {
	x.stopOnce.Do(func() { close(x.stop) })

	select {
	case <-x.done:
	case <-ctx.Done():
	}
	x.client.CloseIdleConnections()
}

// rateLog logs the dropped records at most once per logInterval, with the counts since the previous line.
type rateLog struct {
	entryPoint string

	mu      sync.Mutex
	last    time.Time
	counts  map[string]int
	lastErr error
}

func (l *rateLog) setError(err error) {
	l.mu.Lock()
	l.lastErr = err
	l.mu.Unlock()
}

func (l *rateLog) add(reason string, n int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.counts == nil {
		l.counts = make(map[string]int)
	}
	l.counts[reason] += n

	now := time.Now()
	if now.Sub(l.last) < logInterval {
		return
	}
	l.last = now

	event := log.Warn().Str("entryPoint", l.entryPoint)
	for r, c := range l.counts {
		event = event.Int(r, c)
	}
	if l.lastErr != nil {
		event = event.Err(l.lastErr)
	}
	event.Msg("Dropped fingerprint records")

	clear(l.counts)
	l.lastErr = nil
}
