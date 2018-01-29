package routes

import (
	"net/http"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"github.com/gin-gonic/gin"
)

func createVolumeHandler(ctx *gin.Context) {
	var req rstypes.CreateVolumeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}

	if err := srv.CreateNamespace(ctx.Request.Context(), &req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func deleteUserVolumeHandler(ctx *gin.Context) {
	if err := srv.DeleteUserVolume(ctx.Request.Context(), ctx.Param("label")); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func deleteAllUserVolumes(ctx *gin.Context) {
	if err := srv.DeleteAllUserVolumes(ctx.Request.Context()); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}
