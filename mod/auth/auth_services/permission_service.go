package auth_services

import (
	"regexp"

	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/crud"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/httplib"
	adminTypes "github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/state"
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

	itemMap := make(map[string]*adminTypes.AuthPermission, len(items))
	for _, item := range items {
		key := item.Method + "|" + item.Path
		itemMap[key] = item
	}

	newItems := make([]*adminTypes.AuthPermission, 0)
	updatedItems := make([]*adminTypes.AuthPermission, 0)

	for _, p := range permissions {
		key := p.Method + "|" + p.Path
		if exist, ok := itemMap[key]; ok {
			// 同一路径和方法已存在，若名称有变化则更新
			if exist.Name != p.Name {
				exist.Name = p.Name
				updatedItems = append(updatedItems, exist)
			}
			continue
		}
		newItems = append(newItems, p)
	}

	session := t.db.NewSession()
	defer session.Close()

	if err = session.Begin(); err != nil {
		return errors.Wrap(err, "db begin failed")
	}

	if len(newItems) > 0 {
		if _, err = session.Insert(newItems); err != nil {
			_ = session.Rollback()
			return errors.Wrap(err, "db Insert failed")
		}
	}

	if len(updatedItems) > 0 {
		for _, item := range updatedItems {
			if _, err = session.ID(item.ID).Cols("name").Update(item); err != nil {
				_ = session.Rollback()
				return errors.Wrap(err, "db Update failed")
			}
		}
	}

	if err = session.Commit(); err != nil {
		return errors.Wrap(err, "db Commit failed")
	}

	return nil
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
