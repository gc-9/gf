package storage

import (
	"context"
	"github.com/gc-9/gf/errors"
	"github.com/tencentyun/cos-go-sdk-v5"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TencentCosOptions struct {
	Endpoint  string `json:"endpoint"`
	BucketURL string `json:"bucketUrl"`
	SecretID  string `json:"secretId"`
	SecretKey string `json:"secretKey"`
}

// bucketURL eg:"https://examplebucket-1250000000.cos.ap-guangzhou.myqcloud.com"
// secretID, secretKey 子账号密钥获取可参见 https://cloud.tencent.com/document/product/598/37140
func NewTencentCos(op *TencentCosOptions) (*TencentCos, error) {
	u, err := url.Parse(op.BucketURL)
	if err != nil {
		return nil, err
	}
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Timeout: time.Second * 30,
		Transport: &cos.AuthorizationTransport{
			SecretID:  op.SecretID,
			SecretKey: op.SecretKey,
		},
	})
	return &TencentCos{
		endpoint: strings.TrimRight(op.Endpoint, "/"),
		client:   client,
	}, nil
}

type TencentCos struct {
	endpoint string
	client   *cos.Client
}

func (s *TencentCos) Name() string {
	return "tencent_cos"
}

func (s *TencentCos) Url(key string) string {
	if key == "" {
		return key
	}
	if strings.HasPrefix(key, "http") {
		return key
	}
	return s.endpoint + "/" + strings.TrimLeft(key, "/")
}

func (s *TencentCos) Path(url string) string {
	if url == "" {
		return url
	}
	str, _ := strings.CutPrefix(url, s.endpoint)
	return str
}

func (s *TencentCos) Put(ctx context.Context, key string, r io.Reader) (*FileInfo, error) {
	key = strings.TrimLeft(key, "/")
	opt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			//ContentType: "text/html", // 腾讯云自动设置ContentType, 这里可不设置
		},
	}
	_, err := s.client.Object.Put(ctx, key, r, opt)
	if err != nil {
		return nil, errors.Wrap(err, "tencentCos Put failed")
	}

	return &FileInfo{
		Url:      s.endpoint + "/" + key,
		Endpoint: s.endpoint,
		Path:     key,
	}, nil
}

func (s *TencentCos) Delete(ctx context.Context, key string) error {
	key = strings.TrimLeft(key, "/")
	_, err := s.client.Object.Delete(ctx, key)
	return errors.Wrap(err, "tencentCos Delete failed")
}
