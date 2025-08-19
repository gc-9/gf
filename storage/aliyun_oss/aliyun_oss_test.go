package aliyun_oss

import (
	"bytes"
	"context"
	"os"
	"testing"
)

func TestAliyunOss_Put(t *testing.T) {

	storage, _ := NewAliyunOSS(&AliyunOSSConfig{
		Region:    os.Getenv("OSS_REGION"),
		Bucket:    os.Getenv("OSS_BUCKET"),
		SecretID:  os.Getenv("OSS_SECRET_ID"),
		SecretKey: os.Getenv("OSS_SECRET_KEY"),
	})

	key := "abcdefg.txt"
	put, err := storage.Put(context.Background(), key, bytes.NewReader([]byte("hello world")))
	if err != nil {
		t.Fatalf("put err: %v", err)
	}
	t.Log(put)

	_, err = storage.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}

	_, err = storage.Exist(context.Background(), key)
	if err != nil {
		t.Fatalf("Exist err: %v", err)
	}

	err = storage.Delete(context.Background(), key)
	if err != nil {
		t.Fatalf("delete err: %v", err)
	}

}
