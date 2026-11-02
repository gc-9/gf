package auth_controllers

import (
	"fmt"
	"math"
	"strings"
	"time"

	altcha "github.com/altcha-org/altcha-lib-go"
	"github.com/gc-9/gf/auth"
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/httplib"
	"github.com/gc-9/gf/logger"
	"github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/mod/auth/auth_services"
	"github.com/gc-9/gf/util"
)

func NewPassportController(adminService *auth_services.AdminService,
	altchaService *auth.AltchaService, captchaService auth.CaptchaProvide, confServer *config.Server) httplib.Router {
	return &PassportController{
		adminService:   adminService,
		altchaService:  altchaService,
		captchaService: captchaService,
		confServer:     confServer,
	}
}

type PassportController struct {
	adminService   *auth_services.AdminService
	altchaService  *auth.AltchaService
	captchaService auth.CaptchaProvide
	confServer     *config.Server
}

func (p *PassportController) Routes() []*httplib.Route {
	return []*httplib.Route{
		httplib.NewRoute("POST", "/passport/login", "", p.Login),
		httplib.NewRoute("GET", "/passport/altcha", "", p.Altcha),
		httplib.NewRoute("GET", "/passport/captcha", "", p.Captcha),
		httplib.NewRoute("GET", "/passport/logout", "", p.Logout),
		httplib.NewRoute("GET", "/passport/user", "", p.User),
	}
}

type LoginParam struct {
	Username     string `json:"username" validate:"required"`
	Password     string `json:"password" validate:"required"`
	SecurityMode string `json:"securityMode" validate:"required"`
	Altcha       string `json:"altcha"`
	CaptchaId    string `json:"captchaId"`
	Captcha      string `json:"captcha"`
}

func (p *PassportController) Login(ctx httplib.RequestContext, param *LoginParam) (map[string]interface{}, error) {
	switch param.SecurityMode {
	case "altcha":
		if param.Altcha == "" {
			return nil, errors.Public("请完成安全验证")
		}

		ok, err := p.altchaService.Validate(ctx.Request().Context(), param.Altcha)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.Public("安全验证失败，请重试")
		}
	case "captcha":
		if param.CaptchaId == "" || param.Captcha == "" {
			return nil, errors.Public("请完成安全验证")
		}

		ok, err := p.captchaService.Validate(param.CaptchaId, param.Captcha)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.Public("验证码错误")
		}
	default:
		return nil, errors.Public("不支持的安全验证方式")
	}

	ip := ctx.RealIP()
	wait, _, err := p.adminService.LoginIPLimit(ctx.Request().Context(), ip, false)
	if err != nil {
		return nil, err
	}
	if wait > 0 {
		return nil, errors.Public(fmt.Sprintf("当前IP被禁止，请%.0f秒后再试", math.Ceil(wait.Seconds())))
	}

	var user *types.Admin
	user, err = p.adminService.GetByUserName(param.Username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		ipWait, newlyLocked, err := p.adminService.LoginIPLimit(ctx.Request().Context(), ip, true)
		if err != nil {
			return nil, err
		}
		if newlyLocked {
			if err := p.adminService.CreateLoginIPLockLog(ip, util.Substring(ctx.Request().UserAgent(), 0, 200), ipWait); err != nil {
				logger.Logger().Error(err)
			}
		}
		if ipWait > 0 {
			return nil, errors.Public(fmt.Sprintf("当前IP被禁止，请%.0f秒后再试", math.Ceil(ipWait.Seconds())))
		}
		return nil, errors.Public("用户名或密码错误")
	}

	if user.LoginUnlockAt != nil {
		if remaining := time.Until(*user.LoginUnlockAt); remaining > 0 {
			return nil, errors.Public(fmt.Sprintf("账号登录失败次数过多，请%.0f秒后再试", math.Ceil(remaining.Seconds())))
		}
	}

	if !auth.CompareHashAndPassword(user.Password, param.Password) {
		// 用户名正确、密码错误时，同时增加 IP 和账号失败次数。
		ipWait, newlyLocked, err := p.adminService.LoginIPLimit(ctx.Request().Context(), ip, true)
		if err != nil {
			return nil, err
		}
		if newlyLocked {
			if err := p.adminService.CreateLoginIPLockLog(ip, util.Substring(ctx.Request().UserAgent(), 0, 200), ipWait); err != nil {
				logger.Logger().Error(err)
			}
		}
		currentWait, nextWait, err := p.adminService.RecordLoginFailure(user.ID)
		if err != nil {
			logger.Logger().Error(err)
		}
		if err := p.adminService.CreateLoginFailLog(user.ID, ip, util.Substring(ctx.Request().UserAgent(), 0, 200)); err != nil {
			logger.Logger().Error(err)
		}
		if ipWait > 0 {
			return nil, errors.Public(fmt.Sprintf("当前IP被禁止，请%.0f秒后再试", math.Ceil(ipWait.Seconds())))
		}
		if currentWait > 0 {
			if nextWait > 0 {
				return nil, errors.Public(fmt.Sprintf("用户名或密码错误，请%.0f秒后重试；再次失败需等待%.0f秒", math.Ceil(currentWait.Seconds()), nextWait.Seconds()))
			}
			return nil, errors.Public(fmt.Sprintf("账号登录失败次数过多，请%.0f秒后再试", math.Ceil(currentWait.Seconds())))
		}
		return nil, errors.Public("用户名或密码错误")
	}

	// check status
	if user.Status <= 0 {
		return nil, errors.Public("用户已禁用")
	}

	// get role
	role, err := p.adminService.GetRole(user.ID)
	if err != nil {
		return nil, errors.EnsurePublic(err, "操作失败，请稍后重试")
	}
	if role == nil {
		return nil, errors.Public("该用户无任何权限")
	}

	// data
	data := map[string]interface{}{}

	// token
	token, err := p.adminService.MakeLogin(user.ID, ctx.Request().UserAgent())
	if err != nil {
		return nil, err
	}

	// 更新最后登录信息并清除锁定，使用 map 确保写入零值和 NULL。
	loginAt, loginIP := time.Now(), ip
	_, err = p.adminService.Update(user.ID, map[string]interface{}{
		"last_login_at":    loginAt,
		"last_login_ip":    loginIP,
		"login_fail_times": 0,
		"login_unlock_at":  nil,
	})
	if err != nil {
		return nil, err
	}
	user.LastLoginAt = loginAt
	user.LastLoginIP = loginIP
	user.LoginFailTimes = 0
	user.LoginUnlockAt = nil

	err = p.adminService.CreateLoginLog(user.ID, role.ID, ctx.RealIP(), util.Substring(ctx.Request().UserAgent(), 0, 200))
	if err != nil {
		logger.Logger().Error(err)
	}

	data["token"] = token
	data["user"] = user
	return data, nil
}

// User get user
func (p *PassportController) User(ctx httplib.RequestContext) (map[string]interface{}, error) {
	authUser := ctx.AuthUser().(*types.Admin_RoleId)

	admin, err := p.adminService.GetAdminRolePermissions(authUser.ID)
	if err != nil {
		return nil, err
	}
	if admin.Role == nil {
		return nil, errors.Public("该用户无任何权限")
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

func (p *PassportController) Logout(ctx httplib.RequestContext) error {
	if ctx.AuthUser() == nil {
		return nil
	}
	err := p.adminService.Logout(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (p *PassportController) Altcha() (*altcha.Challenge, error) {
	return p.altchaService.Alloc()
}

func (p *PassportController) Captcha() (*auth.CaptchaAlloc, error) {
	return p.captchaService.Alloc(114, 38)
}
