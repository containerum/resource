package httpapi

import (
	"git.containerum.net/ch/resource-service/server"

	"github.com/gin-gonic/gin"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"github.com/gin-gonic/gin/binding"
)

var srv server.ResourceSvcInterface

func SetupGinEngine(srvarg server.ResourceSvcInterface, g *gin.Engine) error {
	srv = srvarg

	if err := rstypes.RegisterCustomTagsGin(binding.Validator); err != nil {
		return err
	}

	g.Use(parseHeaders)
	g.Use(adminAction)

	ns := g.Group("/namespace")
	{
		ns.POST("", parseCreateResourceReq, CreateNamespace)
		ns.DELETE("", rejectUnprivileged, DeleteAllNamespaces)
		ns.DELETE("/:namespace", DeleteNamespace)
		ns.GET("", ListNamespaces)
		ns.GET("/:namespace", GetNamespace)
		ns.PUT("/:namespace/name", parseRenameReq, RenameNamespace)
		ns.PUT("/:namespace/lock", parseLockReq, SetNamespaceLock)
		ns.PUT("/:namespace/access", parseSetAccessReq, SetNamespaceAccess)
		ns.GET("/:namespace/access", rejectUnprivileged, GetNamespaceAccesses)
		ns.PUT("/:namespace", parseCreateResourceReq, ResizeNamespace)
	}

	vol := g.Group("/volume")
	{
		vol.POST("", parseCreateResourceReq, CreateVolume)
		vol.DELETE("", rejectUnprivileged, DeleteAllVolumes)
		vol.DELETE("/:volume", DeleteVolume)
		vol.GET("", ListVolumes)
		vol.GET("/:volume", GetVolume)
		vol.PUT("/:volume/name", parseRenameReq, RenameVolume)
		vol.PUT("/:volume/lock", parseLockReq, SetVolumeLock)
		vol.PUT("/:volume/access", parseSetAccessReq, SetVolumeAccess)
		vol.GET("/:volume/access", rejectUnprivileged, GetVolumeAccesses)
		vol.PUT("/:volume", parseCreateResourceReq, ResizeVolume)
	}

	adm := g.Group("/adm")
	{
		adm.GET("/namespaces", rejectUnprivileged, parseListAllResources, ListAllNamespaces)
		adm.GET("/volumes", rejectUnprivileged, parseListAllResources, ListAllVolumes)
	}

	g.GET("/access", GetResourcesAccess)

	g.GET("", func(c *gin.Context) {
		c.IndentedJSON(200, map[string]interface{}{
			"service": "resource-service",
			"status":  "not implemented",
		})
	})

	return nil
}
