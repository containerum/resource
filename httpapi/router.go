package httpapi

import (
	"os"

	"git.containerum.net/ch/resource-service/server"

	"github.com/gin-gonic/gin"
)

func NewGinEngine(srv server.ResourceSvcInterface) *gin.Engine {
	g := gin.New()
	g.Use(gin.Recovery())
	g.Use(gin.LoggerWithWriter(os.Stderr))
	g.Use(initializeContext(srv))
	g.Use(parseHeaders)
	g.Use(adminAction)

	g.POST("/namespace", parseCreateResourceReq, CreateNamespace)
	g.DELETE("/namespace/:namespace", DeleteNamespace)
	g.GET("/namespace", ListNamespaces)
	g.GET("/namespace/:namespace", GetNamespace)
	g.PUT("/namespace/:namespace/name", parseRenameReq, RenameNamespace)
	g.PUT("/namespace/:namespace/lock", parseLockReq, SetNamespaceLock)
	g.PUT("/namespace/:namespace/access", parseSetAccessReq, SetNamespaceAccess)

	g.POST("/volume", parseCreateResourceReq, CreateVolume)
	g.DELETE("/volume/:volume", DeleteVolume)
	g.GET("/volume", ListVolumes)
	g.GET("/volume/:volume", GetVolume)
	g.PUT("/volume/:volume/name", parseRenameReq, RenameVolume)
	g.PUT("/volume/:volume/lock", parseLockReq, SetVolumeLock)
	g.PUT("/volume/:volume/access", parseSetAccessReq, SetVolumeAccess)

	g.GET("/adm/namespaces", rejectUnprivileged, ListAllNamespaces)
	g.GET("/adm/volumes", rejectUnprivileged, ListAllVolumes)

	return g
}
