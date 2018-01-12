package httpapi

import (
	"os"

	"git.containerum.net/ch/resource-service/server"

	"github.com/gin-gonic/gin"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"github.com/gin-gonic/gin/binding"
)

var srv server.ResourceSvcInterface

func NewGinEngine(srvarg server.ResourceSvcInterface) *gin.Engine {
	srv = srvarg

	rstypes.RegisterCustomTagsGin(binding.Validator)

	g := gin.New()

	g.Use(gin.Recovery())
	g.Use(gin.LoggerWithWriter(os.Stderr))
	g.Use(parseHeaders)
	g.Use(adminAction)

	g.POST("/namespace", parseCreateResourceReq, CreateNamespace)
	g.DELETE("/namespace/:namespace", DeleteNamespace)
	g.GET("/namespace", ListNamespaces)
	g.GET("/namespace/:namespace", GetNamespace)
	g.PUT("/namespace/:namespace/name", parseRenameReq, RenameNamespace)
	g.PUT("/namespace/:namespace/lock", parseLockReq, SetNamespaceLock)
	g.PUT("/namespace/:namespace/access", parseSetAccessReq, SetNamespaceAccess)
	g.GET("/namespace/:namespace/access", rejectUnprivileged, GetNamespaceAccesses)
	g.PUT("/namespace/:namespace", parseCreateResourceReq, ResizeNamespace)

	g.POST("/volume", parseCreateResourceReq, CreateVolume)
	g.DELETE("/volume/:volume", DeleteVolume)
	g.GET("/volume", ListVolumes)
	g.GET("/volume/:volume", GetVolume)
	g.PUT("/volume/:volume/name", parseRenameReq, RenameVolume)
	g.PUT("/volume/:volume/lock", parseLockReq, SetVolumeLock)
	g.PUT("/volume/:volume/access", parseSetAccessReq, SetVolumeAccess)
	g.GET("/volume/:volume/access", rejectUnprivileged, GetVolumeAccesses)
	g.PUT("/volume/:volume", parseCreateResourceReq, ResizeVolume)

	g.GET("/adm/namespaces", rejectUnprivileged, parseListAllResources, ListAllNamespaces)
	g.GET("/adm/volumes", rejectUnprivileged, parseListAllResources, ListAllVolumes)

	g.GET("", func(c *gin.Context) {
		c.IndentedJSON(200, map[string]interface{}{
			"service": "resource-service",
			"status":  "not implemented",
		})
	})

	return g
}
