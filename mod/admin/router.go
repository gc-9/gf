package admin

import (
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/httplib"
	"github.com/gc-9/gf/mod/admin/middleware"
	"github.com/gc-9/gf/mod/auth/auth_services"
	"github.com/gc-9/gf/state"
	"github.com/labstack/echo/v4"
	"regexp"

	"xorm.io/xorm"
)

func RegisterRouters(routers []httplib.Router, e *echo.Echo, servConf *config.Server,
	db *xorm.Engine, adminService *auth_services.AdminService) error {

	prefix := servConf.Prefix
	g := e.Group(prefix)

	// middlewares
	logMiddleware := middleware.AdminAuditLog(routers, prefix, db, servConf.Acl.IgnoreAuditLogPaths)
	permissionMiddleware := middleware.AdminAuthPermission(prefix, adminService, servConf.Acl.IgnoreAclPaths)
	authMiddleware := middleware.UserAdminAuth(adminService, servConf)

	authGroup := g.Group("", authMiddleware, permissionMiddleware, logMiddleware)
	guestGroup := g

	var routes []*httplib.Route

	// register routes
	ignoreAuthPaths := servConf.Acl.IgnoreAuthPaths
	for _, rt := range routers {
		for _, r := range rt.Routes() {
			if unAuthSkip(r, ignoreAuthPaths) {
				guestGroup.Add(r.Method, r.Path, r.HandlerFunc).Name = r.Name
			} else {
				authGroup.Add(r.Method, r.Path, r.HandlerFunc).Name = r.Name
			}
			routes = append(routes, r)
		}
	}

	// state
	state.Routes = routes

	// api doc
	if servConf.DocPath != "" {
		g.GET(servConf.DocPath, httplib.HandlerApiDoc(servConf, routes))
	}

	return nil
}

func unAuthSkip(r *httplib.Route, ignorePaths []*regexp.Regexp) bool {
	for _, v := range ignorePaths {
		if v.MatchString(r.Path) {
			return true
		}
	}
	return false
}
