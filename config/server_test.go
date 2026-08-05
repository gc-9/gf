package config

import (
	"errors"
	"testing"
)

func TestRequestLogShouldLog(t *testing.T) {
	requestLog := RequestLog{
		IgnorePaths: []string{`^/healthz$`, `^/devices/[^/]+/status$`},
	}
	if err := requestLog.Compile(); err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	if requestLog.ShouldLog("/healthz", 200, nil) {
		t.Fatal("health check should be ignored")
	}
	if requestLog.ShouldLog("/devices/a-1/status", 200, nil) {
		t.Fatal("polling path should be ignored")
	}
	if !requestLog.ShouldLog("/devices/a-1/status", 500, nil) {
		t.Fatal("5xx response should be recorded")
	}
	if !requestLog.ShouldLog("/healthz", 200, errors.New("request failed")) {
		t.Fatal("request error should be recorded")
	}
}

func TestRequestLogCompileRejectsInvalidPattern(t *testing.T) {
	requestLog := RequestLog{IgnorePaths: []string{"["}}
	if err := requestLog.Compile(); err == nil {
		t.Fatal("Compile() error = nil, want invalid regexp error")
	}
}
