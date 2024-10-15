package xorm_log

import (
	"github.com/gc-9/gf/logger"
	"go.uber.org/zap"
	"xorm.io/xorm/log"
)

func NewZapLoggerAdapt(l *zap.SugaredLogger) log.Logger {
	return &zapLoggerAdapt{
		l: l.WithOptions(zap.AddCallerSkip(8)),
	}
}

type zapLoggerAdapt struct {
	l       *zap.SugaredLogger
	level   log.LogLevel
	showSQL bool
}

// Error implement ILogger
func (s *zapLoggerAdapt) Error(v ...interface{}) {
	if s.level <= log.LOG_ERR {
		s.l.Errorln(v...)
	}

	logger.Logger()
}

// Errorf implement ILogger
func (s *zapLoggerAdapt) Errorf(format string, v ...interface{}) {
	if s.level <= log.LOG_ERR {
		s.l.Errorf(format, v...)
	}
}

// Debug implement ILogger
func (s *zapLoggerAdapt) Debug(v ...interface{}) {
	if s.level <= log.LOG_DEBUG {
		s.l.Debugln(v...)
	}
}

// Debugf implement ILogger
func (s *zapLoggerAdapt) Debugf(format string, v ...interface{}) {
	if s.level <= log.LOG_DEBUG {
		s.l.Debugf(format, v...)
	}
}

// Info implement ILogger
func (s *zapLoggerAdapt) Info(v ...interface{}) {
	if s.level <= log.LOG_INFO {
		s.l.Infoln(v...)
	}
}

// Infof implement ILogger
func (s *zapLoggerAdapt) Infof(format string, v ...interface{}) {
	if s.level <= log.LOG_INFO {
		s.l.Infof(format, v...)
	}
}

// Warn implement ILogger
func (s *zapLoggerAdapt) Warn(v ...interface{}) {
	if s.level <= log.LOG_WARNING {
		s.l.Warnln(v...)
	}
}

// Warnf implement ILogger
func (s *zapLoggerAdapt) Warnf(format string, v ...interface{}) {
	if s.level <= log.LOG_WARNING {
		s.l.Warnf(format, v...)
	}
}

// Level implement ILogger
func (s *zapLoggerAdapt) Level() log.LogLevel {
	return s.level
}

// SetLevel implement ILogger
func (s *zapLoggerAdapt) SetLevel(l log.LogLevel) {
	s.level = l
}

// ShowSQL implement ILogger
func (s *zapLoggerAdapt) ShowSQL(show ...bool) {
	if len(show) == 0 {
		s.showSQL = true
		return
	}
	s.showSQL = show[0]
}

// IsShowSQL implement ILogger
func (s *zapLoggerAdapt) IsShowSQL() bool {
	return s.showSQL
}
