package auth_controllers

import (
	"github.com/gc-9/gf/crud"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/httplib"
	adminTypes "github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/mod/auth/auth_services"
	"github.com/gc-9/gf/types"
	"xorm.io/xorm"
)

func NewRoleController(db *xorm.Engine, roleService *auth_services.RoleService) httplib.Router {
	return &roleController{
		db:          db,
		roleService: roleService,
	}
}

type roleController struct {
	db          *xorm.Engine
	roleService *auth_services.RoleService
}

func (p *roleController) Routes() []*httplib.Route {
	return []*httplib.Route{
		httplib.NewRoute("POST", "/authRole/defined", "", p.Defined),
		httplib.NewRoute("POST", "/authRole/index", "角色-列表", p.Index),
		httplib.NewRoute("POST", "/authRole/show", "角色-查看", p.Show),
		httplib.NewRoute("POST", "/authRole/store", "角色-保存", p.Store),
	}
}

func (p *roleController) Defined() (map[string]interface{}, error) {
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

func (p *roleController) Index(param *types.ParamPageQuery) (*types.PagerData[*adminTypes.AuthRole], error) {
	var allowFilters = []string{
		"username",
		"uid",
		"name",
		"status",
	}
	param.Filters.Allows(allowFilters)
	return p.roleService.PagerData(param.ParamPager, param.Filters.QueryOption(), crud.QueryOrderBy("id desc"))
}

func (p *roleController) Show(param *types.ParamID) (*adminTypes.AclRole_Permissions, error) {
	item, err := p.roleService.Get(param.ID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, errors.New("notFound")
	}

	ps, err := p.roleService.GetPermissions(param.ID)
	if err != nil {
		return nil, err
	}
	if ps == nil {
		ps = []*adminTypes.AuthPermission{}
	}

	return &adminTypes.AclRole_Permissions{Permissions: ps, AuthRole: item}, nil
}

func (p *roleController) Store(param *adminTypes.ParamAclRoleStore) error {
	return p.roleService.Store(param)
}
