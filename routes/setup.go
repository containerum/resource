package routes

import (
	rstypes "git.containerum.net/ch/json-types/resource-service"
	umtypes "git.containerum.net/ch/json-types/user-manager"
	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

var srv server.ResourceService

func SetupRoutes(app *gin.Engine, server server.ResourceService) {
	srv = server

	app.Use(utils.SaveHeaders)
	app.Use(utils.PrepareContext)

	rstypes.RegisterCustomTagsGin(binding.Validator)

	ns := app.Group("/namespace")
	{
		ns.POST("",
			utils.RequireHeaders(umtypes.UserIDHeader, umtypes.UserRoleHeader),
			utils.RequireAdminRole,
			utils.SubstituteUserMiddleware,
			createNamespaceHandler)

		ns.GET("",
			utils.RequireHeaders(umtypes.UserIDHeader, umtypes.UserRoleHeader),
			utils.SubstituteUserMiddleware,
			getUserNamespaceHandler)
	}
}
