package auth_services

import (
	"context"
	"fmt"
	"math"
	"net"
	"time"

	"github.com/gc-9/gf/errors"
	adminTypes "github.com/gc-9/gf/mod/admin/types"
	"github.com/redis/go-redis/v9"
)

const (
	loginIPFailThreshold = 20
	loginIPLockDuration  = 10 * time.Minute
	loginIPFailureTTL    = 30 * time.Minute
)

// Redis 原子维护 IP 失败次数和解锁时间。等待期间不累加、不延长等待。
var loginIPLimitScript = redis.NewScript(`
local threshold = tonumber(ARGV[2])
local lockDuration = tonumber(ARGV[3])
local failureTTL = tonumber(ARGV[4])
local clock = redis.call('TIME')
local now = tonumber(clock[1]) * 1000 + math.floor(tonumber(clock[2]) / 1000)
local count = tonumber(redis.call('HGET', KEYS[1], 'count') or '0')
local unlockAt = tonumber(redis.call('HGET', KEYS[1], 'unlockAt') or '0')
if unlockAt > now then
    return {unlockAt - now, 0}
end
if ARGV[1] == '0' then return {0, 0} end
count = count + 1
local wait = count >= threshold and lockDuration or 0
redis.call('HSET', KEYS[1], 'count', count, 'unlockAt', now + wait)
redis.call('PEXPIRE', KEYS[1], failureTTL)
return {wait, wait > 0 and 1 or 0}
`)

// LoginIPLimit 检查 IP 是否受限；failed 为 true 时累加失败。newlyLocked 仅在本次触发锁定时为 true。
func (t *AdminService) LoginIPLimit(ctx context.Context, ip string, failed bool) (wait time.Duration, newlyLocked bool, err error) {
	address := net.ParseIP(ip)
	if address == nil {
		return 0, false, errors.Public("IP地址无效")
	}
	ip = address.String()
	mode := 0
	if failed {
		mode = 1
	}
	key := t.servConf.Auth.CachePrefix + "admin:loginIP:" + ip
	values, err := loginIPLimitScript.Run(ctx, t.redisClient, []string{key},
		mode, loginIPFailThreshold, loginIPLockDuration.Milliseconds(), loginIPFailureTTL.Milliseconds()).Int64Slice()
	if err != nil {
		return 0, false, errors.PublicWrap(err, "操作失败，请稍后重试")
	}
	return time.Duration(values[0]) * time.Millisecond, values[1] == 1, nil
}

// CreateLoginIPLockLog 记录触发锁定的 IP；已锁定期间的请求不重复记录。
func (t *AdminService) CreateLoginIPLockLog(ip, userAgent string, wait time.Duration) error {
	seconds := int64(math.Ceil(wait.Seconds()))
	_, err := t.db.Insert(&adminTypes.AdminLog{
		Method:    "POST",
		Action:    "login_ip_lock",
		Ip:        net.ParseIP(ip).String(),
		UserAgent: userAgent,
		Data:      fmt.Sprintf(`{"waitSeconds":%d}`, seconds),
		Remark:    fmt.Sprintf("登录IP锁定，等待%d秒", seconds),
	})
	return errors.PublicWrap(err, "dbError")
}

// UnlockLoginIP 删除该 IP 的失败次数和锁定时间，账号锁定独立处理。
func (t *AdminService) UnlockLoginIP(ctx context.Context, ip string) error {
	address := net.ParseIP(ip)
	if address == nil {
		return errors.Public("IP地址无效")
	}
	key := t.servConf.Auth.CachePrefix + "admin:loginIP:" + address.String()
	return errors.PublicWrap(t.redisClient.Del(ctx, key).Err(), "操作失败，请稍后重试")
}
