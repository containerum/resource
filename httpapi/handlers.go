package httpapi

import (
	rstypes "git.containerum.net/ch/json-types/resource-service"
	"github.com/gin-gonic/gin"
)

// *** NAMESPACES ***

func CreateNamespace(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	adminAction := c.MustGet("admin-action").(bool)
	reqData := c.MustGet("request-data").(rstypes.CreateResourceRequest)

	err := srv.CreateNamespace(c.Request.Context(), userID, reqData.Label, reqData.TariffID, adminAction)
	if err != nil {
		c.Error(err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
	}
}

func DeleteNamespace(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	nsLabel := c.Param("namespace")

	err := srv.DeleteNamespace(c.Request.Context(), userID, nsLabel)
	if err != nil {
		c.Error(err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
	}
}

func ListNamespaces(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	adminAction := c.MustGet("admin-action").(bool)

	nss, err := srv.ListNamespaces(c.Request.Context(), userID, adminAction)
	if err != nil {
		c.Error(err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
		return
	}
	c.IndentedJSON(200, nss)
}

func GetNamespace(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	adminAction := c.MustGet("admin-action").(bool)
	nsLabel := c.Param("namespace")

	nss, err := srv.GetNamespace(c.Request.Context(), userID, nsLabel, adminAction)
	if err != nil {
		c.Error(err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
		return
	}
	c.IndentedJSON(200, nss)
}

func RenameNamespace(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	nsLabel := c.Param("namespace")
	reqData := c.MustGet("request-data").(rstypes.RenameResourceRequest)

	err := srv.RenameNamespace(c.Request.Context(), userID, nsLabel, reqData.New)
	if err != nil {
		c.Error(err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
	}
}

func SetNamespaceLock(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	label := c.Param("namespace")
	reqData := c.MustGet("request-data").(rstypes.SetResourceLockRequest)

	err := srv.LockNamespace(c.Request.Context(), userID, label, reqData.Lock)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(serverErrorResponse(err))

	}
}

func SetNamespaceAccess(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	label := c.Param("namespace")
	reqData := c.MustGet("request-data").(rstypes.SetResourceAccessRequest)

	err := srv.ChangeNamespaceAccess(c.Request.Context(), userID, label, reqData.UserID, reqData.Access)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(serverErrorResponse(err))

	}
}

// *** VOLUMES ***

func CreateVolume(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	adminAction := c.MustGet("admin-action").(bool)
	reqData := c.MustGet("request-data").(rstypes.CreateResourceRequest)

	err := srv.CreateVolume(c.Request.Context(), userID, reqData.Label, reqData.TariffID, adminAction)
	if err != nil {
		c.Error(err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
	}
}

func DeleteVolume(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	label := c.Param("volume")

	err := srv.DeleteVolume(c.Request.Context(), userID, label)
	if err != nil {
		c.Error(err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
	}
}

func ListVolumes(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	adminAction := c.MustGet("admin-action").(bool)

	vols, err := srv.ListVolumes(c.Request.Context(), userID, adminAction)
	if err != nil {
		c.Error(err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
		return
	}
	c.IndentedJSON(200, vols)
}

func GetVolume(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	adminAction := c.MustGet("admin-action").(bool)
	label := c.Param("volume")

	vols, err := srv.GetVolume(c.Request.Context(), userID, label, adminAction)
	if err != nil {
		c.Error(err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
		return
	}
	c.IndentedJSON(200, vols)
}

func RenameVolume(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	label := c.Param("volume")
	reqData := c.MustGet("request-data").(rstypes.RenameResourceRequest)

	err := srv.RenameVolume(c.Request.Context(), userID, label, reqData.New)
	if err != nil {
		c.Error(err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
	}
}

func SetVolumeLock(c *gin.Context) {

	userID := c.MustGet("user-id").(string)
	label := c.Param("volume")
	reqData := c.MustGet("request-data").(rstypes.SetResourceLockRequest)

	err := srv.LockVolume(c.Request.Context(), userID, label, reqData.Lock)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(serverErrorResponse(err))

	}
}

func SetVolumeAccess(c *gin.Context) {

	userID := c.MustGet("user-id").(string)
	label := c.Param("volume")
	reqData := c.MustGet("request-data").(rstypes.SetResourceAccessRequest)

	err := srv.ChangeVolumeAccess(c.Request.Context(), userID, label, reqData.UserID, reqData.Access)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(serverErrorResponse(err))

	}
}

func ListAllNamespaces(c *gin.Context) {
	nsch, err := srv.ListAllNamespaces(c.Request.Context())
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(serverErrorResponse(err))

		return
	}
	c.Header("content-type", "application/json")
	c.String(200, "[\n")
	firstIter := true
	for ns := range nsch {
		if !firstIter {
			c.String(200, ",\n")
		} else {
			firstIter = !firstIter
		}
		c.IndentedJSON(200, ns)
	}
	c.String(200, "\n]")

}

func ListAllVolumes(c *gin.Context) {
	vch, err := srv.ListAllVolumes(c.Request.Context())
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(serverErrorResponse(err))

		return
	}
	c.Header("content-type", "application/json")
	c.String(200, "[\n")
	firstIter := true
	for v := range vch {
		if !firstIter {
			c.String(200, ",\n")
		} else {
			firstIter = false
		}
		c.IndentedJSON(200, v)
	}
	c.String(200, "\n]")
}

func ResizeNamespace(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	reqData := c.MustGet("request-data").(rstypes.CreateResourceRequest)
	reqData.Label = c.Param("namespace")

	err := srv.ResizeNamespace(c.Request.Context(), userID, reqData.Label, reqData.TariffID)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(serverErrorResponse(err))
	}
}

func ResizeVolume(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	reqData := c.MustGet("request-data").(rstypes.CreateResourceRequest)
	reqData.Label = c.Param("volume")

	err := srv.ResizeVolume(c.Request.Context(), userID, reqData.Label, reqData.TariffID)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(serverErrorResponse(err))
	}
}

func DeleteAllVolumes(c *gin.Context) {
	userID := c.MustGet("user-id").(string)

	err := srv.DeleteAllVolumes(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(serverErrorResponse(err))
	}
}

func GetNamespaceAccesses(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	label := c.Param("namespace")

	ns, err := srv.GetNamespaceAccesses(c.Request.Context(), userID, label)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(serverErrorResponse(err))
		return
	}
	c.IndentedJSON(200, ns)
}

func GetVolumeAccesses(c *gin.Context) {
	userID := c.MustGet("user-id").(string)
	label := c.Param("volume")

	ns, err := srv.GetVolumeAccesses(c.Request.Context(), userID, label)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(serverErrorResponse(err))
		return
	}
	c.IndentedJSON(200, ns)
}

func GetResourcesAccess(c *gin.Context) {
	userID := c.MustGet("user-id").(string)

	resp, err := srv.GetResourceAccess(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(serverErrorResponse(err))
		return
	}
	c.IndentedJSON(200, resp)
}
