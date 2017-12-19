package httpapi

import (
	"regexp"

	"git.containerum.net/ch/resource-service/server"

	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()
var DNSLabel = regexp.MustCompile(`[a-z0-9]([-a-z0-9]*[a-z0-9])?`)

type CreateResourceRequest struct {
	TariffID string `json:"tariff-id"`
	Label    string `json:"label"`
}

type RenameResourceRequest struct {
	New string `json:"label"`
}

type SetResourceLockRequest struct {
	Lock *bool `json:"lock"`
}

type SetResourceAccessRequest struct {
	UserID string `json:"user_id"`
	Access string `json:"access"`
}

func initializeContext(srv server.ResourceSvcInterface) gin.HandlerFunc {
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

// depends on initializeContext
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
		WithField("actor-user-id", userID).
		WithField("user-role", userRole).
		WithField("token-id", tokenID)
	c.Set("logger", logger)
}

// adminAction checks whether the request is performed by the
// ‘admin’ account. If so, it substitutes user ID from query
// parameters, if present, and sets "admin-action" context
// field.
//
// depends on parseHeaders
func adminAction(c *gin.Context) {
	logger := c.MustGet("logger").(*logrus.Entry)

	if c.MustGet("user-role").(string) == "admin" {
		if qpUserID, exists := c.GetQuery("user-id"); exists {
			c.Set("user-id", qpUserID)
			logger = logger.WithField("user-id", qpUserID)
		}
		c.Set("admin-action", true)
		logger = logger.WithField("admin-action", true)
	} else {
		c.Set("admin-action", false)
		logger = logger.WithField("admin-action", false)
	}

	c.Set("logger", logger)
}

func parseCreateResourceReq(c *gin.Context) {
	var req CreateResourceRequest
	log := c.MustGet("logger").(*logrus.Entry)
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Infof("failed to json-bind request data: %v", err)
		c.AbortWithStatusJSON(400, map[string]string{
			"error": "0x03",
			"errcode": "BAD_INPUT",
		})
	}
	log = log.WithField("request-data-type", "CreateResourceRequest")
	c.Set("request-data", req)
	c.Set("logger", log)
}

func parseRenameReq(c *gin.Context) {
	var req RenameResourceRequest
	log := c.MustGet("logger").(*logrus.Entry)
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Infof("failed to json-bind request data: %v", err)
		c.AbortWithStatusJSON(400, map[string]string{
			"error": "0x03",
			"errcode": "BAD_INPUT",
		})
	}
	log = log.WithField("request-data-type", "RenameResourceRequest")
	c.Set("request-data", req)
	c.Set("logger", log)
}

func parseLockReq(c *gin.Context) {
	var req SetResourceLockRequest
	log := c.MustGet("logger").(*logrus.Entry)
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Infof("failed to json-bind request data: %v", err)
		c.AbortWithStatusJSON(400, map[string]string{
			"error": "0x03",
			"errcode": "BAD_INPUT",
		})
	}
	log = log.WithField("request-data-type", "SetResourceLockRequest")
	c.Set("request-data", req)
	c.Set("logger", log)
}

func parseSetAccessReq(c *gin.Context) {
	var req SetResourceAccessRequest
	log := c.MustGet("logger").(*logrus.Entry)
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Infof("failed to json-bind request data: %v", err)
		c.AbortWithStatusJSON(400, map[string]string{
			"error": "0x03",
			"errcode": "BAD_INPUT",
		})
	}
	log = log.WithField("request-data-type", "SetResourceAccessRequest")
	c.Set("request-data", req)
	c.Set("logger", log)
}

func rejectUnprivileged(c *gin.Context) {
	admin := c.MustGet("admin-action").(bool)
	if !admin {
		c.AbortWithStatusJSON(401, map[string]string{
			"errcode": "PERMISSION_DENIED",
			"error":   "denied",
		})
	}
}

func serverErrorResponse(err error) (code int, obj map[string]interface{}) {
	code = 500
	obj = make(map[string]interface{})

	switch err {
	case server.ErrNoSuchResource:
		code = 404
		obj["errcode"] = server.ErrNoSuchResource.ErrCode
	case server.ErrAlreadyExists:
		code = 422
		obj["errcode"] = server.ErrAlreadyExists.ErrCode
	case server.ErrDenied:
		code = 401
		obj["errcode"] = server.ErrDenied.ErrCode
	default:
		switch etyped := err.(type) {
		case server.Err:
			code = 500
			obj["errcode"] = etyped.ErrCode
		case server.BadInputError:
			code = 400
			obj["errcode"] = etyped.Err.ErrCode
		case server.OtherServiceError:
			code = 503
			obj["errcode"] = etyped.Err.ErrCode
		case server.PermissionError:
			code = 401
			obj["errcode"] = etyped.Err.ErrCode
		}
	}

	obj["error"] = "0x03"

	return
}
