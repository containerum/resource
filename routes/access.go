package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/utils"
)

func getUserResourceAccessesHandler(ctx *gin.Context) {
	resp, err := srv.GetUserAccesses(ctx.Request.Context())
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func setUserResourceAccessesHandler(ctx *gin.Context) {
	var req rstypes.SetResourcesAccessRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}

	if err := srv.SetUserAccesses(ctx, req.Access); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func setUserNamespaceAccessHandler(ctx *gin.Context) {
	var req rstypes.SetNamespaceAccessRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}

	if err := srv.SetUserNamespaceAccess(ctx.Request.Context(), ctx.Param("label"), &req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func setUserVolumeAccessHandler(ctx *gin.Context) {
	var req rstypes.SetVolumeAccessRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}
	if err := srv.SetUserVolumeAccess(ctx.Request.Context(), ctx.Param("label"), &req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func getUserNamespaceAccessesHandler(ctx *gin.Context) {
	resp, err := srv.GetUserNamespaceAccesses(ctx.Request.Context(), ctx.Param("label"))
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	utils.MaskForNonAdmin(ctx, &resp)

	ctx.JSON(http.StatusOK, resp)
}

func getUserVolumeAccessesHandler(ctx *gin.Context) {
	resp, err := srv.GetUserVolumeAccesses(ctx.Request.Context(), ctx.Param("label"))
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	utils.MaskForNonAdmin(ctx, &resp)

	ctx.JSON(http.StatusOK, resp)
}
