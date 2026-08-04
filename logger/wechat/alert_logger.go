package wechat

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap/zapcore"
)

const (
	alertQueueSize = 50
	alertInterval  = time.Second
)

// NewAlertLogger creates the asynchronous alert queue used by the Zap core.
func NewAlertLogger(client *Client) *AlertLogger {
	return &AlertLogger{
		client:       client,
		queue:        make([]*alert, 0, alertQueueSize),
		queuedAlerts: make(map[alertKey]*alert),
	}
}

// AlertLogger buffers template alerts so application logging never waits on
// the WeChat API. At most one alert is consumed each second.
type AlertLogger struct {
	client *Client

	mu           sync.Mutex
	queue        []*alert
	queuedAlerts map[alertKey]*alert
	running      bool
}

type alertKey struct {
	level       string
	fingerprint string
}

type alert struct {
	key   alertKey
	data  string
	count int
}

// NewAlertCore creates a Zap core that receives the original entries instead
// of parsing an already encoded log line.
func NewAlertCore(client *Client, levelFilter func(zapcore.Level) bool) zapcore.Core {
	return &alertCore{
		logger:      NewAlertLogger(client),
		levelFilter: levelFilter,
	}
}

type alertCore struct {
	logger      *AlertLogger
	levelFilter func(zapcore.Level) bool
	fields      []zapcore.Field
}

func (c *alertCore) With(fields []zapcore.Field) zapcore.Core {
	clone := *c
	clone.fields = append(append([]zapcore.Field{}, c.fields...), fields...)
	return &clone
}

func (c *alertCore) Enabled(level zapcore.Level) bool {
	return c.levelFilter(level)
}

func (c *alertCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c *alertCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	allFields := append(append([]zapcore.Field{}, c.fields...), fields...)
	c.logger.Log(entry, allFields)
	return nil
}

func (c *alertCore) Sync() error {
	return nil
}

func formatAlert(entry zapcore.Entry, fields []zapcore.Field) string {
	parts := make([]string, 0, 3)
	if !entry.Time.IsZero() {
		parts = append(parts, entry.Time.Format("2006/01/02 15:04:05"))
	}
	if entry.Caller.Defined {
		parts = append(parts, entry.Caller.TrimmedPath())
	}
	if entry.Message != "" {
		parts = append(parts, entry.Message)
	}

	if len(fields) != 0 {
		fieldEncoder := zapcore.NewMapObjectEncoder()
		for _, field := range fields {
			field.AddTo(fieldEncoder)
		}
		if encodedFields, err := json.Marshal(fieldEncoder.Fields); err == nil && len(fieldEncoder.Fields) != 0 {
			parts = append(parts, string(encodedFields))
		}
	}
	return strings.Join(parts, " ")
}

func alertFingerprint(entry zapcore.Entry) string {
	caller := ""
	if entry.Caller.Defined {
		caller = entry.Caller.TrimmedPath()
	}
	return strings.Join([]string{entry.LoggerName, caller, entry.Message}, "\x00")
}

// Log adds an alert to the queue. The stable error fingerprint does not include
// timestamp or fields, so repeated instances of the same error are combined.
func (l *AlertLogger) Log(entry zapcore.Entry, fields []zapcore.Field) {
	key := alertKey{
		level:       strings.ToUpper(entry.Level.String()),
		fingerprint: alertFingerprint(entry),
	}
	data := formatAlert(entry, fields)

	l.mu.Lock()
	if queued := l.queuedAlerts[key]; queued != nil {
		queued.count++
		l.mu.Unlock()
		return
	}

	if len(l.queue) == alertQueueSize {
		dropped := l.queue[0]
		delete(l.queuedAlerts, dropped.key)
		l.queue = l.queue[1:]
		log.Printf("ERROR [AlertLogger] queue is full: %d", alertQueueSize)
	}

	a := &alert{key: key, data: data, count: 1}
	l.queue = append(l.queue, a)
	l.queuedAlerts[key] = a
	if !l.running {
		l.running = true
		go l.startSend()
	}
	l.mu.Unlock()
}

func (l *AlertLogger) getOne() *alert {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.queue) == 0 {
		l.running = false
		return nil
	}

	a := l.queue[0]
	l.queue = l.queue[1:]
	delete(l.queuedAlerts, a.key)
	return a
}

func (l *AlertLogger) startSend() {
	ticker := time.NewTicker(alertInterval)
	defer ticker.Stop()

	for range ticker.C {
		a := l.getOne()
		if a == nil {
			return
		}
		if err := l.client.SendAlert(a.key.level, a.data, a.count); err != nil {
			log.Printf("ERROR [AlertLogger] send alert: %v", err)
		}
	}
}
