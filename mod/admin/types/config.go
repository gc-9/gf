package types

import "time"

type Config struct {
	ID        int    `json:"id" xorm:"pk autoincr 'id'"`
	GroupName string `json:"groupName" xorm:"'group_name'"`
	Name      string `json:"name" xorm:"'name'"`
	Key       string `json:"key" xorm:"'key'"`
	Value     string `json:"value" xorm:"'value'"`
	Type      string `json:"type" xorm:"'type'"`

	CreatedAt time.Time `json:"createdAt" xorm:"created 'created_at'"`
	UpdatedAt time.Time `json:"updatedAt" xorm:"updated 'updated_at'"`
}

func (t *Config) TableName() string {
	return "config"
}
