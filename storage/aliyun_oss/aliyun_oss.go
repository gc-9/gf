package aliyun_oss

import (
	"context"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/storage"
	"io"
	"strings"
)

type AliyunOSSConfig struct {
	SecretID  string `yaml:"secretID"`
	SecretKey string `yaml:"secretKey"`
	Bucket    string `yaml:"bucket"`
	Region    string `yaml:"region"`   // "cn-bejing"
	Endpoint  string `yaml:"endpoint"` // "https://cdnxxxx"
}

type AliyunOSSS struct {
	client *oss.Client
	cfg    *AliyunOSSConfig
}

func NewAliyunOSS(cfg *AliyunOSSConfig) (*AliyunOSSS, error) {
	config := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.SecretID, cfg.SecretKey, "")).
		WithRegion(cfg.Region)
	client := oss.NewClient(config)

	cfg.Endpoint = strings.TrimRight(cfg.Endpoint, "/")
	return &AliyunOSSS{
		client: client,
		cfg:    cfg,
	}, nil
}

func (s *AliyunOSSS) Name() string {
	return "aliyun_oss"
}

func (s *AliyunOSSS) Put(ctx context.Context, key string, r io.Reader) (*storage.FileInfo, error) {
	key = strings.TrimLeft(key, "/")

	_, err := s.client.PutObject(ctx, &oss.PutObjectRequest{
		Bucket: oss.Ptr(s.cfg.Bucket),
		Key:    oss.Ptr(key),
		Body:   r,
	})
	if err != nil {
		return nil, errors.Wrap(err, "aliyunOss Put failed")
	}
	return &storage.FileInfo{
		Url:      s.cfg.Endpoint + "/" + key,
		Endpoint: s.cfg.Endpoint,
		Path:     key,
	}, nil
}

func (s *AliyunOSSS) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(s.cfg.Bucket),
		Key:    oss.Ptr(key),
	})
	return errors.Wrap(err, "aliyunOss Delete failed")
}

func (s *AliyunOSSS) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	res, err := s.client.GetObject(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(s.cfg.Bucket),
		Key:    oss.Ptr(key),
	})
	if err != nil {
		return nil, err
	}
	return res.Body, nil
}

func (s *AliyunOSSS) Exist(ctx context.Context, key string) (bool, error) {
	isExist, err := s.client.IsObjectExist(ctx, s.cfg.Bucket, key)
	return isExist, errors.Wrap(err, "tencentCos Exsit failed")
}

func (s *AliyunOSSS) Rename(ctx context.Context, key string, targetKey string) error {
	err := s.Copy(ctx, key, targetKey)
	if err != nil {
		return err
	}
	s.Delete(ctx, key)
	return nil
}

func (s *AliyunOSSS) Copy(ctx context.Context, key string, targetKey string) error {
	key = s.Path(key)
	_, err := s.client.CopyObject(ctx, &oss.CopyObjectRequest{
		Bucket:       oss.Ptr(s.cfg.Bucket),
		Key:          oss.Ptr(targetKey),
		SourceBucket: oss.Ptr(s.cfg.Bucket),
		SourceKey:    oss.Ptr(key),
	})
	return errors.Wrap(err, "aliyunOss CopyObject failed")
}

func (s *AliyunOSSS) Url(key string) string {
	if key == "" {
		return key
	}
	if strings.HasPrefix(key, "http") {
		return key
	}
	return s.cfg.Endpoint + "/" + strings.TrimLeft(key, "/")
}

func (s *AliyunOSSS) Path(url string) string {
	if url == "" {
		return url
	}
	key, _ := strings.CutPrefix(url, s.cfg.Endpoint)
	return strings.TrimLeft(key, "/")
}
