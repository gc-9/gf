package types

import "time"

type RolePermission struct {
	ID        int       `json:"id" xorm:"pk autoincr 'id'"`
	Rid       int       `json:"rid" xorm:"'rid'"`
	Pid       int       `json:"pid" xorm:"'pid'"`
	CreatedAt time.Time `json:"createdAt" xorm:"created 'created_at'"`
	UpdatedAt time.Time `json:"updatedAt" xorm:"updated 'updated_at'"`
}

func (t *RolePermission) TableName() string {
	return "auth_role_permission"
}
