package auth_services

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"time"

	"github.com/gc-9/gf/auth"
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/crud"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/httplib"
	"github.com/gc-9/gf/logger"
	adminTypes "github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/types"
	"github.com/redis/go-redis/v9"
	"xorm.io/builder"
	"xorm.io/xorm"
)

func NewAdminService(db *xorm.Engine, authService *auth.AuthService,
	redisClient *redis.Client, servConf *config.Server) *AdminService {
	return &AdminService{
		CrudDB:      crud.NewCrudDB[adminTypes.Admin](db),
		db:          db,
		authService: authService,
		redisClient: redisClient,
		servConf:    servConf,
	}
}

type AdminService struct {
	*crud.CrudDB[adminTypes.Admin]
	db          *xorm.Engine
	authService *auth.AuthService
	redisClient *redis.Client
	servConf    *config.Server
}

func (t *AdminService) QueryWithRoleIdOption(query *xorm.Session) {
	query.Table("admin").Alias("a").Select("a.*, r.id role_id, r.`key` role_key").
		Join("LEFT", []string{"auth_admin_role", "rr"}, "rr.uid=a.id").
		Join("LEFT", []string{"auth_role", "r"}, "rr.rid=r.id")
}

func (t *AdminService) ListAdminRole(param *types.ParamPageQuery) (*types.PagerData[*adminTypes.Admin_R], error) {
	pagerData, err := crud.PagerData[adminTypes.Admin_R](t.db, param.ParamPager, t.QueryWithRoleIdOption, func(query *xorm.Session) {
		for k, f := range param.Filters {
			switch k {
			case "name", "username":
				query.Where(builder.Like{"a." + k, param.Filters.ValueString(k)})
			default:
				query.Where(builder.Eq{"a." + k: f})
			}
		}
		query.OrderBy("uid desc")
	})

	if err != nil {
		return nil, err
	}

	roleMap, err := t.RoleMap()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	for _, v := range pagerData.List {
		v.Role, _ = roleMap[v.RoleId]
		if v.LoginUnlockAt != nil && v.LoginUnlockAt.After(now) {
			v.LoginLocked = true
			v.LoginWaitSeconds = int64(math.Ceil(v.LoginUnlockAt.Sub(now).Seconds()))
		}
	}
	return pagerData, nil
}

func (t *AdminService) GetByUserName(username string) (*adminTypes.Admin, error) {
	return crud.GetByOptions[adminTypes.Admin](t.db, func(query *xorm.Session) {
		query.Where("username = ?", username)
	})
}

func (t *AdminService) CreateAdmin(roleId int, admin *adminTypes.Admin) (*adminTypes.Admin, error) {
	tx := t.db.NewSession()
	defer tx.Close()

	err := tx.Begin()
	if err != nil {
		return nil, errors.PublicWrap(err, "dbError")
	}

	exist, err := tx.Where("username=?", admin.Username).Exist(&adminTypes.Admin{})
	if err != nil {
		return nil, errors.PublicWrap(err, "dbError")
	}
	if exist {
		return nil, errors.Public("账号已存在")
	}

	_, err = tx.Insert(admin)
	if err != nil {
		return nil, errors.PublicWrap(err, "dbError")
	}

	role := &adminTypes.AuthAdminRole{Rid: roleId, Uid: admin.ID}
	_, err = tx.Insert(role)
	if err != nil {
		return nil, errors.PublicWrap(err, "dbError")
	}

	err = tx.Commit()
	if err != nil {
		return nil, errors.PublicWrap(err, "dbError")
	}
	return admin, err
}

func loginFailWaitTime(failTimes int64) time.Duration {
	if failTimes < 3 {
		return 0
	}
	duration := 30 * time.Second
	for i := int64(3); i < failTimes && duration < 5*time.Minute; i++ {
		duration = min(duration*2, 5*time.Minute)
	}
	return duration
}

// RecordLoginFailure 在同一事务中累加次数并设置解锁时间，日志不参与状态维护。
func (t *AdminService) RecordLoginFailure(uid int) (wait, nextWait time.Duration, err error) {
	tx := t.db.NewSession()
	defer tx.Close()
	if err = tx.Begin(); err != nil {
		return 0, 0, errors.PublicWrap(err, "dbError")
	}
	defer tx.Rollback()

	// 更新先取得行锁，避免并发失败丢失计数；等待期间不重复计数。
	now := time.Now()
	affected, err := tx.ID(uid).
		Where("login_unlock_at IS NULL OR login_unlock_at <= ?", now.In(t.db.DatabaseTZ).Format(time.DateTime)).
		Incr("login_fail_times", 1).Update(&adminTypes.Admin{})
	if err != nil {
		return 0, 0, errors.PublicWrap(err, "dbError")
	}
	var admin adminTypes.Admin
	has, err := tx.ID(uid).Get(&admin)
	if err != nil {
		return 0, 0, errors.PublicWrap(err, "dbError")
	}
	if !has {
		return 0, 0, errors.Public("用户不存在")
	}
	nextWait = loginFailWaitTime(admin.LoginFailTimes + 1)
	if affected == 0 {
		if admin.LoginUnlockAt != nil {
			wait = max(0, time.Until(*admin.LoginUnlockAt))
		}
		return wait, nextWait, nil
	}
	wait = loginFailWaitTime(admin.LoginFailTimes)
	if wait > 0 {
		unlockAt := time.Now().Add(wait)
		_, err = tx.ID(uid).Cols("login_unlock_at").Update(&adminTypes.Admin{LoginUnlockAt: &unlockAt})
		if err != nil {
			return 0, 0, errors.PublicWrap(err, "dbError")
		}
	}
	return wait, nextWait, errors.PublicWrap(tx.Commit(), "dbError")
}

// UnlockLogin 手动清除登录锁定，不删除或查询日志。
func (t *AdminService) UnlockLogin(uid int) error {
	has, err := t.Exist(uid)
	if err != nil {
		return err
	}
	if !has {
		return errors.Public("用户不存在")
	}
	_, err = t.db.Table(&adminTypes.Admin{}).ID(uid).Update(map[string]interface{}{
		"login_fail_times": 0,
		"login_unlock_at":  nil,
	})
	return errors.PublicWrap(err, "dbError")
}

func (t *AdminService) CreateLoginFailLog(uid int, ip string, userAgent string) error {
	_, err := t.db.Insert(&adminTypes.AdminLog{
		Uid:       uid,
		Rid:       0,
		Action:    "login_fail",
		Ip:        ip,
		UserAgent: userAgent,
	})
	return errors.PublicWrap(err, "dbError")
}

func (t *AdminService) CreateLoginLog(uid int, rid int, ip string, userAgent string) error {
	_, err := t.db.Insert(&adminTypes.AdminLog{
		Uid:       uid,
		Rid:       rid,
		Method:    "POST",
		Action:    "login",
		Ip:        ip,
		UserAgent: userAgent,
		Remark:    "登录",
	})
	return errors.PublicWrap(err, "dbError")
}

func (t *AdminService) Roles() ([]*adminTypes.AuthRole, error) {
	var roles []*adminTypes.AuthRole
	err := t.db.Cols("id", "name").OrderBy("id asc").Find(&roles)
	if err != nil {
		return nil, errors.PublicWrap(err, "dbError")
	}
	return roles, nil
}

func (t *AdminService) RoleMap() (map[int]*adminTypes.AuthRole, error) {
	roles, err := t.Roles()
	if err != nil {
		return nil, err
	}
	var roleMap = make(map[int]*adminTypes.AuthRole)
	for _, role := range roles {
		roleMap[role.ID] = role
	}
	return roleMap, nil
}

func (t *AdminService) GetRole(uid int) (*adminTypes.AuthRole, error) {
	return crud.GetByOptions[adminTypes.AuthRole](t.db, func(query *xorm.Session) {
		query.Where("id in (select rid from auth_admin_role where uid=?)", uid)
	})
}

func (t *AdminService) UpdateRole(uid int, roleId int) error {
	_, err := t.db.Exec("update `auth_admin_role` set rid=? where uid=?", roleId, uid)
	return errors.PublicWrap(err, "dbError")
}

func (t *AdminService) GetPermissions(roleId int) ([]*adminTypes.AuthPermission, error) {
	return crud.List[adminTypes.AuthPermission](t.db, func(query *xorm.Session) {
		query.Where("id in (select pid from auth_role_permission where rid=?)", roleId)
	})
}

func (t *AdminService) IsHasPermission(roleId int, roleKey, method, path string) (bool, error) {
	if roleId <= 0 {
		return false, nil
	}
	if roleKey == t.servConf.Acl.SuperRoleKey {
		return true, nil
	}
	has, err := t.db.Where("pid in (select id from auth_permission where method=? and path=?)", method, path).
		And("rid=? ", roleId).Exist(&adminTypes.RolePermission{})
	return has, errors.PublicWrap(err, "dbError")
}

func (t *AdminService) IsAdminHasPermission(admin *adminTypes.Admin_RoleId, method, path string) (bool, error) {
	if admin.RoleId <= 0 {
		return false, nil
	}
	if admin.RoleKey == t.servConf.Acl.SuperRoleKey {
		return true, nil
	}
	has, err := t.db.Where("pid in (select id from auth_permission where method=? and path=?)", method, path).
		And("rid=? ", admin.RoleId).Exist(&adminTypes.RolePermission{})
	return has, errors.PublicWrap(err, "dbError")
}

func (t *AdminService) GetAdminWithRoleId(uid int) (*adminTypes.Admin_RoleId, error) {
	admin, err := crud.GetByOptions[adminTypes.Admin_RoleId](t.db, t.QueryWithRoleIdOption, func(query *xorm.Session) {
		query.ID(uid)
	})
	if err != nil {
		return nil, err
	}
	admin.IsSuper = admin.RoleKey == t.servConf.Acl.SuperRoleKey
	return admin, nil
}

func (t *AdminService) GetAdminRolePermissions(uid int) (*adminTypes.Admin_R, error) {
	adminRole, err := crud.GetByOptions[adminTypes.Admin_R](t.db, t.QueryWithRoleIdOption, func(query *xorm.Session) {
		query.ID(uid)
	})
	if adminRole == nil || err != nil {
		return adminRole, err
	}
	role, err := t.GetRole(uid)
	if err != nil {
		return nil, err
	}
	adminRole.Role = role

	if adminRole.RoleKey != t.servConf.Acl.SuperRoleKey {
		permissions, err := t.GetPermissions(adminRole.RoleId)
		if err != nil {
			return nil, err
		}
		adminRole.Permissions = permissions
	}
	return adminRole, nil
}

func (t *AdminService) MakeLogin(uid int, device string) (string, error) {
	// check account
	admin, err := t.GetAdminWithRoleId(uid)
	if err != nil {
		return "", err
	}
	if admin == nil {
		return "", errors.Public("账号不存在")
	}
	if admin.RoleId == 0 {
		return "", errors.Public("该账号没有角色，请联系管理员")
	}
	return t.authService.MakeLogin(admin.ID, device)
}

func (t *AdminService) CheckToken(tokenStr string) (*adminTypes.Admin_RoleId, error) {
	uid, err := t.authService.CheckToken(tokenStr)
	if err != nil || uid <= 0 {
		return nil, err
	}

	key := "admin:loginState:" + strconv.Itoa(uid)
	adminRoleId, err := getFallback(t.redisClient, key, func() (*adminTypes.Admin_RoleId, error) {
		return t.GetAdminWithRoleId(uid)
	}, time.Second*10)
	if err != nil {
		return nil, err
	}

	// unusual logout
	if adminRoleId == nil || adminRoleId.RoleId == 0 || adminRoleId.Status <= 0 {
		_ = t.authService.LogoutByToken(tokenStr)
		return nil, nil
	}
	return adminRoleId, err
}

func getFallback[T any](client *redis.Client, key string, fk func() (*T, error), duration time.Duration) (*T, error) {
	result, err := client.Get(context.Background(), key).Result()
	if err != nil && err != redis.Nil {
		return nil, errors.Wrap(err, "redis Get failed")
	}

	if result != "" {
		var t2 T
		err = json.Unmarshal([]byte(result), &t2)
		if err != nil {
			logger.Logger().Warn("json.Unmarshal failed", err)
		} else {
			return &t2, nil
		}
	}

	var t *T
	var buf []byte
	t, err = fk()
	if err != nil {
		return nil, err
	}
	buf, err = json.Marshal(t)
	if err != nil {
		return nil, errors.Wrap(err, "json Marshal failed")
	}
	_, err = client.Set(context.Background(), key, string(buf), duration).Result()
	return t, errors.Wrap(err, "redis Set failed")
}

func (t *AdminService) Logout(ctx httplib.RequestContext) error {
	tokenStr := ctx.Request().Header.Get(t.servConf.Acl.AuthHeader)
	if tokenStr == "" {
		return nil
	}
	return t.authService.LogoutByToken(tokenStr)
}
