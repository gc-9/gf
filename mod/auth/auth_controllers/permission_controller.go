package auth_controllers

import (
	"github.com/gc-9/gf/crud"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/httplib"
	adminTypes "github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/mod/auth/auth_services"
	"github.com/gc-9/gf/types"
)

func NewPermissionController(aclRoleService *auth_services.RoleService, aclPermissionService *auth_services.PermissionService) httplib.Router {
	return &aclPermissionController{
		permissionService: aclPermissionService,
		roleService:       aclRoleService,
	}
}

type aclPermissionController struct {
	permissionService *auth_services.PermissionService
	roleService       *auth_services.RoleService
}

func (p *aclPermissionController) Routes() []*httplib.Route {
	return []*httplib.Route{
		httplib.NewRoute("POST", "/authPermission/defined", "", p.Defined),
		httplib.NewRoute("POST", "/authPermission/index", "权限-列表全部", p.Index),
		httplib.NewRoute("POST", "/authPermission/all", "权限-全部列表", p.All),
		httplib.NewRoute("POST", "/authPermission/show", "权限-查看", p.Show),
		httplib.NewRoute("POST", "/authPermission/store", "权限-保存", p.Store),
		httplib.NewRoute("POST", "/authPermission/destroy", "权限-删除", p.Destroy),
	}
}

func (p *aclPermissionController) Defined() (map[string]interface{}, error) {
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

func (p *aclPermissionController) Index(param *types.ParamPageQuery) (*types.PagerData[*adminTypes.AuthPermission], error) {
	var allowFilters = []string{
		"username",
		"uid",
		"name",
		"status",
	}
	param.Filters.Allows(allowFilters)
	return p.permissionService.PagerData(param.ParamPager, param.Filters.QueryOption(), crud.QueryOrderBy("id desc"))
}

func (p *aclPermissionController) All() ([]*adminTypes.AuthPermission, error) {
	return p.permissionService.All()
}

func (p *aclPermissionController) Show(param *types.ParamID) (*adminTypes.AuthPermission, error) {
	item, err := p.permissionService.Get(param.ID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, errors.New("notFound")
	}
	return item, nil
}

type ParamAcPermissionStore struct {
	ID     int    `json:"id" validate:"omitempty,min=1"`
	Name   string `json:"name" validate:"required,min=1,max=50"`
	Method string `json:"method" validate:"required,min=1,max=100"`
	Path   string `json:"path" validate:"required,min=1,max=300"`
	Remark string `json:"remark" validate:"omitempty,min=1,max=100"`
	Sort   int    `json:"sort" validate:"omitempty"`
}

func (p *aclPermissionController) Store(param *ParamAcPermissionStore) error {
	// store
	up := adminTypes.AuthPermission{
		Name:   param.Name,
		Method: param.Method,
		Path:   param.Path,
		Remark: param.Remark,
		Sort:   param.Sort,
	}
	var err error
	if param.ID == 0 {
		_, err = p.permissionService.Create(&up)
		return err
	} else {
		_, err = p.permissionService.Update(param.ID, &up, crud.QueryMustCols("sort"))
		return err
	}
}

func (p *aclPermissionController) Destroy(param *types.ParamID) error {
	_, err := p.permissionService.Delete(param.ID)
	return err
}
