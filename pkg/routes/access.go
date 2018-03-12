package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/utils"
	"github.com/gin-gonic/gin/binding"
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
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(badRequest(ctx, err))
		return
	}

	if err := srv.SetUserAccesses(ctx.Request.Context(), req.Access); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func setUserNamespaceAccessHandler(ctx *gin.Context) {
	var req rstypes.SetNamespaceAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(badRequest(ctx, err))
		return
	}

	if err := srv.SetUserNamespaceAccess(ctx.Request.Context(), ctx.Param("ns_label"), &req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func setUserVolumeAccessHandler(ctx *gin.Context) {
	var req rstypes.SetVolumeAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(badRequest(ctx, err))
		return
	}
	if err := srv.SetUserVolumeAccess(ctx.Request.Context(), ctx.Param("vol_label"), &req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func getUserNamespaceAccessesHandler(ctx *gin.Context) {
	resp, err := srv.GetUserNamespaceAccesses(ctx.Request.Context(), ctx.Param("ns_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	utils.MaskForNonAdmin(ctx, &resp)

	ctx.JSON(http.StatusOK, resp)
}

func getUserVolumeAccessesHandler(ctx *gin.Context) {
	resp, err := srv.GetUserVolumeAccesses(ctx.Request.Context(), ctx.Param("vol_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	utils.MaskForNonAdmin(ctx, &resp)

	ctx.JSON(http.StatusOK, resp)
}

func deleteUserNamespaceAccessHandler(ctx *gin.Context) {
	var req rstypes.DeleteNamespaceAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	if err := srv.DeleteUserNamespaceAccess(ctx.Request.Context(), ctx.Param("ns_label"), req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func deleteUserVolumeAccessHandler(ctx *gin.Context) {
	var req rstypes.DeleteVolumeAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	if err := srv.DeleteUserVolumeAccess(ctx.Request.Context(), ctx.Param("vol_label"), req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}
