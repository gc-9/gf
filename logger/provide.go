package logger

import (
	"os"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const zapCoreTags = `group:"zapCores"`

func AsZapCores(core any) any {
	return fx.Annotate(
		core,
		fx.As(new(zapcore.Core)),
		fx.ResultTags(zapCoreTags),
	)
}

func ProvideCores(cores ...any) fx.Option {
	rs := make([]any, len(cores))
	for i, r := range cores {
		rs[i] = AsZapCores(r)
	}
	return fx.Provide(rs...)
}

func InvokeInitLogger() fx.Option {
	return fx.Invoke(fx.Annotate(InitLogger, fx.ParamTags(zapCoreTags)))
}

func InitLogger(cores []zapcore.Core) {
	var tmp []zapcore.Core
	for _, v := range cores {
		if v == nil {
			continue
		}
		tmp = append(tmp, v)
	}
	cores = tmp
	if len(cores) == 0 {
		return
	}

	core := zapcore.NewTee(
		cores...,
	)
	lg := zap.New(core, zap.WithCaller(true))
	logger = lg.Sugar()
	loggerNoCaller = logger.WithOptions(zap.WithCaller(false))
}

func NewConsoleCore() zapcore.Core {
	cfg := defaultConfig()
	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(cfg)
	consoleStdout := zapcore.Lock(os.Stdout)
	consolePriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool { return true })

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleStdout, consolePriority),
	)
	return core
}

func defaultConfig() zapcore.EncoderConfig {
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = encodeTime
	cfg.ConsoleSeparator = " "
	return cfg
}

// encodeTime with timezone
var asZone, _ = time.LoadLocation("Asia/Shanghai")
var timeLayout = "2006/01/02 15:04:05" // as log.Ldate
var encodeTime = timeEncoderOfLayout(timeLayout, asZone)

func timeEncoderOfLayout(layout string, zone *time.Location) zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		t = t.In(zone)
		encodeTimeLayout(t, layout, enc)
	}
}

func encodeTimeLayout(t time.Time, layout string, enc zapcore.PrimitiveArrayEncoder) {
	type appendTimeEncoder interface {
		AppendTimeLayout(time.Time, string)
	}

	if enc, ok := enc.(appendTimeEncoder); ok {
		enc.AppendTimeLayout(t, layout)
		return
	}

	enc.AppendString(t.Format(layout))
}
