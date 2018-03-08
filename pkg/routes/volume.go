package routes

import (
	"net/http"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func createVolumeHandler(ctx *gin.Context) {
	var req rstypes.CreateVolumeRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(badRequest(ctx, err))
		return
	}

	if err := srv.CreateVolume(ctx.Request.Context(), req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func deleteUserVolumeHandler(ctx *gin.Context) {
	if err := srv.DeleteUserVolume(ctx.Request.Context(), ctx.Param("vol_label")); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func deleteAllUserVolumesHandler(ctx *gin.Context) {
	if err := srv.DeleteAllUserVolumes(ctx.Request.Context()); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func getUserVolumesHandler(ctx *gin.Context) {
	vols, err := srv.GetUserVolumes(ctx.Request.Context(), ctx.Query("filters"))
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	utils.MaskForNonAdmin(ctx, vols)

	ctx.JSON(http.StatusOK, vols)
}

func getUserVolumeHandler(ctx *gin.Context) {
	vol, err := srv.GetUserVolume(ctx.Request.Context(), ctx.Param("vol_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	utils.MaskForNonAdmin(ctx, &vol)

	ctx.JSON(http.StatusOK, vol)
}

func getAllVolumesHandler(ctx *gin.Context) {
	var params rstypes.GetAllResourcesQueryParams
	if err := ctx.ShouldBindWith(&params, binding.Form); err != nil {
		ctx.AbortWithStatusJSON(badRequest(ctx, err))
		return
	}

	vols, err := srv.GetAllVolumes(ctx.Request.Context(), params)
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, vols)
}

func renameUserVolumeHandler(ctx *gin.Context) {
	var req rstypes.RenameVolumeRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(badRequest(ctx, err))
		return
	}
	if err := srv.RenameUserVolume(ctx.Request.Context(), ctx.Param("vol_label"), req.NewLabel); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func resizeUserVolumeHandler(ctx *gin.Context) {
	var req rstypes.ResizeVolumeRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(badRequest(ctx, err))
	}
	if err := srv.ResizeUserVolume(ctx.Request.Context(), ctx.Param("vol_label"), req.NewTariffID); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func getVolumesLinkedWithUserNamespaceHandler(ctx *gin.Context) {
	resp, err := srv.GetVolumesLinkedWithUserNamespace(ctx.Request.Context(), ctx.Param("vol_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	utils.MaskForNonAdmin(ctx, &resp)

	ctx.JSON(http.StatusOK, resp)
}