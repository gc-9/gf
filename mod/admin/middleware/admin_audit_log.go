package middleware

import (
	"bytes"
	"github.com/gc-9/gf/httplib"
	"github.com/gc-9/gf/logger"
	adminTypes "github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/util"
	"github.com/gc-9/gf/util/http_util"
	"github.com/labstack/echo/v4"
	"io"
	"regexp"
	"strings"
	"xorm.io/xorm"
)

func AdminAuditLog(routers []httplib.Router, prefix string, db *xorm.Engine, ignorePath []*regexp.Regexp) echo.MiddlewareFunc {
	permissionNames := make(map[string]string)
	for _, g := range routers {
		for _, r := range g.Routes() {
			permissionNames[r.Method+":"+r.Path] = r.Name
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.(httplib.RequestContext)

			if ctx.AuthUser() == nil {
				return next(c)
			}

			for _, v := range ignorePath {
				if v.MatchString(ctx.Path()) {
					return next(c)
				}
			}

			req := ctx.Request()

			// dump body
			var reqBody []byte
			if req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH" {
				if req.Body != nil {
					reqBody, _ = io.ReadAll(c.Request().Body)
					req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
				}
			}

			errNext := next(c)

			body := ""
			if len(reqBody) > 0 {
				if strings.HasPrefix(req.Header.Get("Content-Type"), "application/json") {
					body = string(reqBody)
				} else {
					body, _ = http_util.DumpRequestForm(req)
				}
			}
			body = util.Substring(body, 0, 500)
			path := strings.TrimPrefix(req.URL.Path, prefix)
			uri := strings.TrimPrefix(req.URL.RequestURI(), prefix)
			name, _ := permissionNames[req.Method+":"+path]
			admin := ctx.AuthUser().(*adminTypes.Admin_RoleId)
			_, err := db.Insert(&adminTypes.AdminLog{
				Uid:       admin.ID,
				Rid:       admin.RoleId,
				Method:    req.Method,
				Action:    uri,
				Data:      filterSensContent(body),
				Ip:        ctx.RealIP(),
				UserAgent: util.Substring(ctx.Request().UserAgent(), 0, 200),
				Remark:    name,
			})
			if err != nil {
				logger.Logger().Error(err)
			}

			return errNext
		}
	}

}

var regPass = regexp.MustCompile(`("password":")([^"]*)`)
var sensFilterFunc = []func(string) string{
	func(s string) string {
		return regPass.ReplaceAllString(s, "${1}***")
	},
}

func filterSensContent(str string) string {
	for _, f := range sensFilterFunc {
		str = f(str)
	}
	return str
}
