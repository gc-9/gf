package logger

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gc-9/gf/logger/loki"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestInitLoggerKeepsLokiCoreOutOfRequestLogger(t *testing.T) {
	var (
		mu       sync.Mutex
		payloads []map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode push payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		payloads = append(payloads, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := loki.NewClient(&loki.Config{Enabled: true, URL: server.URL, BatchWait: time.Hour})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	oldLogger, oldNoCaller := logger, loggerNoCaller
	oldRequestLogger, oldRequestNoCaller := requestLogger, requestLoggerNoCaller
	defer func() {
		logger, loggerNoCaller = oldLogger, oldNoCaller
		requestLogger, requestLoggerNoCaller = oldRequestLogger, oldRequestNoCaller
	}()

	localCore, observed := observer.New(zapcore.DebugLevel)
	InitLogger([]zapcore.Core{
		localCore,
		loki.NewCore(client, loki.Labels{"source": "log"}, zapcore.DebugLevel),
	})

	Logger().Info("regular log")
	RequestNoCaller().Info("request log")
	if err := client.Sync(context.Background()); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if got := observed.Len(); got != 2 {
		t.Fatalf("local log count = %d, want 2", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(payloads) != 1 {
		t.Fatalf("Loki push count = %d, want 1", len(payloads))
	}
	streams, ok := payloads[0]["streams"].([]any)
	if !ok || len(streams) != 1 {
		t.Fatalf("streams = %#v, want one stream", payloads[0]["streams"])
	}
	values := streams[0].(map[string]any)["values"].([]any)
	if len(values) != 1 {
		t.Fatalf("Loki entry count = %d, want 1 regular entry", len(values))
	}
}
