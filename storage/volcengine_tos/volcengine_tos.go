package volcengine_tos

import (
	"context"
	"io"
	"mime"
	"path"
	"strings"

	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/storage"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
)

type TosConfig struct {
	AccessKeyID     string `yaml:"accessKeyID"`
	SecretAccessKey string `yaml:"secretAccessKey"`
	Bucket          string `yaml:"bucket"`
	Endpoint        string `yaml:"endpoint"`
	Region          string `yaml:"region"`
	CdnUrl          string `yaml:"cdnUrl"`
}

type VolcengineTos struct {
	client *tos.ClientV2
	cfg    *TosConfig
}

func NewVolcengineTos(cfg *TosConfig) (*VolcengineTos, error) {
	client, err := tos.NewClientV2(
		cfg.Endpoint,
		tos.WithRegion(cfg.Region),
		tos.WithCredentials(tos.NewStaticCredentials(cfg.AccessKeyID, cfg.SecretAccessKey)),
	)
	if err != nil {
		return nil, err
	}
	return &VolcengineTos{
		client: client,
		cfg:    cfg,
	}, nil
}

func (s *VolcengineTos) Name() string {
	return "volcengine_tos"
}

func (s *VolcengineTos) Put(ctx context.Context, key string, r io.Reader) (*storage.FileInfo, error) {
	contentType := mime.TypeByExtension(path.Ext(key))

	putInput := &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket:      s.cfg.Bucket,
			Key:         key,
			ContentType: contentType,
		},
		Content: r,
	}

	_, err := s.client.PutObjectV2(ctx, putInput)
	if err != nil {
		return nil, err
	}

	return &storage.FileInfo{
		Path:     key,
		Endpoint: s.cfg.Endpoint,
		Url:      s.Url(key),
	}, nil
}

func (s *VolcengineTos) Delete(ctx context.Context, key string) error {
	deleteInput := &tos.DeleteObjectV2Input{
		Bucket: s.cfg.Bucket,
		Key:    key,
	}
	_, err := s.client.DeleteObjectV2(ctx, deleteInput)
	return err
}

func (s *VolcengineTos) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	getInput := &tos.GetObjectV2Input{
		Bucket: s.cfg.Bucket,
		Key:    key,
	}

	output, err := s.client.GetObjectV2(ctx, getInput)
	if err != nil {
		return nil, errors.Wrap(err, "tos Get failed")
	}

	return output.Content, nil
}

func (s *VolcengineTos) Exist(ctx context.Context, key string) (bool, error) {
	headInput := &tos.HeadObjectV2Input{
		Bucket: s.cfg.Bucket,
		Key:    key,
	}

	_, err := s.client.HeadObjectV2(ctx, headInput)
	if err != nil {
		if e, ok := err.(*tos.TosServerError); ok && e.StatusCode == 404 {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s *VolcengineTos) Rename(ctx context.Context, key string, targetKey string) error {
	err := s.Copy(ctx, key, targetKey)
	if err != nil {
		return err
	}
	return s.Delete(ctx, key)
}

func (s *VolcengineTos) Copy(ctx context.Context, key string, targetKey string) error {
	copyInput := &tos.CopyObjectInput{
		Bucket:    s.cfg.Bucket,
		Key:       targetKey,
		SrcBucket: s.cfg.Bucket,
		SrcKey:    key,
	}
	_, err := s.client.CopyObject(ctx, copyInput)
	return errors.Wrap(err, "tos CopyObject failed")
}

func (s *VolcengineTos) Url(key string) string {
	if key == "" {
		return key
	}
	if strings.HasPrefix(key, "http") {
		return key
	}
	return s.cfg.CdnUrl + "/" + strings.TrimLeft(key, "/")
}

func (s *VolcengineTos) Path(url string) string {
	if url == "" {
		return url
	}
	str, _ := strings.CutPrefix(url, s.cfg.CdnUrl)
	return strings.TrimLeft(str, "/")
}
