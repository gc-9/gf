package middleware

import (
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/httplib"
	"github.com/gc-9/gf/mod/auth/auth_services"
	"github.com/gc-9/gf/types"
	"github.com/labstack/echo/v4"
)

/*
func UserAuth(userService *services.UserService, authService *auth.AuthService, serveConf *config.Server) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			c := ctx.(http.RequestContext)

			token := c.Request().Header.Get(serveConf.Acl.AuthHeader)
			u, err := checkToken(userService, authService, token)
			if err != nil {
				return http.SendResponse(c, nil, err)
			}

			if u == nil {
				pt := strings.TrimPrefix(c.Path(), serveConf.Prefix)
				for _, v := range serveConf.Acl.AllowGuestPaths {
					if v.MatchString(pt) {
						return next(c)
					}
				}
				err = &errors.ErrMessage{
					Code:     types2.StatusCodeUnauthorized,
					HumanMsg: "authError",
				}
				return http.SendResponse(c, nil, err)
			}
			c.Set("user", u)
			return next(c)
		}
	}
}

func checkToken(userService *services.UserService, authService *auth.AuthService, token string) (*types.User, error) {
	if len(token) == 0 {
		return nil, nil
	}

	uid, err := authService.CheckToken(token)
	if err != nil || uid <= 0 {
		return nil, err
	}
	return userService.AuthUser(uid)
}*/

func UserAdminAuth(adminService *auth_services.AdminService, servConf *config.Server) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			c := ctx.(httplib.RequestContext)

			admin, err := adminService.CheckToken(c.Request().Header.Get(servConf.Acl.AuthHeader))
			if err != nil {
				return httplib.SendResponse(c, nil, err)
			}
			if admin == nil {
				return httplib.SendResponse(c, nil, &errors.ErrMessage{
					Code:     types.StatusCodeUnauthorized,
					HumanMsg: "authError",
				})
			}
			c.Set("authUser", admin)
			return next(c)
		}
	}
}
