package tencent_cos

import (
	"context"
	"fmt"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/storage"
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
	return strings.TrimLeft(str, "/")
}

func (s *TencentCos) Put(ctx context.Context, key string, r io.Reader) (*storage.FileInfo, error) {
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

	return &storage.FileInfo{
		Url:      s.endpoint + "/" + key,
		Endpoint: s.endpoint,
		Path:     key,
	}, nil
}

func (s *TencentCos) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	key = strings.TrimLeft(key, "/")
	opt := &cos.ObjectGetOptions{}
	res, err := s.client.Object.Get(ctx, key, opt)
	if err != nil {
		return nil, errors.Wrap(err, "tencentCos Get failed")
	}
	return res.Body, nil
}

func (s *TencentCos) Exist(ctx context.Context, key string) (bool, error) {
	key = strings.TrimLeft(key, "/")
	isExist, err := s.client.Object.IsExist(ctx, key)
	return isExist, errors.Wrap(err, "tencentCos Exsits failed")
}

func (s *TencentCos) Rename(ctx context.Context, key, targetKey string) error {
	err := s.Copy(ctx, key, targetKey)
	if err != nil {
		return err
	}
	return s.Delete(ctx, key)
}

func (s *TencentCos) Copy(ctx context.Context, key, targetKey string) error {
	sourceUrl := fmt.Sprintf("%s/%s", s.client.BaseURL.BucketURL.Host, strings.TrimLeft(key, "/"))
	targetKey = strings.TrimLeft(targetKey, "/")
	_, _, err := s.client.Object.Copy(ctx, targetKey, sourceUrl, nil)
	return errors.Wrap(err, "tencentCos Copy failed")
}

func (s *TencentCos) Delete(ctx context.Context, key string) error {
	key = strings.TrimLeft(key, "/")
	_, err := s.client.Object.Delete(ctx, key)
	return errors.Wrap(err, "tencentCos Delete failed")
}
