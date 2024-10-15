package types

import "time"

type AdminLog struct {
	ID        int       `json:"id" xorm:"pk autoincr 'id'"`
	Uid       int       `json:"uid" xorm:"'uid'"`
	Rid       int       `json:"rid" xorm:"'rid'"`
	Method    string    `json:"method" xorm:"'method'"`
	Action    string    `json:"action" xorm:"'action'"`
	Data      string    `json:"data" xorm:"'data'"`
	Ip        string    `json:"ip" xorm:"'ip'"`
	UserAgent string    `json:"userAgent" xorm:"'user_agent'"`
	Remark    string    `json:"remark" xorm:"'remark'"`
	CreatedAt time.Time `json:"createdAt" xorm:"created 'created_at'"`
}

func (t *AdminLog) TableName() string {
	return "admin_log"
}

type AdminLogFull struct {
	ID        int       `json:"id" xorm:"pk autoincr 'id'"`
	Uid       int       `json:"uid" xorm:"'uid'"`
	Rid       int       `json:"rid" xorm:"'rid'"`
	Method    string    `json:"method" xorm:"'method'"`
	Action    string    `json:"action" xorm:"'action'"`
	Data      string    `json:"data" xorm:"'data'"`
	Ip        string    `json:"ip" xorm:"'ip'"`
	UserAgent string    `json:"userAgent" xorm:"'user_agent'"`
	Remark    string    `json:"remark" xorm:"'remark'"`
	CreatedAt time.Time `json:"createdAt" xorm:"created 'created_at'"`

	OpUsername string `json:"opUsername"` // 操作人用户名
	OpName     string `json:"opName"`     // 操作人姓名
	OpRoleName string `json:"opRoleName"`
}

func (t *AdminLogFull) TableName() string {
	return "admin_log"
}
