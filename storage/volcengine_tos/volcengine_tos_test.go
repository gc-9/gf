package volcengine_tos

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func TestStoreService_Put(t *testing.T) {
	endpoint := "tos-cn-beijing.volces.com"

	op := &TosConfig{
		Bucket:   os.Getenv("TOS_BUCKET"),
		Endpoint: endpoint,
		//Region:          "cn-beijing",
		CdnUrl:          os.Getenv("TOS_CDN_URL"),
		AccessKeyID:     os.Getenv("TOS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("TOS_SECRET_ACCESS_KEY"),
	}

	s, err := NewVolcengineTos(op)
	if err != nil {
		t.Error(err)
	}

	testContent := "hi"
	key := "test.txt"
	renameKey := "test1.txt"

	_, err = s.Put(context.Background(), key, strings.NewReader(testContent))
	if err != nil {
		t.Fatalf("Put() error:%v", err)
	}

	isExist, err := s.Exist(context.Background(), key)
	if err != nil {
		t.Fatalf("Exist() error:%v", err)
	}
	if !isExist {
		t.Fatalf("Exist() got:%v, want:true", isExist)
	}

	r, err := s.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("Get() error:%v", err)
	}
	all, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error:%v", err)
	}
	if string(all) != testContent {
		t.Fatalf("ReadAll() got:%v, want:%v", string(all), testContent)
	}

	err = s.Rename(context.Background(), key, renameKey)
	if err != nil {
		t.Fatalf("Rename() error:%v", err)
	}

	err = s.Delete(context.Background(), renameKey)
	if err != nil {
		t.Fatalf("Delete() error:%v", err)
	}
}
