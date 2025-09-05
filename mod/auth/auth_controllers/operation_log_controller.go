package auth_controllers

import (
	"github.com/gc-9/gf/crud"
	"github.com/gc-9/gf/httplib"
	adminTypes "github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/mod/auth/auth_services"
	"github.com/gc-9/gf/types"
	"xorm.io/builder"
	"xorm.io/xorm"
)

func NewOperationLogController(db *xorm.Engine, roleService *auth_services.RoleService) httplib.Router {
	return &OperationLogController{
		db:          db,
		roleService: roleService,
	}
}

type OperationLogController struct {
	db          *xorm.Engine
	roleService *auth_services.RoleService
}

func (p *OperationLogController) Routes() []*httplib.Route {
	return []*httplib.Route{
		httplib.NewRoute("POST", "/operation_log/defined", "日志-定义", p.Defined),
		httplib.NewRoute("POST", "/operation_log/index", "日志-列表", p.Index),
	}
}

func (p *OperationLogController) Defined(ctx httplib.RequestContext) (map[string]interface{}, error) {
	roles, err := p.roleService.ListNames()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"statusOptions": types.StatusOptions,
		"statusMap":     types.StatusMap,
		"roles":         roles,
	}, nil
}

func (p *OperationLogController) Index(param *types.ParamPageQuery) (*types.PagerData[*adminTypes.AdminLogFull], error) {
	var allowFilters = []string{
		"uid",
		"keyword",
	}
	param.Filters.Allows(allowFilters)

	return crud.PagerData[adminTypes.AdminLogFull](p.db, param.ParamPager, func(session *xorm.Session) {
		session.Alias("l").Select("l.*, r.name op_role_name, u.username op_username, u.name op_name").
			Join("LEFT", []string{"auth_role", "r"}, "r.id=l.rid").
			Join("LEFT", []string{"admin", "u"}, "u.id=l.uid").
			Desc("l.id").
			Limit(param.PageSize, param.Offset)

		for key, f := range param.Filters {
			switch key {
			case "keyword":
				v := "%" + param.Filters.ValueString(key) + "%"
				session.Where("l.action like ? or l.data like ? or l.method like ?", v, v, v)
				/*session.Where(builder.Or(
					builder.Like{"l.action", f.ValueString()},
					builder.Like{"l.data=1 and l.data", f.ValueString()},
					builder.Like{"l.method", f.ValueString()},
				))*/
			default:
				session.Where(builder.Eq{"l." + key: f})
			}
		}
	})
}
