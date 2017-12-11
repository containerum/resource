package httpapi

import (
	"os"

	"bitbucket.org/exonch/resource-service/server"

	"github.com/gin-gonic/gin"
)

func NewGinEngine(srv server.ResourceSvcInterface) *gin.Engine {
	g := gin.New()
	g.Use(gin.Recovery())
	g.Use(gin.LoggerWithWriter(os.Stderr))
	g.Use(initializeContext(srv))
	g.Use(parseHeaders)
	g.Use(adminAction)

	g.POST("/namespace", CreateNamespace)
	g.DELETE("/namespace/:namespace", DeleteNamespace)
	g.GET("/namespace", ListNamespaces)
	g.GET("/namespace/:namespace", GetNamespace)
	g.PUT("/namespace/:namespace/name", RenameNamespace)
	g.PUT("/namespace/:namespace/lock", SetNamespaceLock)
	g.PUT("/namespace/:namespace/access", SetNamespaceAccess)

	g.POST("/volume", CreateVolume)
	g.DELETE("/volume/:volume", DeleteVolume)
	g.GET("/volume", ListVolumes)
	g.GET("/volume/:volume", GetVolume)
	g.PUT("/volume/:volume/name", RenameVolume)
	g.PUT("/volume/:volume/lock", SetVolumeLock)
	g.PUT("/volume/:volume/access", SetVolumeAccess)

	return g
}
