package httpapi

import (
	"encoding/json"

	"bitbucket.org/exonch/resource-service/server"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func CreateNamespace(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	adminAction := c.MustGet("admin-action").(bool)
	var reqData struct {
		TariffID string `json:"tariff-id"`
		Label    string `json:"label"`
	}
	data, err := c.GetRawData()
	if err != nil {
		logger.Warnf("gin.Context.GetRawData: %v", err)
		c.AbortWithStatusJSON(400, map[string]string{
			"error": "0x03",
		})
	}
	err = json.Unmarshal(data, &reqData)
	if err != nil {
		logger.Warnf("cannot unmarshal request data: %v", err)
		c.AbortWithStatusJSON(400, map[string]string{
			"error": "0x03",
		})
	}

	logger.Infof("creating namespace %s", reqData.Label)
	err = srv.CreateNamespace(c.Request.Context(), userID, reqData.Label, reqData.TariffID, adminAction)
	if err != nil {
		logger.Errorf("failed to create namespace %s: %v", reqData.Label, err)
		c.AbortWithStatusJSON(500, map[string]string{
			"error": "0x03",
		})
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
		c.AbortWithStatusJSON(500, map[string]string{
			"error": "0x03",
		})
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
		c.AbortWithStatusJSON(500, map[string]string{
			"error": "0x03",
		})
		return
	}
	c.IndentedJSON(200, nss)
}

func RenameNamespace(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	nsLabel := c.Param("namespace")

	var reqData struct {
		New string `json:"label"`
	}

	data, err := c.GetRawData()
	if err != nil {
		logger.Warnf("gin.Context.GetRawData: %v", err)
		c.AbortWithStatusJSON(400, map[string]interface{}{
			"error": "0x03",
		})
		return
	}
	err = json.Unmarshal(data, &reqData)
	if err != nil {
		logger.Warnf("failed to unmarshal request data: %v", err)
		c.AbortWithStatusJSON(400, map[string]interface{}{
			"error": "0x03",
		})
		return
	}
	if reqData.New == "" || !DNSLabel.MatchString(reqData.New) {
		logger.Warnf("invalid new label: empty or does not match DNS_LABEL: %q", reqData.New)
		c.AbortWithStatusJSON(400, map[string]interface{}{
			"error":   "0x03",
			"errcode": "BAD_INPUT",
		})
		return
	}

	err = srv.RenameNamespace(c.Request.Context(), userID, nsLabel, reqData.New)
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
	nsLabel := c.Param("namespace")

	var reqData struct {
		Lock *bool `json:"lock"`
	}

	data, err := c.GetRawData()
	if err != nil {
		logger.Warnf("gin.Context.GetRawData: %v", err)
		c.AbortWithStatusJSON(400, map[string]interface{}{
			"error": "0x03",
		})
		return
	}
	err = json.Unmarshal(data, &reqData)
	if err != nil {
		logger.Warnf("failed to unmarshal request data: %v", err)
		c.AbortWithStatusJSON(400, map[string]interface{}{
			"error": "0x03",
		})
		return
	}
	if reqData.Lock == nil {
		logger.Warnf("invalid input: missing field \"lock\"")
		c.AbortWithStatusJSON(400, map[string]interface{}{
			"error":"0x03",
		})
		return
	}

	err = srv.LockNamespace(c.Request.Context(), userID, nsLabel, *reqData.Lock)
	if err != nil {
		logger.Errorf("failed to lock access to namespace %s: %v", err)
		code, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(code, respObj)
	}
}

func SetNamespaceAccess(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	nsLabel := c.Param("namespace")

	var reqData struct {
		UserID string `json:"user_id"`
		Access string `json:"access"`
	}

	data, err := c.GetRawData()
	if err != nil {
		logger.Warnf("gin.Context.GetRawData: %v", err)
		c.AbortWithStatusJSON(400, map[string]interface{}{
			"error": "0x03",
		})
		return
	}
	err = json.Unmarshal(data, &reqData)
	if err != nil {
		logger.Warnf("failed to unmarshal request data: %v", err)
		c.AbortWithStatusJSON(400, map[string]interface{}{
			"error": "0x03",
		})
		return
	}

	err = srv.ChangeNamespaceAccess(c.Request.Context(), userID, nsLabel, reqData.UserID, reqData.Access)
	if err != nil {
		logger.Errorf("failed to lock access to namespace %s: %v", err)
		code, respObj := serverErrorResponse(err)
		c.AbortWithStatusJSON(code, respObj)
	}
}

func CreateVolume(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceSvcInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	adminAction := c.MustGet("admin-action").(bool)

	var reqData struct {
		TariffID string `json:"tariff-id"`
		Label    string `json:"label"`
	}
	data, err := c.GetRawData()
	if err != nil {
		logger.Warnf("gin.Context.GetRawData: %v", err)
		c.AbortWithStatusJSON(400, map[string]string{
			"error": "0x03",
		})
	}
	err = json.Unmarshal(data, &reqData)
	if err != nil {
		logger.Warnf("cannot unmarshal request data: %v", err)
		c.AbortWithStatusJSON(400, map[string]string{
			"error": "0x03",
		})
	}

	logger.Infof("creating volume %s", reqData.Label)
	err = srv.CreateVolume(c.Request.Context(), userID, reqData.Label, reqData.TariffID, adminAction)
	if err != nil {
		logger.Warnf("failed to create volume %s: %v", reqData.Label, err)
		c.AbortWithStatusJSON(500, map[string]string{
			"error": "0x03",
		})
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
		c.AbortWithStatusJSON(500, map[string]string{
			"error": "0x03",
		})
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
		c.AbortWithStatusJSON(500, map[string]string{
			"error": "0x03",
		})
		return
	}
	c.IndentedJSON(200, vols)
}
