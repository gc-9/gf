package di

import (
	"context"

	"github.com/gc-9/gf/auth"
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/database"
	"github.com/gc-9/gf/i18n"
	storageService "github.com/gc-9/gf/storage/service"
	"github.com/gc-9/gf/telegram"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"xorm.io/xorm"
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

func ProvideDBManager(lc fx.Lifecycle, conf *config.Config) (*database.DBManager, error) {
	confDatabases, err := config.Get[map[string]*config.Database](conf, "databases")
	if err != nil {
		return nil, err
	}

	db := database.NewDBManager(*confDatabases)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return db.Close()
		},
	})
	return db, err
}

func ProvideStorageManager(conf *config.Config) (*storageService.StorageManager, error) {
	confStorages, err := config.Get[map[string]map[string]string](conf, "storages")
	if err != nil {
		return nil, err
	}

	return storageService.NewStorageManager(*confStorages)
}

func NewBot(conf *config.Config) (*telegram.Bot, error) {
	confBot, err := config.Get[telegram.BotConfig](conf, "telegram")
	if err != nil {
		return nil, err
	}

	return telegram.NewBot(confBot), nil
}
