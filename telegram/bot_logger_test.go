package telegram

import (
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestFormatLogEntryUsesOriginalEntry(t *testing.T) {
	message := formatLogEntry(zapcore.Entry{
		Time:    time.Date(2026, time.August, 5, 12, 34, 56, 0, time.UTC),
		Level:   zapcore.ErrorLevel,
		Caller:  zapcore.EntryCaller{Defined: true, File: "/srv/app/logger.go", Line: 42},
		Message: "connection failed",
	}, []zapcore.Field{zap.String("requestID", "abc")})

	for _, want := range []string{"2026/08/05 12:34:56", "ERROR", "logger.go:42", "connection failed", `"requestID":"abc"`} {
		if !strings.Contains(message, want) {
			t.Fatalf("message = %q, want it to contain %q", message, want)
		}
	}
}
