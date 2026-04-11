package aws_s3

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/storage"
	"io"
	"mime"
	"path"
	"strings"
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

func (s *AwsS3) Put(ctx context.Context, key string, r io.Reader) (*storage.FileInfo, error) {
	contentType := mime.TypeByExtension(path.Ext(key))
	if _, err := s.uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(s.cfg.Bucket),
		Key:         aws.String(key),
		Body:        r,
		ContentType: aws.String(contentType),
	}); err != nil {
		return nil, err
	}
	return &storage.FileInfo{
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

func (s *AwsS3) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	res, err := s.svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, errors.Wrap(err, "s3 Get failed")
	}
	return res.Body, nil
}

func (s *AwsS3) Exist(ctx context.Context, key string) (bool, error) {
	_, err := s.svc.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Use Aerr to check for 404
		return false, nil
	}
	return true, nil
}

func (s *AwsS3) Size(ctx context.Context, key string) (int64, error) {
	res, err := s.svc.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, errors.Wrap(err, "s3 Size failed")
	}
	if res.ContentLength == nil {
		return 0, nil
	}
	return *res.ContentLength, nil
}

func (s *AwsS3) Rename(ctx context.Context, key string, targetKey string) error {
	err := s.Copy(ctx, key, targetKey)
	if err != nil {
		return err
	}
	return s.Delete(ctx, key)
}

func (s *AwsS3) Copy(ctx context.Context, key string, targetKey string) error {
	_, err := s.svc.CopyObjectWithContext(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s.cfg.Bucket),
		Key:        aws.String(targetKey),
		CopySource: aws.String(s.cfg.Bucket + "/" + key),
	})
	return errors.Wrap(err, "s3 Copy failed")
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
	return strings.TrimLeft(str, "/")
}
