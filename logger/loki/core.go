package loki

import (
	"bytes"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Core adapts original Zap entries to JSON Loki log lines.
type Core struct {
	client  *Client
	labels  Labels
	encoder zapcore.Encoder
	level   zapcore.LevelEnabler
}

func NewCore(client *Client, labels Labels, level zapcore.LevelEnabler) *Core {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	encoderConfig.EncodeDuration = zapcore.MillisDurationEncoder
	c := &Core{
		client:  client,
		labels:  labels,
		encoder: zapcore.NewJSONEncoder(encoderConfig),
		level:   level,
	}
	return c
}

func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	clone := *c
	clone.encoder = c.encoder.Clone()
	for _, field := range fields {
		field.AddTo(clone.encoder)
	}
	return &clone
}

func (c *Core) Enabled(level zapcore.Level) bool {
	return c.client != nil && c.client.Enabled() && c.level.Enabled(level)
}

func (c *Core) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.Enabled(entry.Level) {
		return checked
	}
	return checked.AddCore(entry, c)
}

func (c *Core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	buf, err := c.encoder.EncodeEntry(entry, fields)
	if err != nil {
		return err
	}
	line := bytes.TrimSuffix(buf.Bytes(), []byte{'\n'})
	c.client.Push(c.labels, entry.Time, line)
	buf.Free()
	return nil
}

func (c *Core) Sync() error { return nil }
