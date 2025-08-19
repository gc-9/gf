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
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Exist(ctx context.Context, key string) (bool, error)
	Rename(ctx context.Context, key string, targetKey string) error
	Url(key string) string
	Path(url string) string
}
