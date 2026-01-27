package types

import (
	"encoding/json"
	"time"
)

type Config struct {
	ID        int    `json:"id" xorm:"pk autoincr 'id'"`
	GroupName string `json:"groupName" xorm:"'group_name'"`
	Name      string `json:"name" xorm:"'name'"`
	Key       string `json:"key" xorm:"'key'"`
	Value     string `json:"value" xorm:"'value'"`
	Type      string `json:"type" xorm:"'type'"`

	Options ConfigOptions `json:"options" xorm:"'options'"`

	CreatedAt time.Time `json:"createdAt" xorm:"created 'created_at'"`
	UpdatedAt time.Time `json:"updatedAt" xorm:"updated 'updated_at'"`
}

func (t *Config) TableName() string {
	return "config"
}

type ConfigOptions []any

func (t *ConfigOptions) FromDB(buf []byte) error {
	if len(buf) != 0 {
		var f ConfigOptions
		if err := json.Unmarshal(buf, &f); err != nil {
			return err
		}
		*t = f
	}
	return nil
}

func (t *ConfigOptions) ToDB() ([]byte, error) {
	if t == nil {
		return nil, nil
	}
	return json.Marshal(t)
}
