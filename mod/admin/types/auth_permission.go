package types

import "time"

type AuthPermission struct {
	ID        int       `json:"id" xorm:"pk autoincr 'id'"`
	Name      string    `json:"name" xorm:"'name'"`
	Path      string    `json:"path" xorm:"'path'"`
	Method    string    `json:"method" xorm:"'method'"`
	Remark    string    `json:"remark" xorm:"'remark'"`
	Sort      int       `json:"sort" xorm:"'sort'"`
	CreatedAt time.Time `json:"createdAt" xorm:"created 'created_at'"`
	UpdatedAt time.Time `json:"updatedAt" xorm:"updated 'updated_at'"`
}

func (t *AuthPermission) TableName() string {
	return "auth_permission"
}
