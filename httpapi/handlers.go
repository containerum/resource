package httpapi

import (
	"bitbucket.org/exonch/resource-manager/server"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func CreateNamespace(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceManagerInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	//userRole := c.MustGet("user-role").(string)
	//tokenID := c.MustGet("token-id").(string)
	tariffID := c.MustGet("tariff-id").(string)
	adminAction := c.MustGet("admin-action").(bool)
	nsLabel := c.Param("namespace")

	logger.Infof("creating namespace %s", nsLabel)
	err := srv.CreateNamespace(c.Request.Context(), userID, nsLabel, tariffID, adminAction)
	if err != nil {
		logger.Errorf("failed to create namespace %s: %v", nsLabel, err)
		c.AbortWithStatusJSON(500, map[string]string{
			"error": "0x03",
		})
	}
}

func DeleteNamespace(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceManagerInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	//userRole := c.MustGet("user-role").(string)
	//tokenID := c.MustGet("token-id").(string)
	nsLabel := c.Param("namespace")

	logger.Infof("deleting namespace %s", nsLabel)
	err := srv.DeleteNamespace(c.Request.Context(), userID, nsLabel)
	if err != nil {
		logger.Errorf("failed to delete namespace %s: %v", nsLabel, err)
		c.AbortWithStatusJSON(500, map[string]string{
			"error": "0x03",
		})
	}
}

func ListNamespaces(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceManagerInterface)
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
