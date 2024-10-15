package services

import (
	"github.com/gc-9/gf/crud"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/mod/admin/types"
	"github.com/shopspring/decimal"
	"xorm.io/xorm"
)

func NewConfigService(db *xorm.Engine) *ConfigService {
	base := crud.NewCrudDB[types.Config](db)
	return &ConfigService{
		CrudDB: base,
		db:     db,
	}
}

type ConfigService struct {
	*crud.CrudDB[types.Config]
	db *xorm.Engine
}

func (t *ConfigService) SetValue(key string, value string) (int64, error) {
	// update
	return t.db.Where("`key`=?", key).Cols("value").Update(&types.Config{Value: value})
}

func (t *ConfigService) GetValue(key string) (string, error) {
	conf, err := t.GetByOptions(func(query *xorm.Session) {
		query.Select("value").Where("`key`=?", key)
	})
	if err != nil {
		return "", err
	}
	if conf == nil {
		return "", errors.WithStackf("config '%s' not found", key)
	}
	return conf.Value, nil
}

func (t *ConfigService) GetValueDecimal(key string) (*decimal.Decimal, error) {
	v, err := t.GetValue(key)
	if err != nil {
		return nil, err
	}
	c, err := decimal.NewFromString(v)
	if err != nil {
		return nil, errors.Wrapf(err, "config '%s' decimal failed", key)
	}
	return &c, nil
}
