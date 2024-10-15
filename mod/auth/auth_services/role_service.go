package auth_services

import (
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/crud"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/mod/admin/types"
	"xorm.io/xorm"
)

func NewRoleService(db *xorm.Engine, servConf *config.Server) *RoleService {
	return &RoleService{
		db:       db,
		CrudDB:   crud.NewCrudDB[types.AuthRole](db),
		servConf: servConf,
	}
}

type RoleService struct {
	*crud.CrudDB[types.AuthRole]
	db       *xorm.Engine
	servConf *config.Server
}

func (s *RoleService) ListNames() ([]*types.AuthRole, error) {
	return s.List(func(query *xorm.Session) {
		query.Cols("id", "name").OrderBy("id asc")
	})
}

func (s *RoleService) GetPermissions(rolId int) ([]*types.AuthPermission, error) {
	return crud.List[types.AuthPermission](s.db, func(query *xorm.Session) {
		query.Where("id in (select pid from auth_role_permission where rid=?)", rolId).
			OrderBy("`sort` asc, id asc")
	})
}

func (s *RoleService) Store(param *types.ParamAclRoleStore) error {
	if param.Key == s.servConf.Acl.SuperRoleKey {
		return errors.New("系统角色不能添加或修改")
	}

	// check param permissions
	count, err := s.db.In("id", param.Permissions).Count(&types.AuthPermission{})
	if err != nil {
		return errors.Wrap(err, "Count failed")
	}
	if int(count) != len(param.Permissions) {
		return errors.New("权限列表错误")
	}

	// store
	up := types.AuthRole{
		Name:   param.Name,
		Key:    param.Key,
		Remark: param.Remark,
	}

	tx := s.db.NewSession()
	err = tx.Begin()
	if err != nil {
		return errors.Wrap(err, "db tx.Begin failed")
	}

	roleTX := s.TX(tx)
	permissionTx := crud.NewCrudTX[types.RolePermission](tx)

	roleId := param.ID
	if param.ID == 0 {
		_, err = roleTX.Create(&up)
		roleId = up.ID
	} else {
		_, err = roleTX.Update(param.ID, &up)
	}
	if err != nil {
		return errors.Wrap(err, "db Insert/Update failed")
	}

	// store permissions
	if param.Permissions == nil || len(param.Permissions) == 0 {
		_, err = permissionTx.DeleteOptions(func(query *xorm.Session) {
			query.Where("rid = ?", roleId)
		})
		if err != nil {
			return err
		}
	} else {
		permissions, err := permissionTx.List(func(query *xorm.Session) {
			query.Where("rid = ?", roleId)
		})
		if err != nil {
			return err
		}

		oldIds := map[int]bool{}
		for _, v := range permissions {
			oldIds[v.Pid] = true
		}

		var inserts []*types.RolePermission
		for _, pid := range param.Permissions {
			if _, ok := oldIds[pid]; ok {
				continue
			}
			inserts = append(inserts, &types.RolePermission{
				Pid: pid,
				Rid: roleId,
			})
		}

		// del
		_, err = permissionTx.DeleteOptions(func(query *xorm.Session) {
			query.Where("rid = ?", roleId).NotIn("pid", param.Permissions)
		})
		if err != nil {
			return errors.Wrap(err, "db Delete failed")
		}

		// add
		if len(inserts) > 0 {
			_, err = permissionTx.Creates(inserts)
			if err != nil {
				return errors.Wrap(err, "db Insert failed")
			}
		}

	}

	err = tx.Commit()
	return errors.Wrap(err, "db tx.Commit failed")
}
