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

// SetupRoutes sets up a router
func SetupRoutes(app *gin.Engine, server server.ResourceService) {
	srv = server

	app.Use(utils.SaveHeaders)
	app.Use(utils.PrepareContext)
	app.Use(utils.RequireHeaders(umtypes.UserIDHeader, umtypes.UserRoleHeader))
	app.Use(utils.SubstituteUserMiddleware)

	rstypes.RegisterCustomTagsGin(binding.Validator)

	ns := app.Group("/namespace")
	{
		ns.POST("", utils.RequireAdminRole, createNamespaceHandler)

		ns.GET("", getUserNamespacesHandler)
		ns.GET("/:label", getUserNamespaceHandler)
		ns.GET("/:label/access", utils.RequireAdminRole, getUserNamespaceAccessesHandler)
	}

	app.GET("/namespaces", utils.RequireAdminRole, getAllNamespacesHandler)
}
