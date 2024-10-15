package controllers

import (
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/httpLib"
	"github.com/gc-9/gf/mod/admin/services"
	adminTypes "github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/types"
	"github.com/samber/lo"
	"xorm.io/builder"
	"xorm.io/xorm"
)

func NewAttachmentController(attachmentService *services.AttachmentService) httpLib.Router {
	return &AttachmentController{
		attachmentService: attachmentService,
	}
}

type AttachmentController struct {
	attachmentService *services.AttachmentService
}

func (p *AttachmentController) Routes() []*httpLib.Route {
	return []*httpLib.Route{
		httpLib.NewRoute("POST", "/sys/attachment/index", "附件-列表", p.Index),
		httpLib.NewRoute("POST", "/sys/attachment/store", "附件-保存", p.Store),
		httpLib.NewRoute("POST", "/sys/attachment/destroy", "附件-删除", p.Destroy),
	}
}

func (p *AttachmentController) Index(param *types.ParamPageQuery) (*types.PagerData[*adminTypes.AttachmentItem], error) {
	param.Filters.Allows([]string{
		"filename",
		"uid",
	})
	pagerData, err := p.attachmentService.PagerData(param.ParamPager, func(query *xorm.Session) {
		param.Filters.QueryOption()
		for k, f := range param.Filters {
			switch k {
			case "filename":
				query.Where(builder.Like{k, param.Filters.ValueString(k)})
			default:
				query.Where(builder.Eq{k: f})
			}
		}
	})
	if err != nil {
		return nil, err
	}

	itemList := lo.Map(pagerData.List, func(item *adminTypes.Attachment, index int) *adminTypes.AttachmentItem {
		v := &adminTypes.AttachmentItem{Attachment: item}
		v.Url = p.attachmentService.Url(item.Path)
		return v
	})

	return &types.PagerData[*adminTypes.AttachmentItem]{
		List:  itemList,
		Pager: pagerData.Pager,
	}, nil
}

func (p *AttachmentController) Store(ctx httpLib.RequestContext) (*adminTypes.AttachmentItem, error) {
	fh, err := ctx.FormFile("filedata")
	if err != nil {
		return nil, errors.New("文件为空")
	}

	authUser := ctx.AuthUser().(*adminTypes.Admin_RoleId)
	return p.attachmentService.Store(authUser.ID, fh)
}

type paramAttachmentDestroy struct {
	ID         int  `json:"id" form:"id" query:"id" validate:"required"`
	FullDelete bool `json:"fullDelete" form:"fullDelete" query:"fullDelete"`
}

func (p *AttachmentController) Destroy(param *paramAttachmentDestroy) (err error) {
	if param.FullDelete {
		_, err = p.attachmentService.FullDestroy(param.ID)
		return err
	} else {
		_, err = p.attachmentService.Destroy(param.ID)
		return err
	}
}
