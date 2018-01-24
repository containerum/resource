package routes

import (
	"context"

	umtypes "git.containerum.net/ch/json-types/user-manager"
	"git.containerum.net/ch/utils"
	"github.com/gin-gonic/gin"
)

// replaces user id in context with user id from query if it set and user is admin
func substituteUserMiddleware(ctx *gin.Context) {
	role := ctx.GetHeader(umtypes.UserIDHeader)
	if userID, set := ctx.GetQuery("user-id"); set && role == "admin" {
		rctx := context.WithValue(ctx.Request.Context(), utils.UserIDContextKey, userID)
		ctx.Request = ctx.Request.WithContext(rctx)
	}
}
