package di

import (
	"context"
	"github.com/gc-9/gf/auth"
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/database"
	"github.com/gc-9/gf/i18n"
	"github.com/gc-9/gf/storage"
	"github.com/gc-9/gf/telegram"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"xorm.io/xorm"

	"encoding/json"
	"errors"
)

func ProvideConfig() *config.Config {
	return config.Parse()
}

func ProvideI18n() (i18n.I18n, error) {
	return config.NewI18n()
}

func ProvideConfigServer(part string) func(conf *config.Config) (*config.Server, error) {
	return func(conf *config.Config) (*config.Server, error) {
		return config.Get[config.Server](conf, part)
	}
}

func ProvideRedis(lc fx.Lifecycle, conf *config.Config) (*redis.Client, error) {
	confRedis, err := config.Get[config.Redis](conf, "redis")
	if err != nil {
		return nil, err
	}

	client := database.NewRedis(confRedis)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return client.Close()
		},
	})
	return client, nil
}

func ProvideCrypto(conf *config.Config) (*auth.EncryptService, error) {
	confCrypto, err := config.Get[config.Crypto](conf, "crypto")
	if err != nil {
		return nil, err
	}

	return auth.NewEncryptService(confCrypto)
}

func ProvideDB(lc fx.Lifecycle, conf *config.Config) (*xorm.Engine, error) {
	confDatabase, err := config.Get[config.Database](conf, "database")
	if err != nil {
		return nil, err
	}

	db := database.NewDB(confDatabase)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return db.Close()
		},
	})
	return db, err
}

func ProvideStorage(conf *config.Config) (storage.Storage, error) {
	confStorageT, err := config.Get[map[string]string](conf, "storage")
	if err != nil {
		return nil, err
	}
	confStorage := *confStorageT

	driver := confStorage["driver"]
	delete(confStorage, "driver")

	switch driver {
	case "tencent_cos":
		// map to struct
		buf, _ := json.Marshal(confStorage)
		var op storage.TencentCosOptions
		err := json.Unmarshal(buf, &op)
		if err != nil {
			return nil, errors.New("storage config error: " + err.Error())
		}
		return storage.NewTencentCos(&op)
	case "local":
		buf, _ := json.Marshal(confStorage)
		var op storage.LocalOptions
		err := json.Unmarshal(buf, &op)
		if err != nil {
			return nil, errors.New("storage config error: " + err.Error())
		}
		return storage.NewLocal(&op)
	case "s3":
		buf, _ := json.Marshal(confStorage)
		var op storage.S3Config
		err := json.Unmarshal(buf, &op)
		if err != nil {
			return nil, errors.New("storage config error: " + err.Error())
		}
		return storage.NewAwsS3(&op)
	}

	return nil, errors.New("unknown storage driver")
}

func NewBot(conf *config.Config) (*telegram.Bot, error) {
	confBot, err := config.Get[telegram.BotConfig](conf, "telegram")
	if err != nil {
		return nil, err
	}

	return telegram.NewBot(confBot), nil
}
