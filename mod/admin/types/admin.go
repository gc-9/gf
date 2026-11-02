package types

import (
	"time"
)

type Admin struct {
	ID          int       `json:"id" xorm:"pk autoincr 'id'"`
	Username    string    `json:"username" xorm:"'username'"`
	Name        string    `json:"name" xorm:"'name'"`
	Mobile      string    `json:"mobile" xorm:"'mobile'"`
	Sex         int       `json:"sex" xorm:"'sex'"`
	Birthday    string    `json:"birthday" xorm:"'birthday'"`
	Avatar      string    `json:"avatar" xorm:"'avatar'"`
	Password    string    `json:"-" xorm:"'password'"`
	Salt        string    `json:"-" xorm:"'salt'"`
	Status      int       `json:"status" xorm:"'status'"`
	LastLoginIP string    `json:"lastLoginIP" xorm:"'last_login_ip'"`
	LastLoginAt time.Time `json:"lastLoginAt" xorm:"'last_login_at'"`
	CreatedAt   time.Time `json:"createdAt" xorm:"created 'created_at'"`
	UpdatedAt   time.Time `json:"updatedAt" xorm:"updated 'updated_at'"`

	LoginFailTimes int64      `json:"loginFailTimes" xorm:"notnull default 0 'login_fail_times'"`
	LoginUnlockAt  *time.Time `json:"loginUnlockAt" xorm:"'login_unlock_at'"`
}

func (t *Admin) TableName() string {
	return "admin"
}

type Admin_RoleId struct {
	ID          int       `json:"id" xorm:"pk autoincr 'id'"`
	Username    string    `json:"username" xorm:"'username'"`
	Name        string    `json:"name" xorm:"'name'"`
	Mobile      string    `json:"mobile" xorm:"'mobile'"`
	Sex         int       `json:"sex" xorm:"'sex'"`
	Birthday    string    `json:"birthday" xorm:"'birthday'"`
	Avatar      string    `json:"avatar" xorm:"'avatar'"`
	Password    string    `json:"-" xorm:"'password'"`
	Salt        string    `json:"-" xorm:"'salt'"`
	Status      int       `json:"status" xorm:"'status'"`
	LastLoginIP string    `json:"lastLoginIP" xorm:"'last_login_ip'"`
	LastLoginAt time.Time `json:"lastLoginAt" xorm:"'last_login_at'"`
	CreatedAt   time.Time `json:"createdAt" xorm:"created 'created_at'"`
	UpdatedAt   time.Time `json:"updatedAt" xorm:"updated 'updated_at'"`

	LoginFailTimes int64      `json:"loginFailTimes" xorm:"notnull default 0 'login_fail_times'"`
	LoginUnlockAt  *time.Time `json:"loginUnlockAt" xorm:"'login_unlock_at'"`

	RoleKey string `json:"roleKey,omitempty" xorm:"role_key"`
	RoleId  int    `json:"roleId,omitempty" xorm:"role_id"`

	IsSuper bool `json:"isSuper,omitempty" xorm:"-"`
}

func (t *Admin_RoleId) TableName() string {
	return "admin"
}

type Admin_R struct {
	ID       int    `json:"id" xorm:"pk autoincr 'id'"`
	Username string `json:"username" xorm:"'username'"`
	Name     string `json:"name" xorm:"'name'"`
	Mobile   string `json:"mobile" xorm:"'mobile'"`
	Sex      int    `json:"sex" xorm:"'sex'"`
	Birthday string `json:"birthday" xorm:"'birthday'"`
	Avatar   string `json:"avatar" xorm:"'avatar'"`
	Password string `json:"-" xorm:"'password'"`
	Salt     string `json:"-" xorm:"'salt'"`
	Status   int    `json:"status" xorm:"'status'"`

	LastLoginIP string    `json:"lastLoginIP" xorm:"'last_login_ip'"`
	LastLoginAt time.Time `json:"lastLoginAt" xorm:"'last_login_at'"`
	CreatedAt   time.Time `json:"createdAt" xorm:"created 'created_at'"`
	UpdatedAt   time.Time `json:"updatedAt" xorm:"updated 'updated_at'"`

	LoginFailTimes int64      `json:"loginFailTimes" xorm:"notnull default 0 'login_fail_times'"`
	LoginUnlockAt  *time.Time `json:"loginUnlockAt" xorm:"'login_unlock_at'"`

	Role        *AuthRole         `json:"role,omitempty" xorm:"-"`
	RoleId      int               `json:"roleId,omitempty" xorm:"role_id"`
	RoleKey     string            `json:"roleKey,omitempty" xorm:"role_key"`
	Permissions []*AuthPermission `json:"permissions,omitempty" xorm:"-"`

	LoginLocked      bool  `json:"loginLocked" xorm:"-"`
	LoginWaitSeconds int64 `json:"loginWaitSeconds" xorm:"-"`
}

func (t *Admin_R) TableName() string {
	return "admin"
}
