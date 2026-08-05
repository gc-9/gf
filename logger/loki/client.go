package loki

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const pushPath = "/loki/api/v1/push"

// Client accepts raw Loki entries through Push and asynchronously delivers
// them in batches. Push never waits for a network request.
type Client struct {
	enabled bool

	endpoint    string
	tenantID    string
	username    string
	password    string
	bearerToken string

	batchSize      int
	batchWait      time.Duration
	requestTimeout time.Duration
	maxRetries     int

	httpClient *http.Client
	queue      chan record
	flush      chan chan error
	stop       chan struct{}
	done       chan struct{}
	stopOnce   sync.Once

	dropped       atomic.Uint64
	sent          atomic.Uint64
	failedBatches atomic.Uint64
}

// NewClient creates and starts a client. A disabled config creates a no-op
// client, which keeps Loki optional for applications using the same Fx graph.
func NewClient(conf *Config) (*Client, error) {
	if conf == nil || !conf.Enabled {
		return &Client{}, nil
	}
	confCopy := *conf
	conf = &confCopy

	endpoint, err := normalizeEndpoint(conf.URL)
	if err != nil {
		return nil, err
	}

	if conf.QueueCapacity <= 0 {
		conf.QueueCapacity = 10_000
	}
	if conf.BatchSize <= 0 {
		conf.BatchSize = 1_000
	}
	if conf.BatchWait <= 0 {
		conf.BatchWait = time.Second
	}
	if conf.RequestTimeout <= 0 {
		conf.RequestTimeout = 5 * time.Second
	}
	if conf.MaxRetries < 0 {
		conf.MaxRetries = 0
	}

	c := &Client{
		enabled:        true,
		endpoint:       endpoint,
		tenantID:       conf.TenantID,
		username:       conf.Username,
		password:       conf.Password,
		bearerToken:    conf.BearerToken,
		batchSize:      conf.BatchSize,
		batchWait:      conf.BatchWait,
		requestTimeout: conf.RequestTimeout,
		maxRetries:     conf.MaxRetries,
		httpClient:     &http.Client{},
		queue:          make(chan record, conf.QueueCapacity),
		flush:          make(chan chan error),
		stop:           make(chan struct{}),
		done:           make(chan struct{}),
	}
	go c.run()
	return c, nil
}

func normalizeEndpoint(rawURL string) (string, error) {
	if rawURL == "" {
		return "", errors.New("loki url is required when Loki is enabled")
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid loki url %q", rawURL)
	}
	if !strings.HasSuffix(u.Path, pushPath) {
		u.Path = strings.TrimRight(u.Path, "/") + pushPath
	}
	return u.String(), nil
}

// Enabled reports whether records are accepted for delivery.
func (c *Client) Enabled() bool { return c != nil && c.enabled }

// Push accepts one raw Loki entry. labels are copied before queuing and line
// is copied because callers commonly reuse encoding buffers. It returns false
// only when Loki is disabled, stopped, or its bounded queue is full.
func (c *Client) Push(labels Labels, timestamp time.Time, line []byte) bool {
	if !c.Enabled() {
		return false
	}
	_, labels = labels.canonical()
	item := record{
		labels: labels,
		time:   timestamp,
		line:   bytes.Clone(line),
	}
	select {
	case c.queue <- item:
		return true
	default:
		c.dropped.Add(1)
		return false
	}
}

// Sync flushes all entries accepted before the call. It is intended for an
// application's shutdown hook.
func (c *Client) Sync(ctx context.Context) error {
	if !c.Enabled() {
		return nil
	}
	ack := make(chan error, 1)
	select {
	case c.flush <- ack:
	case <-ctx.Done():
		return ctx.Err()
	case <-c.done:
		return nil
	}
	select {
	case err := <-ack:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close stops the worker after flushing queued entries until ctx expires.
func (c *Client) Close(ctx context.Context) error {
	if !c.Enabled() {
		return nil
	}
	err := c.Sync(ctx)
	c.stopOnce.Do(func() { close(c.stop) })
	select {
	case <-c.done:
	case <-ctx.Done():
		if err == nil {
			err = ctx.Err()
		}
	}
	return err
}

func (c *Client) Stats() Stats {
	return Stats{
		Dropped:       c.dropped.Load(),
		Sent:          c.sent.Load(),
		FailedBatches: c.failedBatches.Load(),
	}
}

func (c *Client) run() {
	defer close(c.done)

	batches := make(map[string]*stream)
	count := 0
	timer := time.NewTimer(c.batchWait)
	if !timer.Stop() {
		<-timer.C
	}
	var timerC <-chan time.Time

	resetTimer := func() {
		if timerC != nil {
			return
		}
		timer.Reset(c.batchWait)
		timerC = timer.C
	}
	stopTimer := func() {
		if timerC == nil {
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timerC = nil
	}
	flush := func() error {
		if count == 0 {
			return nil
		}
		payload := pushPayload{Streams: make([]stream, 0, len(batches))}
		for _, s := range batches {
			payload.Streams = append(payload.Streams, *s)
		}
		err := c.send(payload)
		if err != nil {
			c.failedBatches.Add(1)
			c.dropped.Add(uint64(count))
		} else {
			c.sent.Add(uint64(count))
		}
		batches = make(map[string]*stream)
		count = 0
		stopTimer()
		return err
	}
	appendRecord := func(item record) {
		key, normalized := item.labels.canonical()
		s, ok := batches[key]
		if !ok {
			s = &stream{Stream: normalized}
			batches[key] = s
		}
		s.Values = append(s.Values, [2]string{
			strconv.FormatInt(item.time.UnixNano(), 10),
			string(item.line),
		})
		count++
		resetTimer()
	}

	for {
		select {
		case item := <-c.queue:
			appendRecord(item)
			if count >= c.batchSize {
				_ = flush()
			}
		case <-timerC:
			_ = flush()
		case ack := <-c.flush:
			draining := true
			for draining {
				select {
				case item := <-c.queue:
					appendRecord(item)
				default:
					draining = false
				}
			}
			ack <- flush()
		case <-c.stop:
			_ = flush()
			return
		}
	}
}

func (c *Client) send(payload pushPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), c.requestTimeout)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
		if err != nil {
			lastErr = err
		} else {
			req.Header.Set("Content-Type", "application/json")
			if c.tenantID != "" {
				req.Header.Set("X-Scope-OrgID", c.tenantID)
			}
			if c.bearerToken != "" {
				req.Header.Set("Authorization", "Bearer "+c.bearerToken)
			} else if c.username != "" {
				req.SetBasicAuth(c.username, c.password)
			}

			resp, doErr := c.httpClient.Do(req)
			if doErr == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
					cancel()
					return nil
				}
				doErr = fmt.Errorf("loki push returned %s", resp.Status)
			}
			lastErr = doErr
		}
		cancel()

		if attempt < c.maxRetries {
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}
	return lastErr
}
