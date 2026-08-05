// Package filelog provides a rotating Zap file core.
package filelog

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	DefaultMaxSize    = 500 // megabytes
	DefaultMaxBackups = 7
	DefaultMaxAge     = 7 // days
)

// Config configures a rotating file log.
type Config struct {
	Filename   string `json:"filename" yaml:"filename"`
	MaxSize    int    `json:"maxSize" yaml:"maxSize"`
	MaxBackups int    `json:"maxBackups" yaml:"maxBackups"`
	MaxAge     int    `json:"maxAge" yaml:"maxAge"`
}

// DefaultConfig returns the default rotating-log settings for filename.
func DefaultConfig(filename string) Config {
	return Config{
		Filename:   filename,
		MaxSize:    DefaultMaxSize,
		MaxBackups: DefaultMaxBackups,
		MaxAge:     DefaultMaxAge,
	}
}

func (c Config) withDefaults() Config {
	defaults := DefaultConfig(c.Filename)
	if c.MaxSize <= 0 {
		c.MaxSize = defaults.MaxSize
	}
	if c.MaxBackups <= 0 {
		c.MaxBackups = defaults.MaxBackups
	}
	if c.MaxAge <= 0 {
		c.MaxAge = defaults.MaxAge
	}
	return c
}

// NewCore creates a rotating file core that records entries accepted by levelFilter.
// Zero-valued rotation fields use DefaultConfig's values.
func NewCore(config Config, levelFilter func(zapcore.Level) bool) zapcore.Core {
	config = config.withDefaults()
	writer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
	})

	encoder := zapcore.NewConsoleEncoder(defaultEncoderConfig())
	priority := zap.LevelEnablerFunc(levelFilter)
	return zapcore.NewCore(encoder, writer, priority)
}

func defaultEncoderConfig() zapcore.EncoderConfig {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = timeEncoder
	config.ConsoleSeparator = " "
	return config
}

var chinaZone, _ = time.LoadLocation("Asia/Shanghai")

func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	t = t.In(chinaZone)
	const layout = "2006/01/02 15:04:05"
	type appendTimeEncoder interface {
		AppendTimeLayout(time.Time, string)
	}

	if enc, ok := enc.(appendTimeEncoder); ok {
		enc.AppendTimeLayout(t, layout)
		return
	}
	enc.AppendString(t.Format(layout))
}
