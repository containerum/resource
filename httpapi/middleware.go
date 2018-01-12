package httpapi

import (
	"context"
	"fmt"
	"strconv"
	"time"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	rserrors "git.containerum.net/ch/resource-service/server/errors"

	"git.containerum.net/ch/json-types/errors"
	"github.com/gin-gonic/gin"
)

func parseHeaders(c *gin.Context) {
	userID := c.Request.Header.Get("x-user-id")
	userRole := c.Request.Header.Get("x-user-role")
	tokenID := c.Request.Header.Get("x-user-token-id")

	c.Set("user-id", userID)
	c.Set("user-role", userRole)
	c.Set("token-id", tokenID)
}

// adminAction checks whether the request is performed by the
// ‘admin’ account. If so, it substitutes user ID from query
// parameters, if present, and sets "admin-action" context
// field.
//
// depends on parseHeaders
func adminAction(c *gin.Context) {
	if c.MustGet("user-role").(string) == "admin" {
		if qpUserID, exists := c.GetQuery("user-id"); exists {
			c.Set("user-id", qpUserID)
		}
		c.Set("admin-action", true)
	} else {
		c.Set("admin-action", false)
	}
}

func parseCreateResourceReq(c *gin.Context) {
	var req rstypes.CreateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(400, errors.New(err.Error()))
	}
	c.Set("request-data", req)
}

func parseRenameReq(c *gin.Context) {
	var req rstypes.RenameResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(400, errors.New(err.Error()))
	}
	c.Set("request-data", req)
}

func parseLockReq(c *gin.Context) {
	var req rstypes.SetResourceLockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(400, errors.New(err.Error()))
	}
	c.Set("request-data", req)
}

func parseSetAccessReq(c *gin.Context) {
	var req rstypes.SetResourceAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(400, errors.New(err.Error()))
	}
	c.Set("request-data", req)
}

func rejectUnprivileged(c *gin.Context) {
	admin := c.MustGet("admin-action").(bool)
	if !admin {
		c.AbortWithStatusJSON(401, errors.New("denied"))
	}
}

func serverErrorResponse(err error) (code int, resp *errors.Error) {
	code = 500

	switch err {
	case rserrors.ErrNoSuchResource:
		code = 404
	case rserrors.ErrAlreadyExists:
		code = 422
	case rserrors.ErrDenied:
		code = 401
	default:
		switch err.(type) {
		case *errors.Error:
			code = 500
		case *rserrors.BadInputError:
			code = 400
		case *rserrors.OtherServiceError:
			code = 503
		case *rserrors.PermissionError:
			code = 401
		}
	}

	resp = errors.New(err.Error())

	return
}

func parseListAllResources(c *gin.Context) {
	var err error
	ctx := c.Request.Context()

	if countstr := c.Query("count"); countstr != "" {
		count, err := strconv.Atoi(countstr)
		if count < 0 && err == nil {
			err = fmt.Errorf("less than zero")
		}
		if err != nil {
			c.AbortWithStatusJSON(400, errors.Format(`parsing query parameter "count": %v`, err))
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
			c.Error(err)
			c.AbortWithStatusJSON(400, errors.Format(`parsing query parameter "after": %v`, err))
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
			c.Error(err)
			c.AbortWithStatusJSON(400, errors.Format(`parsing boolean query parameter "deleted": %v`, err))
			return
		}
		ctx = context.WithValue(ctx, "deleted", b)
	}

	if boolstr := c.Query("limited"); boolstr == "" {
		ctx = context.WithValue(ctx, "limited", false)
	} else {
		b, err := strconv.ParseBool(boolstr)
		if err != nil {
			c.Error(err)
			c.AbortWithStatusJSON(400, errors.Format(`parsing boolean query parameter "limited": %v`, err))
			return
		}
		ctx = context.WithValue(ctx, "limited", b)
	}
}
