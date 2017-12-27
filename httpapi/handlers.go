package httpapi

import (
	"context"

	"git.containerum.net/ch/resource-service/server"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// *** NAMESPACES ***

func CreateNamespace(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	adminAction := c.MustGet("admin-action").(bool)
	reqData := c.MustGet("request-data").(CreateResourceRequest)

	logger.Infof("creating namespace %s", reqData.Label)
	err := srv.CreateNamespace(c.Request.Context(), userID, reqData.Label, reqData.TariffID, adminAction)
	if err != nil {
		logger.Errorf("failed to create namespace %s: %v", reqData.Label, err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
	}
}

func DeleteNamespace(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	nsLabel := c.Param("namespace")

	logger.Infof("deleting namespace %s", nsLabel)
	err := srv.DeleteNamespace(c.Request.Context(), userID, nsLabel)
	if err != nil {
		logger.Errorf("failed to delete namespace %s: %v", nsLabel, err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
	}
}

func ListNamespaces(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	adminAction := c.MustGet("admin-action").(bool)

	logger.Infof("listing namespaces")
	nss, err := srv.ListNamespaces(c.Request.Context(), userID, adminAction)
	if err != nil {
		logger.Errorf("failed to list namespaces: %v", err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
		return
	}
	c.IndentedJSON(200, nss)
}

func GetNamespace(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	adminAction := c.MustGet("admin-action").(bool)
	nsLabel := c.Param("namespace")

	logger.Infof("getting namespace %s", nsLabel)
	nss, err := srv.GetNamespace(c.Request.Context(), userID, nsLabel, adminAction)
	if err != nil {
		logger.Errorf("failed to get namespace %s: %v", nsLabel, err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
		return
	}
	c.IndentedJSON(200, nss)
}

func RenameNamespace(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	nsLabel := c.Param("namespace")
	reqData := c.MustGet("request-data").(RenameResourceRequest)

	if reqData.New == "" || !DNSLabel.MatchString(reqData.New) {
		logger.Warnf("invalid new label: empty or does not match DNS_LABEL: %q", reqData.New)
		c.AbortWithStatusJSON(400, map[string]interface{}{
			"error":   "0x03",
			"errcode": "BAD_INPUT",
		})
		return
	}

	logger.Infof("renaming namespace %s to %s user %s", nsLabel, reqData.New, userID)
	err := srv.RenameNamespace(c.Request.Context(), userID, nsLabel, reqData.New)
	if err != nil {
		logger.Errorf("failed to rename namespace %s into %s: %v", nsLabel, reqData.New, err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
	}
}

func SetNamespaceLock(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	label := c.Param("namespace")
	reqData := c.MustGet("request-data").(SetResourceLockRequest)

	if reqData.Lock == nil {
		logger.Warnf("invalid input: missing field \"lock\"")
		c.AbortWithStatusJSON(400, map[string]interface{}{
			"error":   "0x03",
			"errcode": "BAD_INPUT",
		})
		return
	}

	if *reqData.Lock {
		logger.Infof("locking namespace %s user %s", label, userID)
	} else {
		logger.Infof("unlocking namespace %s user %s", label, userID)
	}
	err := srv.LockNamespace(c.Request.Context(), userID, label, *reqData.Lock)
	if err != nil {
		logger.Errorf("failed to lock access to namespace %s: %v", label, err)
		code, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(code, respObj)
	}
}

func SetNamespaceAccess(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	label := c.Param("namespace")
	reqData := c.MustGet("request-data").(SetResourceAccessRequest)

	logger.Infof("setting access level %s to user %s on namespace %s of user %s",
		reqData.Access, reqData.UserID, label, userID)
	err := srv.ChangeNamespaceAccess(c.Request.Context(), userID, label, reqData.UserID, reqData.Access)
	if err != nil {
		logger.Errorf("failed to set access to namespace %s: %v", label, err)
		code, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(code, respObj)
	}
}

// *** VOLUMES ***

func CreateVolume(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	adminAction := c.MustGet("admin-action").(bool)
	reqData := c.MustGet("request-data").(CreateResourceRequest)

	logger.Infof("creating volume %s", reqData.Label)
	err := srv.CreateVolume(c.Request.Context(), userID, reqData.Label, reqData.TariffID, adminAction)
	if err != nil {
		logger.Warnf("failed to create volume %s: %v", reqData.Label, err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
	}
}

func DeleteVolume(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	label := c.Param("volume")

	logger.Infof("deleting volume %s", label)
	err := srv.DeleteVolume(c.Request.Context(), userID, label)
	if err != nil {
		logger.Errorf("failed to delete volume %s: %v", label, err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
	}
}

func ListVolumes(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	adminAction := c.MustGet("admin-action").(bool)

	logger.Infof("listing volumes")
	vols, err := srv.ListVolumes(c.Request.Context(), userID, adminAction)
	if err != nil {
		logger.Errorf("failed to list volumes: %v", err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
		return
	}
	c.IndentedJSON(200, vols)
}

func GetVolume(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	adminAction := c.MustGet("admin-action").(bool)
	label := c.Param("volume")

	logger.Infof("getting volume %s", label)
	vols, err := srv.GetVolume(c.Request.Context(), userID, label, adminAction)
	if err != nil {
		logger.Errorf("failed to get volume %s: %v", label, err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
		return
	}
	c.IndentedJSON(200, vols)
}

func RenameVolume(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	label := c.Param("volume")
	reqData := c.MustGet("request-data").(RenameResourceRequest)

	if reqData.New == "" || !DNSLabel.MatchString(reqData.New) {
		logger.Warnf("invalid new label: empty or does not match DNS_LABEL: %q", reqData.New)
		c.AbortWithStatusJSON(400, map[string]interface{}{
			"error":   "0x03",
			"errcode": "BAD_INPUT",
		})
		return
	}

	logger.Infof("rename volume %s to %s user %s", label, reqData.New, userID)
	err := srv.RenameVolume(c.Request.Context(), userID, label, reqData.New)
	if err != nil {
		logger.Errorf("failed to rename volume %s into %s: %v", label, reqData.New, err)
		status, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(status, respObj)
	}
}

func SetVolumeLock(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	label := c.Param("volume")
	reqData := c.MustGet("request-data").(SetResourceLockRequest)

	if reqData.Lock == nil {
		logger.Warnf("invalid input: missing field \"lock\"")
		c.AbortWithStatusJSON(400, map[string]interface{}{
			"error": "0x03",
		})
		return
	}

	if *reqData.Lock {
		logger.Infof("lock volume %s user %s", label, userID)
	} else {
		logger.Infof("unlock volume %s user %s", label, userID)
	}
	err := srv.LockVolume(c.Request.Context(), userID, label, *reqData.Lock)
	if err != nil {
		logger.Errorf("failed to lock access to volume %s: %v", label, err)
		code, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(code, respObj)
	}
}

func SetVolumeAccess(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	label := c.Param("volume")
	reqData := c.MustGet("request-data").(SetResourceAccessRequest)

	logger.Infof("setting access level %s to user %s on volume %s of user %s",
		reqData.Access, reqData.UserID, label, userID)
	err := srv.ChangeVolumeAccess(c.Request.Context(), userID, label, reqData.UserID, reqData.Access)
	if err != nil {
		logger.Errorf("failed to set access to volume %s: %v", label, err)
		code, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(code, respObj)
	}
}

func ListAllNamespaces(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	ctx := c.MustGet("request-context").(context.Context)

	logger.Info("list all namespaces")
	nsch, err := srv.ListAllNamespaces(ctx)
	if err != nil {
		logger.Errorf("failed to list all namespaces: %v", err)
		code, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(code, respObj)
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
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	ctx := c.MustGet("request-context").(context.Context)

	logger.Info("list all volumes")
	vch, err := srv.ListAllVolumes(ctx)
	if err != nil {
		logger.Errorf("failed to list all volumes: %v", err)
		code, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(code, respObj)
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
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	log := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	reqData := c.MustGet("request-data").(CreateResourceRequest)
	reqData.Label = c.Param("namespace")

	log.Infof("resize namespace: user=%s label=%s tariff=%s", userID, reqData.Label, reqData.TariffID)
	err := srv.ResizeNamespace(c.Request.Context(), userID, reqData.Label, reqData.TariffID)
	if err != nil {
		log.Errorf("failed to resize namespace %s: %v", reqData.Label, err)
		code, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(code, respObj)
	}
}
