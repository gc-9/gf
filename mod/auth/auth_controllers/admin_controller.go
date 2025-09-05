package auth_controllers

import (
	"github.com/gc-9/gf/auth"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/httplib"
	adminTypes "github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/mod/auth/auth_services"
	"github.com/gc-9/gf/types"
)

func NewAdminController(aclRoleService *auth_services.RoleService, adminService *auth_services.AdminService) httplib.Router {
	return &adminController{
		adminService:   adminService,
		aclRoleService: aclRoleService,
	}
}

type adminController struct {
	adminService   *auth_services.AdminService
	aclRoleService *auth_services.RoleService
}

func (p *adminController) Routes() []*httplib.Route {
	return []*httplib.Route{
		httplib.NewRoute("POST", "/admin/defined", "", p.Defined),
		httplib.NewRoute("POST", "/admin/index", "管理员-列表", p.Index),
		httplib.NewRoute("POST", "/admin/show", "管理员-查看", p.Show),
		httplib.NewRoute("POST", "/admin/store", "管理员-保存", p.Store),
		httplib.NewRoute("POST", "/admin/toggleStatus", "管理员-状态更新", p.ToggleStatus),
		httplib.NewRoute("GET", "/admin/showSelf", "管理员-查看自己", p.ShowSelf),
		httplib.NewRoute("POST", "/admin/storeSelf", "管理员-保存自己", p.StoreSelf),
	}
}

func (p *adminController) Defined() (map[string]interface{}, error) {
	roles, err := p.aclRoleService.ListNames()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"statusOptions": types.StatusOptions,
		"statusMap":     types.StatusMap,
		"roles":         roles,
	}, nil
}

func (p *adminController) Index(param *types.ParamPageQuery) (*types.PagerData[*adminTypes.Admin_R], error) {
	var allowFilters = []string{
		"username",
		"id",
		"name",
		"status",
	}
	param.Filters.Allows(allowFilters)

	return p.adminService.ListAdminRole(param)
}

type paramPassportShow struct {
	ID int `json:"id" form:"id" query:"id" validate:"required,min=1"`
}

func (p *adminController) Show(param *paramPassportShow) (*adminTypes.Admin_RoleId, error) {
	return p.adminService.GetAdminWithRoleId(param.ID)
}

func (p *adminController) ShowSelf(ctx httplib.RequestContext) (map[string]interface{}, error) {
	adminRole := ctx.AuthUser().(*adminTypes.Admin_RoleId)
	admin, err := p.adminService.Get(adminRole.ID)
	if err != nil {
		return nil, err
	}
	if admin == nil {
		return nil, errors.New("notFound")
	}

	// get role
	role, err := p.adminService.GetRole(admin.ID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("该用户无任何权限")
	}

	return map[string]interface{}{
		"admin": admin,
		"role":  role,
	}, nil
}

type paramStoreSelf struct {
	Password string `json:"password" validate:"omitempty,min=6,max=20"`
	Name     string `json:"name" validate:"required,min=2,max=20"`
	Mobile   string `json:"mobile" validate:"required,min=6,max=20"`
}

func (p *adminController) StoreSelf(ctx httplib.RequestContext, param *paramStoreSelf) (err error) {
	// store
	up := adminTypes.Admin{
		Name:   param.Name,
		Mobile: param.Mobile,
	}
	if param.Password != "" {
		up.Password = auth.GenerateFromPassword(param.Password)
	}

	adminRole := ctx.AuthUser().(*adminTypes.Admin_RoleId)
	_, err = p.adminService.Update(adminRole.ID, &up)
	return err
}

type paramStore struct {
	*types.IgnoreAutoValidate
	ID       int    `json:"id" validate:"omitempty,min=1"`
	Username string `json:"username" validate:"required,min=3,max=20"`
	Name     string `json:"name" validate:"required,min=2,max=20"`
	Password string `json:"password" validate:"omitempty,min=6,max=20"`
	Mobile   string `json:"mobile" validate:"omitempty,min=3,max=20"`
	Status   int    `json:"status" validate:"required"`
	RoleId   int    `json:"roleId" validate:"required,min=1"`
}

func (p *adminController) Store(ctx httplib.RequestContext, param *paramStore) (err error) {
	adminRole := ctx.AuthUser().(*adminTypes.Admin_RoleId)
	role, err := p.adminService.GetRole(adminRole.ID)
	if err != nil {
		return err
	}
	if role.ID != 1 {
		return errors.New("非管理员不能创建和修改角色")
	}

	// store
	admin := &adminTypes.Admin{
		Username: param.Username,
		Name:     param.Name,
		Mobile:   param.Mobile,
		Status:   param.Status,
	}
	if param.Password != "" {
		admin.Password = auth.GenerateFromPassword(param.Password)
	}

	if param.ID == 0 {
		if err = ctx.Validate(param); err != nil {
			return
		}
		if param.Password == "" {
			return errors.New("密码不能为空")
		}
		_, err = p.adminService.CreateAdmin(param.RoleId, admin)
		if err != nil {
			return
		}
	} else {
		has, err := p.adminService.Exist(param.ID)
		if err != nil {
			return err
		}
		if !has {
			return errors.New("用户不存在")
		}

		// 不能修改自己的角色和状态
		if param.ID == adminRole.ID {
			param.Username = ""
			param.RoleId = 0
			param.Status = 0
		}

		_, err = p.adminService.Update(param.ID, admin)
		if err != nil {
			return err
		}

		if param.RoleId != 0 {
			err = p.adminService.UpdateRole(param.ID, param.RoleId)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *adminController) ToggleStatus(param *types.ParamToggleStatus) (int, error) {
	c, err := p.adminService.Update(param.ID, &adminTypes.Admin{
		ID:     param.ID,
		Status: param.Status, // 状态
	})
	return c, err
}
