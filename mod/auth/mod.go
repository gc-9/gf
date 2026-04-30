package auth

import (
	"github.com/gc-9/gf/auth"
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/mod/auth/auth_controllers"
	"github.com/gc-9/gf/mod/auth/auth_services"
	"github.com/redis/go-redis/v9"
)

var (
	Routers = []any{
		auth_controllers.NewAdminController,
		auth_controllers.NewPermissionController,
		auth_controllers.NewRoleController,
		auth_controllers.NewPassportController,
		auth_controllers.NewOperationLogController,
	}

	Services = []any{
		func(servConf *config.Server, redisClient *redis.Client, encryptService *auth.EncryptService) *auth.AuthService {
			return auth.NewAuthService(servConf.Auth, redisClient, encryptService)
		},
		auth.NewCaptcha2Service,
		auth_services.NewPermissionService,
		auth_services.NewRoleService,
		auth_services.NewAdminService,
	}

	BootInvoke = []any{
		// update permissions
		func(service *auth_services.PermissionService, servConf *config.Server) error {
			return service.UpdateAclPermissions(servConf)
		},
	}
)
