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

	ctx.JSON(http.StatusOK, vols)
}

func getUserVolumeHandler(ctx *gin.Context) {
	vol, err := srv.GetUserVolume(ctx.Request.Context(), ctx.Param("label"))
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, vol)
}

func getAllVolumesHandler(ctx *gin.Context) {
	var params rstypes.GetAllResourcesQueryParams
	if err := ctx.ShouldBindQuery(&params); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}

	vols, err := srv.GetAllVolumes(ctx.Request.Context(), &params)
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, vols)
}

func getUserVolumeAccessesHandler(ctx *gin.Context) {
	resp, err := srv.GetUserVolumeAccesses(ctx.Request.Context(), ctx.Param("label"))
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func renameUserVolumeHandler(ctx *gin.Context) {
	var req rstypes.RenameVolumeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}
	if err := srv.RenameUserVolume(ctx.Request.Context(), ctx.Param("label"), req.NewLabel); err != nil {
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
	if err := srv.SetUserVolumeAccess(ctx.Request.Context(), ctx.Param("label"), req.Access); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func resizeUserVolumeHandler(ctx *gin.Context) {
	var req rstypes.ResizeVolumeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
	}
	if err := srv.ResizeUserVolume(ctx.Request.Context(), ctx.Param("label"), req.NewTariffID); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}