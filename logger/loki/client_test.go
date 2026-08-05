package loki

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestClientBatchesEntriesByLabels(t *testing.T) {
	var (
		mu       sync.Mutex
		payloads []pushPayload
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var payload pushPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		payloads = append(payloads, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(&Config{
		Enabled:       true,
		URL:           server.URL,
		QueueCapacity: 10,
		BatchSize:     100,
		BatchWait:     time.Hour,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	labels := Labels{"env": "test", "app": "gf", "source": "request"}
	if !client.Push(labels, time.Unix(0, 1), []byte(`{"message":"one"}`)) {
		t.Fatal("first Push() = false")
	}
	if !client.Push(labels, time.Unix(0, 2), []byte(`{"message":"two"}`)) {
		t.Fatal("second Push() = false")
	}
	if err := client.Sync(context.Background()); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(payloads) != 1 {
		t.Fatalf("push request count = %d, want 1", len(payloads))
	}
	if len(payloads[0].Streams) != 1 {
		t.Fatalf("stream count = %d, want 1", len(payloads[0].Streams))
	}
	stream := payloads[0].Streams[0]
	if got := len(stream.Values); got != 2 {
		t.Fatalf("entry count = %d, want 2", got)
	}
	if stream.Stream["source"] != "request" {
		t.Fatalf("source label = %q, want request", stream.Stream["source"])
	}
}
