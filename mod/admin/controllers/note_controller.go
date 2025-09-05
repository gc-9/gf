package controllers

import (
	"github.com/gc-9/gf/crud"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/httplib"
	adminTypes "github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/types"
	"xorm.io/xorm"
)

func NewNoteController(db *xorm.Engine) httplib.Router {
	return &NoteController{
		crud: crud.NewCrudDB[adminTypes.Note](db),
	}
}

type NoteController struct {
	crud *crud.CrudDB[adminTypes.Note]
}

func (p *NoteController) Routes() []*httplib.Route {
	return []*httplib.Route{
		httplib.NewRoute("POST", "/sys/note/index", "备忘-列表", p.Index),
		httplib.NewRoute("POST", "/sys/note/show", "备忘-查看", p.Show),
		httplib.NewRoute("POST", "/sys/note/store", "备忘-保存", p.Store),
	}
}

func (p *NoteController) Index(param *types.ParamPageQuery) (*types.PagerData[*adminTypes.Note], error) {
	var allowFilters = []string{
		"title",
	}
	param.Filters.Allows(allowFilters)
	return p.crud.PagerData(param.ParamPager, param.Filters.QueryOption(), crud.QueryOrderBy("id desc"))
}

func (p *NoteController) Show(param *types.ParamID) (*adminTypes.Note, error) {
	item, err := p.crud.Get(param.ID)
	if err == nil && item == nil {
		err = errors.New("notFound")
	}
	return item, nil
}

type paramNoticeStore struct {
	ID      int    `json:"id"`
	Title   string `json:"title" validate:"required"`
	Content string `json:"content" validate:"required"`
	Status  int    `json:"status"`
}

func (p *NoteController) Store(param *paramNoticeStore) (err error) {
	if param.ID > 0 {
		_, err = p.crud.Update(param.ID, &adminTypes.Note{
			Title:   param.Title,
			Content: param.Content,
			Status:  param.Status,
		})
		return err
	} else {
		_, err = p.crud.Create(&adminTypes.Note{
			Title:   param.Title,
			Content: param.Content,
			Status:  param.Status,
		})
		return err
	}
}
