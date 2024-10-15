package cache

import (
	"context"
	"encoding/json"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/logger"
	"github.com/redis/go-redis/v9"
	"time"
)

type fallback[T any] func() (value T, isCache bool, err error)

func CacheFallback[T any](key string, redisClient *redis.Client, funk fallback[T], expiration time.Duration) (T, error) {
	var rs T
	buf, err := redisClient.Get(context.Background(), key).Bytes()
	if err == nil {
		err = json.Unmarshal(buf, &rs)
		if err == nil {
			return rs, err
		} else {
			logger.Logger().Errorf("json.Unmarshal faild '%s', err:%v", string(buf), err)
		}
	} else if err != redis.Nil {
		return rs, errors.Wrap(err, "redis Get failed")
	}

	rs1, useCache, err := funk()
	if err != nil {
		return rs, errors.Wrap(err, "fallback failed")
	}
	if !useCache {
		return rs1, nil
	}

	buf, err = json.Marshal(rs1)
	if err != nil {
		return rs, errors.Wrap(err, "json.Marshal failed")
	}

	_, err = redisClient.Set(context.Background(), key, string(buf), expiration).Result()
	if err != nil {
		return rs, errors.Wrap(err, "redis set failed")
	}

	return rs1, nil
}
