package storage

import (
	"context"
	"io"
	"mime"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3Config struct {
	AccessKey string `yaml:"accessKey"`
	SecretKey string `yaml:"secretKey"`
	Bucket    string `yaml:"bucket"`
	Endpoint  string `yaml:"endpoint"`
	Region    string `yaml:"region"`
}

type AwsS3 struct {
	svc      *s3.S3
	cfg      *S3Config
	uploader *s3manager.Uploader
}

func NewAwsS3(cfg *S3Config) (*AwsS3, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(cfg.Region),
		Credentials: credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey, ""),
	})
	if err != nil {
		return nil, err
	}
	svc := s3.New(sess)
	return &AwsS3{
		svc:      svc,
		cfg:      cfg,
		uploader: s3manager.NewUploader(sess),
	}, nil
}

func (s *AwsS3) Name() string {
	return "aws_s3"
}

func (s *AwsS3) Put(ctx context.Context, key string, r io.Reader) (*FileInfo, error) {
	contentType := mime.TypeByExtension(path.Ext(key))
	if _, err := s.uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(s.cfg.Bucket),
		Key:         aws.String(key),
		Body:        r,
		ContentType: aws.String(contentType),
	}); err != nil {
		return nil, err
	}
	return &FileInfo{
		Path:     key,
		Endpoint: s.cfg.Endpoint,
		Url:      s.Url(key),
	}, nil
}

func (s *AwsS3) Delete(ctx context.Context, key string) error {
	_, err := s.svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	return err
}

func (s *AwsS3) Url(key string) string {
	if key == "" {
		return key
	}
	if strings.HasPrefix(key, "http") {
		return key
	}
	return s.cfg.Endpoint + "/" + strings.TrimLeft(key, "/")
}

func (s *AwsS3) Path(url string) string {
	if url == "" {
		return url
	}
	str, _ := strings.CutPrefix(url, s.cfg.Endpoint)
	return str
}
