package admin

import (
	"github.com/gc-9/gf/mod/admin/controllers"
	"github.com/gc-9/gf/mod/admin/services"
)

var (
	Routers = []any{
		controllers.NewNoteController,
		controllers.NewAttachmentController,
		controllers.NewConfigController,
	}

	Services = []any{
		services.NewAttachmentService,
		services.NewConfigService,
		services.NewEncryptService,
	}
)
