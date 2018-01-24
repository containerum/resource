package routes

import (
	"git.containerum.net/ch/utils"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(app *gin.Engine /*server*/) {

	app.Use(utils.SaveHeaders)
	app.Use(utils.PrepareContext)

	ns := app.Group("/namespace")
	{
		ns.POST("", substituteUserMiddleware, namespaceCreateHandler)
	}
}
