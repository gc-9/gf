package storage

import (
	"context"
	"strings"
	"testing"
)

func TestLocal(t *testing.T) {
	baseUrl := "https://127.0.0.1"

	op := &LocalOptions{
		Endpoint: baseUrl,
		Root:     "public/files",
	}
	s, err := NewLocal(op)
	if err != nil {
		t.Error(err)
	}

	key := "test.txt"
	_, err = s.Put(context.Background(), key, strings.NewReader("hi"))
	if err != nil {
		t.Fatalf("Put() error:%v", err)
	}

	err = s.Delete(context.Background(), key)
	if err != nil {
		t.Fatalf("Delete() error:%v", err)
	}
}
