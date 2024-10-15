package auth

import (
	"context"
	"encoding/base64"
	"github.com/gc-9/gf/errors"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"strconv"
	"strings"
	"time"
)

func NewAuthService(cachePrefix string, duration time.Duration, redisClient *redis.Client, encryptService *EncryptService) *AuthService {
	return &AuthService{cachePrefix: cachePrefix, duration: duration, redisClient: redisClient, encryptService: encryptService}
}

type AuthService struct {
	cachePrefix    string
	duration       time.Duration
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

func (t *AuthService) MakeLogin(uid int, device string) (string, error) {
	token := strings.Replace(uuid.New().String(), "-", "", -1)
	key := t.getKey(uid, device)
	timeStr := time.Now().Format(time.DateTime)
	_, err := t.redisClient.HMSet(context.Background(), key,
		"token", token,
		"loginAt", timeStr,
		"lastActiveAt", timeStr,
	).Result()
	if err != nil {
		return "", errors.Wrap(err, "redis HMSet failed")
	}
	_, err = t.redisClient.Expire(context.Background(), key, t.getDuration()).Result()
	if err != nil {
		return "", errors.Wrap(err, "redis Expire failed")
	}

	authText := strconv.Itoa(uid) + ":" + device + ":" + token
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
	tokenStore, err := t.redisClient.HGet(context.Background(), key, "token").Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, errors.Wrap(err, "redis HGet failed")
	}
	if tokenStore != token {
		return 0, nil
	}
	_, err = t.redisClient.HSet(context.Background(), key, "lastActiveAt", time.Now().Format(time.DateTime)).Result()
	if err != nil {
		return 0, errors.Wrap(err, "redis HSet failed")
	}
	_, err = t.redisClient.Expire(context.Background(), key, t.getDuration()).Result()
	if err != nil {
		return 0, errors.Wrap(err, "redis Expire failed")
	}

	return uid, err
}

func (t *AuthService) Logout(uid int, device string) error {
	key := t.getKey(uid, device)
	_, err := t.redisClient.Del(context.Background(), key).Result()
	return errors.Wrap(err, "redis Del failed")
}

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
	return t.Logout(uid, device)
}
