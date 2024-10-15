package middleware

import (
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/httpLib"
	adminTypes "github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/mod/auth/auth_services"
	"github.com/gc-9/gf/types"
	"github.com/labstack/echo/v4"
	"regexp"
	"strings"
)

func AdminAuthPermission(prefix string, adminService *auth_services.AdminService, ignorePaths []*regexp.Regexp) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.(httpLib.RequestContext)

			method, path := ctx.Request().Method, ctx.Path()
			if prefix != "" {
				path = strings.TrimPrefix(path, prefix)
			}

			for _, r := range ignorePaths {
				if r.MatchString(path) {
					return next(c)
				}
			}

			adminRole := ctx.AuthUser().(*adminTypes.Admin_RoleId)

			ok, err := adminService.IsHasPermission(adminRole.RoleId, adminRole.RoleKey, method, path)
			if err != nil {
				return httpLib.SendResponse(c, nil, err)
			}
			if !ok {
				return httpLib.SendResponse(c, nil, &errors.ErrMessage{
					Code:     types.StatusCodeNoPermission,
					HumanMsg: "noPermission",
				})
			}
			return next(c)
		}
	}
}
