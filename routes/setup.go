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
		ns.POST("", createNamespaceHandler)

		ns.GET("", getUserNamespacesHandler)
		ns.GET("/:label", getUserNamespaceHandler)
		ns.GET("/:label/access", utils.RequireAdminRole, getUserNamespaceAccessesHandler)

		ns.DELETE("/:label", deleteUserNamespaceHandler)

		ns.PUT("/:label/name", renameUserNamespaceHandler)
		ns.PUT("/:label/access", utils.RequireAdminRole, setUserNamespaceAccessHandler)
		ns.PUT("/:label", resizeNamespaceHandler)
	}

	nss := app.Group("/namespaces")
	{
		nss.GET("", utils.RequireAdminRole, getAllNamespacesHandler)

		nss.DELETE("", utils.RequireAdminRole, deleteAllUserNamespacesHandler)
	}

	vol := app.Group("/volume")
	{
		vol.POST("", createVolumeHandler)

		vol.GET("", getUserVolumesHandler)
		vol.GET("/:label", getUserVolumeHandler)

		vol.DELETE("/:label", deleteUserVolumeHandler)
	}

	vols := app.Group("/volumes")
	{
		vols.GET("", utils.RequireAdminRole, getAllVolumesHandler)

		vols.DELETE("", utils.RequireAdminRole, deleteAllUserVolumesHandler)
	}
}
