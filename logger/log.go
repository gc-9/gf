package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger
var loggerNoCaller *zap.SugaredLogger
var requestLogger *zap.SugaredLogger
var requestLoggerNoCaller *zap.SugaredLogger

func init() {
	coreConsole := NewConsoleCore()
	lg := zap.New(coreConsole, zap.WithCaller(true))
	logger = lg.Sugar()
	loggerNoCaller = logger.WithOptions(zap.WithCaller(false))
	requestLogger = zap.New(coreConsole, zap.WithCaller(true)).Sugar()
	requestLoggerNoCaller = requestLogger.WithOptions(zap.WithCaller(false))
}

func Logger() *zap.SugaredLogger {
	return logger
}

func NoCaller() *zap.SugaredLogger {
	return loggerNoCaller
}

// RequestNoCaller writes access logs to the non-Loki logger cores.
func RequestNoCaller() *zap.SugaredLogger {
	return requestLoggerNoCaller
}

func newSugaredLogger(cores []zapcore.Core) (*zap.SugaredLogger, *zap.SugaredLogger) {
	lg := zap.New(zapcore.NewTee(cores...), zap.WithCaller(true)).Sugar()
	return lg, lg.WithOptions(zap.WithCaller(false))
}
