package httpapi

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

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
			"error":   "0x03",
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
			"error":   "0x03",
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
			"error":   "0x03",
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
			"error":   "0x03",
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

func parseListAllResources(c *gin.Context) {
	var err error
	log := c.MustGet("logger").(*logrus.Entry)
	ctx := c.Request.Context()

	if countstr := c.Query("count"); countstr != "" {
		count, err := strconv.Atoi(countstr)
		if count < 0 && err == nil {
			err = fmt.Errorf("less than zero")
		}
		if err != nil {
			log.Warnf("invalid integer in QP count: %v", err)
			c.AbortWithStatusJSON(400, map[string]string{
				"error":   `parsing query parameter "count": ` + err.Error(),
				"errcode": "BAD_INPUT",
			})
			return
		} else {
			ctx = context.WithValue(ctx, "count", uint(count))
		}
	} else {
		ctx = context.WithValue(ctx, "count", uint(20))
	}

	if orderstr := c.Query("order"); orderstr != "" {
		ctx = context.WithValue(ctx, "sort-direction", c.Query("order"))
	}

	if afterstr := c.Query("after"); afterstr != "" {
		var afterTime time.Time
		afterTime, err = time.Parse(time.RFC3339Nano, afterstr)
		if err != nil {
			log.Warnf("invalid timestamp in QP after: %v", err)
			c.AbortWithStatusJSON(400, map[string]string{
				"error":   `parsing query parameter "after": ` + err.Error(),
				"errcode": "BAD_INPUT",
			})
			return
		} else {
			ctx = context.WithValue(ctx, "after-time", afterTime)
		}
	}

	if boolstr := c.Query("deleted"); boolstr == "" {
		ctx = context.WithValue(ctx, "deleted", false)
	} else {
		b, err := strconv.ParseBool(boolstr)
		if err != nil {
			log.Warnf("invalid boolean in QP deleted: %v", err)
			c.AbortWithStatusJSON(400, map[string]string{
				"error":   `parsing boolean query parameter "deleted": ` + err.Error(),
				"errcode": "BAD_INPUT",
			})
			return
		}
		ctx = context.WithValue(ctx, "deleted", b)
	}

	if boolstr := c.Query("limited"); boolstr == "" {
		ctx = context.WithValue(ctx, "limited", false)
	} else {
		b, err := strconv.ParseBool(boolstr)
		if err != nil {
			log.Warnf("invalid boolean in QP limited: %v", err)
			c.AbortWithStatusJSON(400, map[string]string{
				"error":   `parsing boolean query parameter "limited": ` + err.Error(),
				"errcode": "BAD_INPUT",
			})
			return
		}
		ctx = context.WithValue(ctx, "limited", b)
	}

	c.Set("request-context", ctx)
}
