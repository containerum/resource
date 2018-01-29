package routes

import (
	rstypes "git.containerum.net/ch/json-types/resource-service"
	"github.com/gin-gonic/gin"
)

func createVolumeHanler(ctx *gin.Context) {
	var req rstypes.CreateVolumeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}

	if err := srv.CreateNamespace(ctx, &req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	return
}
