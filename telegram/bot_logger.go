package telegram

import (
	"encoding/json"
	"log"
	"strings"
	"sync"

	"go.uber.org/zap/zapcore"
)

func NewBotLogger(bot *Bot, maxQueueSize int) *BotLogger {
	return &BotLogger{
		bot:          bot,
		maxQueueSize: maxQueueSize,
		q:            make([]string, 0),
		lock:         sync.Mutex{},
	}
}

type BotLogger struct {
	bot          *Bot
	q            []string
	lock         sync.Mutex
	maxQueueSize int
	isStart      bool
}

// Log adds an original Zap entry to the Telegram delivery queue.
func (t *BotLogger) Log(entry zapcore.Entry, fields []zapcore.Field) {
	t.lock.Lock()
	t.q = append(t.q, formatLogEntry(entry, fields))
	if len(t.q) > t.maxQueueSize {
		t.q = t.q[1:]
		log.Printf("ERROR [BotLogger] queue is full: %v", t.maxQueueSize)
	}
	t.lock.Unlock()
	t.onceStart()
}

func (t *BotLogger) getOne() (string, bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if len(t.q) == 0 {
		t.isStart = false
		return "", false
	}
	m := t.q[0]
	t.q = t.q[1:]
	return m, true
}

func (t *BotLogger) onceStart() {
	t.lock.Lock()
	defer t.lock.Unlock()
	if !t.isStart {
		t.isStart = true
		go t.startSend()
	}
}

func (t *BotLogger) startSend() {
	for {
		m, ok := t.getOne()
		if !ok {
			return
		}
		err := t.bot.SendMessage(m)
		if err != nil {
			log.Println("ERROR [BotLogger] err send message:", err)
		}
	}
}

// NewTgBotLogCore creates a Telegram core that receives original Zap entries.
func NewTgBotLogCore(bot *Bot, levelFilter func(zapcore.Level) bool) zapcore.Core {
	return &botLogCore{
		logger:      NewBotLogger(bot, 10),
		levelFilter: levelFilter,
	}
}

type botLogCore struct {
	logger      *BotLogger
	levelFilter func(zapcore.Level) bool
	fields      []zapcore.Field
}

func (c *botLogCore) With(fields []zapcore.Field) zapcore.Core {
	clone := *c
	clone.fields = append(append([]zapcore.Field{}, c.fields...), fields...)
	return &clone
}

func (c *botLogCore) Enabled(level zapcore.Level) bool {
	return c.levelFilter(level)
}

func (c *botLogCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c *botLogCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	allFields := append(append([]zapcore.Field{}, c.fields...), fields...)
	c.logger.Log(entry, allFields)
	return nil
}

func (c *botLogCore) Sync() error {
	return nil
}

func formatLogEntry(entry zapcore.Entry, fields []zapcore.Field) string {
	parts := make([]string, 0, 4)
	if !entry.Time.IsZero() {
		parts = append(parts, entry.Time.Format("2006/01/02 15:04:05"))
	}
	parts = append(parts, strings.ToUpper(entry.Level.String()))
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
