package types

import "time"

type AuthAdminRole struct {
	ID        int       `json:"id" xorm:"pk autoincr 'id'"`
	Rid       int       `json:"rid" xorm:"'rid'"`
	Uid       int       `json:"uid" xorm:"'uid'"`
	CreatedAt time.Time `json:"createdAt" xorm:"created 'created_at'"`
	UpdatedAt time.Time `json:"updatedAt" xorm:"updated 'updated_at'"`
}

func (t *AuthAdminRole) TableName() string {
	return "auth_admin_role"
}
