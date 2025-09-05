package auth_services

import (
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/crud"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/httplib"
	adminTypes "github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/state"
	"regexp"
	"xorm.io/xorm"
)

func NewPermissionService(db *xorm.Engine) *PermissionService {
	return &PermissionService{
		db:     db,
		CrudDB: crud.NewCrudDB[adminTypes.AuthPermission](db),
	}
}

type PermissionService struct {
	*crud.CrudDB[adminTypes.AuthPermission]
	db *xorm.Engine
}

func (t *PermissionService) Update(id int, permission *adminTypes.AuthPermission, options ...crud.QueryOption) (int64, error) {
	query := t.db.ID(id)
	for _, opFunc := range options {
		opFunc(query)
	}
	return query.Update(permission)
}

func (t *PermissionService) All() ([]*adminTypes.AuthPermission, error) {
	var items []*adminTypes.AuthPermission
	err := t.db.OrderBy("`sort` asc, `path` asc").Find(&items)
	return items, errors.Wrap(err, "db Find failed")
}

func (t *PermissionService) StoreAll(permissions []*adminTypes.AuthPermission) error {
	items, err := t.All()
	if err != nil {
		return err
	}

	newItems := make([]*adminTypes.AuthPermission, 0)
	for _, p := range permissions {
		var isExist bool
		for _, item := range items {
			if item.Method == p.Method && item.Path == p.Path {
				isExist = true
				break
			}
		}
		if isExist {
			continue
		}
		newItems = append(newItems, p)
	}
	if len(newItems) > 0 {
		_, err = t.db.Insert(newItems)
	}
	return errors.Wrap(err, "db Insert failed")
}

func (t *PermissionService) UpdateAclPermissions(servConf *config.Server) error {
	var paths []*regexp.Regexp
	paths = append(paths, servConf.Acl.IgnoreAuthPaths...)
	paths = append(paths, servConf.Acl.IgnoreAclPaths...)
	permissions := filterRoutesToPermissions(state.Routes, paths)
	return t.StoreAll(permissions)
}

func filterRoutesToPermissions(routes []*httplib.Route, ignorePaths []*regexp.Regexp) []*adminTypes.AuthPermission {
	var permissions []*adminTypes.AuthPermission

outLoop:
	for _, v := range routes {

		for _, r := range ignorePaths {
			if r.MatchString(v.Path) {
				continue outLoop
			}
		}

		permissions = append(permissions, &adminTypes.AuthPermission{
			Name:   v.Name,
			Method: v.Method,
			Path:   v.Path,
		})
	}
	return permissions
}
