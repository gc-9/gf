package storage

import (
	"context"
	"github.com/gc-9/gf/errors"
	"io"
	"os"
	"path"
	"strings"
)

type LocalOptions struct {
	Root     string `json:"root"`
	Endpoint string `json:"endpoint"`
}

func NewLocal(op *LocalOptions) (*Local, error) {
	root := path.Dir(op.Root + "/")
	err := os.MkdirAll(root, 0644)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(op.Endpoint, "/")
	return &Local{
		root:     root,
		endpoint: endpoint,
	}, nil
}

type Local struct {
	root     string
	endpoint string
}

func (s *Local) Name() string {
	return "local"
}

func (s *Local) Url(key string) string {
	if key == "" {
		return key
	}
	if strings.HasPrefix(key, "http") {
		return key
	}
	return s.endpoint + "/" + strings.TrimLeft(key, "/")
}

func (s *Local) Path(url string) string {
	if url == "" {
		return url
	}
	p, found := strings.CutPrefix(url, s.endpoint)
	if found {
		return strings.TrimLeft(p, "/")
	}
	return url
}

func (s *Local) Put(ctx context.Context, key string, r io.Reader) (*FileInfo, error) {
	key = strings.TrimLeft(key, "/")
	fp := s.root + "/" + key

	err := os.MkdirAll(path.Dir(fp), 0644)
	if err != nil {
		return nil, errors.Wrap(err, "local MkdirAll failed")
	}

	file, err := os.OpenFile(fp, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "local OpenFile failed")
	}
	defer file.Close()

	_, err = io.Copy(file, r)

	return &FileInfo{
		Url:  s.endpoint + "/" + key,
		Path: key,
	}, nil
}

func (s *Local) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	f := s.root + "/" + strings.TrimLeft(key, "/")
	return os.Open(f)
}

func (s *Local) Exist(ctx context.Context, key string) (bool, error) {
	f := s.root + "/" + strings.TrimLeft(key, "/")
	_, err := os.Stat(f)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, errors.Wrap(err, "local Exsit failed")
}

func (s *Local) Rename(ctx context.Context, key string, targetKey string) error {
	sf := s.root + "/" + strings.TrimLeft(key, "/")
	tf := s.root + "/" + strings.TrimLeft(targetKey, "/")
	return errors.Wrap(os.Rename(sf, tf), "local rename failed")
}

func (s *Local) Delete(ctx context.Context, key string) error {
	f := s.root + strings.TrimLeft(key, "/")

	err := os.Remove(f)
	if err != nil && err != os.ErrNotExist {
		return errors.Wrap(err, "local Remove failed")
	}
	return nil
}
