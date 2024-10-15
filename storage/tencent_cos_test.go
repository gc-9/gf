package storage

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestStoreService_Put(t *testing.T) {
	bucketURL := "https://pet-1323272191.cos.ap-beijing.myqcloud.com"

	op := &TencentCosOptions{
		Endpoint:  bucketURL,
		BucketURL: bucketURL,
		SecretID:  os.Getenv("COS_SECRET_ID"),
		SecretKey: os.Getenv("COS_SECRET_KEY"),
	}
	s, err := NewTencentCos(op)
	if err != nil {
		t.Error(err)
	}

	key := "test.text"
	_, err = s.Put(context.Background(), key, strings.NewReader("hi"))
	if err != nil {
		t.Fatalf("Put() error:%v", err)
	}

	err = s.Delete(context.Background(), key)
	if err != nil {
		t.Fatalf("Delete() error:%v", err)
	}
}
