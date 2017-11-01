package httpapi

import (
	"bitbucket.org/exonch/resource-manager/server"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

var logger = logrus.New()

func initializeContext(srv server.ResourceManagerInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.NewV4().String()
		c.Header("x-request-id", requestID)
		c.Set("request-id", requestID)
		c.Set("logger", logrus.NewEntry(logger).
			WithField("client-ip", c.ClientIP()).
			WithField("request-id", requestID).
			WithField("http-method", c.Request.Method).
			WithField("http-uri", c.Request.RequestURI))
		c.Set("server", srv)
	}
}

func parseHeaders(c *gin.Context) {
	userID := c.Request.Header.Get("x-user-id")
	userRole := c.Request.Header.Get("x-user-role")
	tokenID := c.Request.Header.Get("x-user-token-id")

	c.Set("user-id", userID)
	c.Set("user-role", userRole)
	c.Set("token-id", tokenID)

	logger := c.MustGet("logger").(*logrus.Entry)
	logger = logger.
		WithField("user-id", userID).
		WithField("user-role", userRole).
		WithField("token-id", tokenID)
	c.Set("logger", logger)
}
