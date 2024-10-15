package logger

import (
	"go.uber.org/zap"
)

var logger *zap.SugaredLogger
var loggerNoCaller *zap.SugaredLogger

func init() {
	coreConsole := NewConsoleCore()
	lg := zap.New(coreConsole, zap.WithCaller(true))
	logger = lg.Sugar()
	loggerNoCaller = logger.WithOptions(zap.WithCaller(false))
}

func Logger() *zap.SugaredLogger {
	return logger
}

func NoCaller() *zap.SugaredLogger {
	return loggerNoCaller
}
