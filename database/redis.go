package database

import (
	"github.com/gc-9/gf/config"
	"github.com/redis/go-redis/v9"
)

func NewRedis(conf *config.Redis) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     conf.Addr,
		Username: conf.Username,
		Password: conf.Password,
		DB:       conf.DB,
	})
	return rdb
}
