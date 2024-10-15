package auth_controllers

import (
	"github.com/gc-9/gf/auth"
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/httpLib"
	"github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/mod/auth/auth_services"
	"github.com/gc-9/gf/util"
	"math"
	"strings"
	"time"
)

func NewPassportController(adminService *auth_services.AdminService,
	captchaService auth.CaptchaProvide, confServer *config.Server) httpLib.Router {
	return &PassportController{
		adminService:   adminService,
		captchaService: captchaService,
		confServer:     confServer,
	}
}

type PassportController struct {
	adminService   *auth_services.AdminService
	captchaService auth.CaptchaProvide
	confServer     *config.Server
}

func (p *PassportController) Routes() []*httpLib.Route {
	return []*httpLib.Route{
		httpLib.NewRoute("POST", "/passport/login", "", p.Login),
		httpLib.NewRoute("GET", "/passport/captcha", "", p.Captcha),
		httpLib.NewRoute("GET", "/passport/logout", "", p.Logout),
		httpLib.NewRoute("GET", "/passport/user", "", p.User),
	}
}

type LoginParam struct {
	Username  string `json:"username" validate:"required"`
	Password  string `json:"password" validate:"required"`
	CaptchaId string `json:"captchaId" validate:"required"`
	Captcha   string `json:"captcha" validate:"required"`
}

func (p *PassportController) Login(ctx httpLib.RequestContext, param *LoginParam) (map[string]interface{}, error) {
	ok, err := p.captchaService.Validate(param.CaptchaId, param.Captcha)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("验证码错误")
	}

	var user *types.Admin
	user, err = p.adminService.GetByUserName(param.Username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("用户名或密码错误")
	}

	// 检查错误次数
	dueTime, err := p.adminService.CheckLoginFailTimes(user.ID)
	if err != nil {
		return nil, err
	}
	if dueTime != nil {
		return nil, errors.Errorf("错误次数过多，%.0f分钟后再试", math.Ceil(dueTime.Minutes()))
	}

	if !auth.CompareHashAndPassword(user.Password, param.Password) {
		_ = p.adminService.CreateLoginFailLog(user.ID, ctx.RealIP(), util.Substring(ctx.Request().UserAgent(), 0, 200))
		return nil, errors.New("用户名或密码错误")
	}

	// check status
	if user.Status <= 0 {
		return nil, errors.New("用户已禁用")
	}

	// get role
	role, err := p.adminService.GetRole(user.ID)
	if err != nil {
		return nil, errors.Wrap(err, "GetRole failed")
	}
	if role == nil {
		return nil, errors.New("该用户无任何权限")
	}

	// data
	data := map[string]interface{}{}

	// update last login
	up := types.Admin{
		LastLoginAt: time.Now(),
		LastLoginIP: ctx.RealIP(),
	}
	_, err = p.adminService.Update(user.ID, &up)
	if err != nil {
		return nil, err
	}

	// token
	token, err := p.adminService.MakeLogin(user.ID)
	if err != nil {
		return nil, err
	}

	data["token"] = token
	data["user"] = user
	return data, nil
}

// User get user
func (p *PassportController) User(ctx httpLib.RequestContext) (map[string]interface{}, error) {
	authUser := ctx.AuthUser().(*types.Admin_RoleId)

	admin, err := p.adminService.GetAdminRolePermissions(authUser.ID)
	if err != nil {
		return nil, err
	}
	if admin.Role == nil {
		return nil, errors.New("该用户无任何权限")
	}

	var data = map[string]interface{}{}
	if admin.RoleKey == p.confServer.Acl.SuperRoleKey {
		data["isSuper"] = true
		data["permissions"] = []string{"*"}
	} else {
		// get permission
		data["isSuper"] = false
		var ps []string
		for _, v := range admin.Permissions {
			ps = append(ps, strings.ToLower(v.Method)+":"+strings.ReplaceAll(strings.TrimLeft(v.Path, "/"), "/", "."))
		}
		data["permissions"] = ps
	}

	data["role"] = admin.Role
	data["user"] = admin
	admin.Permissions = nil
	admin.Role = nil
	return data, nil
}

func (p *PassportController) Logout(ctx httpLib.RequestContext) error {
	if ctx.AuthUser() == nil {
		return nil
	}
	authUser := ctx.AuthUser().(*types.Admin_RoleId)
	err := p.adminService.Logout(authUser.ID)
	if err != nil {
		return err
	}
	return nil
}

func (p *PassportController) Captcha() (*auth.CaptchaAlloc, error) {
	return p.captchaService.Alloc(210, 70)
}
