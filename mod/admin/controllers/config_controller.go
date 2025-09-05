package controllers

import (
	"github.com/gc-9/gf/crud"
	"github.com/gc-9/gf/httplib"
	adminTypes "github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/types"
	"xorm.io/xorm"
)

func NewConfigController(db *xorm.Engine) httplib.Router {
	return &ConfigController{
		crud: crud.NewCrudDB[adminTypes.Config](db),
	}
}

type ConfigController struct {
	crud *crud.CrudDB[adminTypes.Config]
}

func (p *ConfigController) Routes() []*httplib.Route {
	return []*httplib.Route{
		httplib.NewRoute("POST", "/sys/config/all", "系统配置-全部", p.Index),
		httplib.NewRoute("POST", "/sys/config/store", "系统配置-保存", p.Store),
	}
}

func (p *ConfigController) Index() (*types.PagerData[*adminTypes.Config], error) {
	list, err := p.crud.List()
	if err != nil {
		return nil, err
	}
	return &types.PagerData[*adminTypes.Config]{
		List: list,
	}, nil
}

type paramConfigStore struct {
	ID    int    `json:"id"`
	Value string `json:"value" validate:"required"`
}

func (p *ConfigController) Store(param *paramConfigStore) (err error) {
	_, err = p.crud.Update(param.ID, &adminTypes.Config{
		Value: param.Value,
	})
	return err
}
