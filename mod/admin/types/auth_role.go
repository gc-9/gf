package types

import "time"

type AuthRole struct {
	ID        int       `json:"id" xorm:"pk autoincr 'id'"`
	Key       string    `json:"key" xorm:"'key'"`
	Name      string    `json:"name" xorm:"'name'"`
	Remark    string    `json:"remark" xorm:"'remark'"`
	CreatedAt time.Time `json:"createdAt" xorm:"created 'created_at'"`
	UpdatedAt time.Time `json:"updatedAt" xorm:"updated 'updated_at'"`
}

type AclRole_Permissions struct {
	*AuthRole
	Permissions []*AuthPermission `json:"permissions" xorm:"-"`
}

func (t *AuthRole) TableName() string {
	return "auth_role"
}

type ParamAclRoleStore struct {
	ID          int    `json:"id" validate:"omitempty,min=1"`
	Key         string `json:"key" validate:"required,min=3,max=50"`
	Name        string `json:"name" validate:"required,min=1,max=50"`
	Remark      string `json:"remark" validate:"min=1,max=50"`
	Permissions []int  `json:"permissions"`
}
