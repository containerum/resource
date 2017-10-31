package httpapi

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func CreateNamespace(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceManagerInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	userRole := c.MustGet("user-role").(string)
	tokenID := c.MustGet("token-id").(string)
	nsLabel := c.Param("namespace")

	logger.Infof("creating namespace %s", nsLabel)
	err := srv.CreateNamespace(c.Request.Context(), userID, nsLabel, tariffID)
	if err != nil {
		logger.Errorf("failed to create namespace %s: %v", nsLabel, err)
		c.AbortWithStatusJSON(500, map[string]string{
			"error": "nope.",
		})
	}
}

func DeleteNamespace(c *gin.Context) {
	srv := c.MustGet("server").(server.ResourceManagerInterface)
	logger := c.MustGet("logger").(*logrus.Entry)
	userID := c.MustGet("user-id").(string)
	userRole := c.MustGet("user-role").(string)
	tokenID := c.MustGet("token-id").(string)
	nsLabel := c.Param("namespace")

	logger.Infof("deleting namespace %s", nsLabel)
	err := srv.DeleteNamespace(c.Request.Context(), userID, nsLabel)
	if err != nil {
		logger.Errorf("failed to delete namespace %s: %v", nsLabel, err)
		c.AbortWithStatusJSON(500, map[string]string{
			"error": "nope",
		})
	}
}
