package auth

import (
	"context"
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/util"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func NewAuthService(config config.AuthConfig, redisClient *redis.Client, encryptService *EncryptService) *AuthService {
	if config.CachePrefix == "" {
		config.CachePrefix = "auth_admin"
	}
	if config.Duration == 0 {
		config.Duration = time.Hour * 24
	}
	return &AuthService{
		cachePrefix:    config.CachePrefix,
		duration:       config.Duration,
		maxDevices:     config.MaxDevices,
		redisClient:    redisClient,
		encryptService: encryptService,
	}
}

type AuthService struct {
	cachePrefix    string
	duration       time.Duration
	maxDevices     int
	redisClient    *redis.Client
	encryptService *EncryptService
}

func (t *AuthService) getKey(uid int, device string) string {
	return t.cachePrefix + ":" + device + ":" + strconv.Itoa(uid)
}

func (t *AuthService) getDuration() time.Duration {
	return t.duration
}

func (t *AuthService) encryptText(text string) string {
	buf := t.encryptService.Encrypt([]byte(text))
	return base64.StdEncoding.EncodeToString(buf)
}

func (t *AuthService) decryptText(text string) string {
	buf, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return ""
	}
	buf, _ = t.encryptService.Decrypt(buf)
	return string(buf)
}

func (t *AuthService) MakeLogin(uid int, deviceInfo string) (string, error) {
	deviceId := strings.Replace(uuid.New().String(), "-", "", -1)
	token := strings.Replace(uuid.New().String(), "-", "", -1)
	key := t.getKey(uid, deviceId)
	now := time.Now()
	timeStr := now.Format(time.DateTime)

	ctx := context.Background()
	pipe := t.redisClient.Pipeline()
	pipe.HMSet(ctx, key,
		"token", token,
		"loginAt", timeStr,
		"lastActiveAt", timeStr,
		"deviceInfo", util.Substring(deviceInfo, 0, 200),
	)
	pipe.Expire(ctx, key, t.getDuration())

	zsetKey := t.cachePrefix + ":devices:" + strconv.Itoa(uid)
	pipe.ZAdd(ctx, zsetKey, redis.Z{
		Score:  float64(now.UnixMilli()),
		Member: key,
	})
	pipe.Expire(ctx, zsetKey, t.getDuration())

	_, err := pipe.Exec(ctx)
	if err != nil {
		return "", errors.Wrap(err, "redis HMSet failed")
	}

	maxDev := t.maxDevices
	if maxDev <= 0 {
		maxDev = 1
	}

	count, err := t.redisClient.ZCard(ctx, zsetKey).Result()
	if err == nil && count > int64(maxDev) {
		remCount := count - int64(maxDev)
		oldDevices, _ := t.redisClient.ZRange(ctx, zsetKey, 0, remCount-1).Result()
		if len(oldDevices) > 0 {
			var remMembers []interface{}
			for _, od := range oldDevices {
				remMembers = append(remMembers, od)
			}

			delPipe := t.redisClient.Pipeline()
			delPipe.Del(ctx, oldDevices...)
			delPipe.ZRem(ctx, zsetKey, remMembers...)
			_, _ = delPipe.Exec(ctx)
		}
	}

	authText := strconv.Itoa(uid) + ":" + deviceId + ":" + token
	return t.encryptText(authText), err
}

func (t *AuthService) CheckToken(tokenStr string) (int, error) {
	if len(tokenStr) < 10 {
		return 0, nil
	}
	tokenStr = t.decryptText(tokenStr)
	tmp := strings.Split(tokenStr, ":")
	if len(tmp) != 3 {
		return 0, nil
	}
	uidStr, device, token := tmp[0], tmp[1], tmp[2]
	uid, err := strconv.Atoi(uidStr)
	if err != nil {
		return 0, nil
	}

	key := t.getKey(uid, device)
	ctx := context.Background()
	tokenStore, err := t.redisClient.HGet(ctx, key, "token").Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, errors.Wrap(err, "redis HGet failed")
	}
	if tokenStore != token {
		return 0, nil
	}

	now := time.Now()
	pipe := t.redisClient.Pipeline()
	pipe.HSet(ctx, key, "lastActiveAt", now.Format(time.DateTime))
	pipe.Expire(ctx, key, t.getDuration())

	zsetKey := t.cachePrefix + ":devices:" + strconv.Itoa(uid)
	pipe.ZAdd(ctx, zsetKey, redis.Z{
		Score:  float64(now.UnixMilli()),
		Member: key,
	})
	pipe.Expire(ctx, zsetKey, t.getDuration())

	_, err = pipe.Exec(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "redis pipeline exec failed")
	}

	return uid, nil
}

/* func (t *AuthService) Logout(uid int, device string) error {
	device = t.hashDevice(device)
	key := t.getKey(uid, device)
	zsetKey := t.cachePrefix + ":devices:" + strconv.Itoa(uid)

	ctx := context.Background()
	pipe := t.redisClient.Pipeline()
	pipe.Del(ctx, key)
	pipe.ZRem(ctx, zsetKey, device)
	_, err := pipe.Exec(ctx)

	return errors.Wrap(err, "redis Del failed")
} */

func (t *AuthService) LogoutByToken(tokenStr string) error {
	if len(tokenStr) < 10 {
		return nil
	}
	tokenStr = t.decryptText(tokenStr)
	tmp := strings.Split(tokenStr, ":")
	if len(tmp) != 3 {
		return nil
	}
	uidStr, device, _ := tmp[0], tmp[1], tmp[2]
	uid, err := strconv.Atoi(uidStr)
	if err != nil {
		return nil
	}

	key := t.getKey(uid, device)
	zsetKey := t.cachePrefix + ":devices:" + strconv.Itoa(uid)

	ctx := context.Background()
	pipe := t.redisClient.Pipeline()
	pipe.Del(ctx, key)
	pipe.ZRem(ctx, zsetKey, key)
	_, err = pipe.Exec(ctx)

	return errors.Wrap(err, "redis Del failed")
}
