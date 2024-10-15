package auth_services

import (
	"context"
	"encoding/json"
	"github.com/gc-9/gf/auth"
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/crud"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/logger"
	adminTypes "github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/types"
	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
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

	for _, v := range pagerData.List {
		v.Role, _ = roleMap[v.RoleId]
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
		return nil, errors.Wrap(err, "error")
	}

	exist, err := tx.Where("username=?", admin.Username).Exist(&adminTypes.Admin{})
	if err != nil {
		return nil, errors.Wrap(err, "error")
	}
	if exist {
		return nil, errors.New("账号已存在")
	}

	_, err = tx.Insert(admin)
	if err != nil {
		return nil, errors.Wrap(err, "error")
	}

	role := &adminTypes.AuthAdminRole{Rid: roleId, Uid: admin.ID}
	_, err = tx.Insert(role)
	if err != nil {
		return nil, errors.Wrap(err, "error")
	}

	err = tx.Commit()
	if err != nil {
		return nil, errors.Wrap(err, "error")
	}
	return admin, err
}

func (t *AdminService) CheckLoginFailTimes(uid int) (*time.Duration, error) {
	duration := 10 * time.Minute
	maxFailTimes := int64(5)

	c, err := t.db.Where("uid=? AND action=? AND created_at >= ?", uid, "login_fail", time.Now().Add(-duration)).Count(&adminTypes.AdminLog{})
	if err != nil {
		return nil, err
	}

	if c > maxFailTimes {
		var lastOne adminTypes.AdminLog
		has, err := t.db.Where("uid=? AND action=? AND created_at >= ?", uid, "login_fail", time.Now().Add(-duration)).
			OrderBy("id DESC").Get(&lastOne)
		if err != nil {
			return nil, err
		}
		if !has {
			return nil, nil
		}
		diff := duration - time.Now().Sub(lastOne.CreatedAt)
		return &diff, nil
	}
	return nil, nil
}

func (t *AdminService) CreateLoginFailLog(uid int, ip string, userAgent string) error {
	_, err := t.db.Insert(&adminTypes.AdminLog{
		Uid:       uid,
		Rid:       0,
		Action:    "login_fail",
		Ip:        ip,
		UserAgent: userAgent,
	})
	return err
}

func (t *AdminService) Roles() ([]*adminTypes.AuthRole, error) {
	var roles []*adminTypes.AuthRole
	err := t.db.Cols("id", "name").OrderBy("id asc").Find(&roles)
	if err != nil {
		return nil, errors.Wrap(err, "db Find AuthRole failed")
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
	return errors.Wrap(err, "db exec failed")
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
	return t.db.Where("pid in (select id from auth_permission where method=? and path=?)", method, path).
		And("rid=? ", roleId).Exist(&adminTypes.RolePermission{})
}

func (t *AdminService) GetAdminWithRoleId(uid int) (*adminTypes.Admin_RoleId, error) {
	return crud.GetByOptions[adminTypes.Admin_RoleId](t.db, t.QueryWithRoleIdOption, func(query *xorm.Session) {
		query.ID(uid)
	})
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

func (t *AdminService) MakeLogin(uid int) (string, error) {
	// check account
	admin, err := t.GetAdminWithRoleId(uid)
	if err != nil {
		return "", err
	}
	if admin == nil {
		return "", errors.New("账号不存在")
	}
	if admin.RoleId == 0 {
		return "", errors.New("该账号没有角色，请联系管理员")
	}
	return t.authService.MakeLogin(admin.ID, "")
}

func (t *AdminService) CheckToken(tokenStr string) (*adminTypes.Admin_RoleId, error) {
	uid, err := t.authService.CheckToken(tokenStr)
	if err != nil || uid <= 0 {
		return nil, err
	}

	key := "admin:loginState:" + strconv.Itoa(uid)
	adminRoleId, err := getFallback[adminTypes.Admin_RoleId](t.redisClient, key, func() (*adminTypes.Admin_RoleId, error) {
		return t.GetAdminWithRoleId(uid)
	}, time.Second*10)

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

func (t *AdminService) Logout(uid int) error {
	return t.authService.Logout(uid, "")
}
