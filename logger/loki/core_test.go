package loki

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestCoreEncodesZapFields(t *testing.T) {
	var (
		mu       sync.Mutex
		payloads []pushPayload
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var payload pushPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode payload: %v", err)
			return
		}
		mu.Lock()
		payloads = append(payloads, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(&Config{Enabled: true, URL: server.URL, BatchWait: time.Hour})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	core := NewCore(client, Labels{"source": "log"}, zapcore.DebugLevel)
	logger := zap.New(core)
	logger.Info("order created", zap.String("order_id", "o-1"))
	if err := client.Sync(context.Background()); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(payloads) != 1 || len(payloads[0].Streams) != 1 || len(payloads[0].Streams[0].Values) != 1 {
		t.Fatalf("payloads = %#v, want exactly one regular log", payloads)
	}
	var line map[string]any
	if err := json.Unmarshal([]byte(payloads[0].Streams[0].Values[0][1]), &line); err != nil {
		t.Fatalf("decode line: %v", err)
	}
	if line["msg"] != "order created" || line["order_id"] != "o-1" {
		t.Fatalf("line = %#v", line)
	}
}
