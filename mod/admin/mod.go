package admin

import (
	"github.com/gc-9/gf/mod/admin/controllers"
	"github.com/gc-9/gf/mod/admin/services"
	"github.com/gc-9/gf/storage"
	"xorm.io/xorm"
)

var (
	Routers = []any{
		controllers.NewNoteController,
		controllers.NewAttachmentController,
		controllers.NewConfigController,
	}

	Services = []any{
		func(db *xorm.Engine, storage storage.Storage) *services.AttachmentService {
			return services.NewAttachmentService(db, "files", storage)
		},
		services.NewConfigService,
		services.NewEncryptService,
	}
)
