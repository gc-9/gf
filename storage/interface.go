package storage

import (
	"context"
	"io"
)

// StorageService storage interface
type Storage interface {
	Name() string
	Put(ctx context.Context, key string, r io.Reader) (*FileInfo, error)
	Delete(ctx context.Context, key string) error
	Url(key string) string
	Path(url string) string
}
