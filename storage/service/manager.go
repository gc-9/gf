package service

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/gc-9/gf/storage"
	"github.com/gc-9/gf/storage/aliyun_oss"
	"github.com/gc-9/gf/storage/local"
	"github.com/gc-9/gf/storage/tencent_cos"
	"github.com/gc-9/gf/storage/volcengine_tos"
)

func NewStorage(conf map[string]string) (storage.Storage, error) {
	driver := conf["driver"]
	delete(conf, "driver")

	switch driver {
	case "tencent_cos":
		// map to struct
		var op tencent_cos.TencentCosOptions
		err := mapToStruct(conf, &op)
		if err != nil {
			return nil, err
		}
		return tencent_cos.NewTencentCos(&op)
	case "local":
		var op local.LocalOptions
		err := mapToStruct(conf, &op)
		if err != nil {
			return nil, err
		}
		return local.NewLocal(&op)
	case "aliyun_oss":
		var op aliyun_oss.AliyunOSSConfig
		err := mapToStruct(conf, &op)
		if err != nil {
			return nil, err
		}
		return aliyun_oss.NewAliyunOSS(&op)
	case "volcengine_tos":
		var op volcengine_tos.TosConfig
		err := mapToStruct(conf, &op)
		if err != nil {
			return nil, err
		}
		return volcengine_tos.NewVolcengineTos(&op)

		//case "s3":
		//	buf, _ := json.Marshal(confStorage)
		//	var op storage.S3Config
		//	err := json.Unmarshal(buf, &op)
		//	if err != nil {
		//		return nil, errors.New("storage config error: " + err.Error())
		//	}
		//	return storage.NewAwsS3(&op)
	}

	return nil, errors.New("unknown storage driver")
}

type StorageManager struct {
	Engines map[string]storage.Storage
}

func NewStorageManager(confs map[string]map[string]string) (*StorageManager, error) {
	manager := &StorageManager{
		Engines: make(map[string]storage.Storage),
	}
	var err error
	for name, conf := range confs {
		manager.Engines[name], err = NewStorage(conf)
		if err != nil {
			return nil, err
		}
	}
	return manager, nil
}

func (d *StorageManager) Get(name string) (storage.Storage, bool) {
	engine, ok := d.Engines[name]
	return engine, ok
}

func (d *StorageManager) MustGet(name string) storage.Storage {
	engine, ok := d.Engines[name]
	if !ok {
		log.Panicf("storage %s not found", name)
	}
	return engine
}

func mapToStruct(conf map[string]string, v interface{}) error {
	buf, _ := json.Marshal(conf)
	err := json.Unmarshal(buf, v)
	if err != nil {
		return errors.New("storage config error: " + err.Error())
	}
	return nil
}
