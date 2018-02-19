package routes

import (
	"github.com/gin-gonic/gin"

	"net/http"

	rstypes "git.containerum.net/ch/json-types/resource-service"
)

func createStorageHandler(ctx *gin.Context) {
	var req rstypes.CreateStorageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}

	if err := srv.CreateStorage(ctx.Request.Context(), req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func getStoragesHandler(ctx *gin.Context) {
	resp, err := srv.GetStorages(ctx.Request.Context())
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func updateStorageHandler(ctx *gin.Context) {
	var req rstypes.UpdateStorageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}

	if err := srv.UpdateStorage(ctx.Request.Context(), ctx.Param("storage_name"), req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func deleteStorageHandler(ctx *gin.Context) {
	if err := srv.DeleteStorage(ctx.Request.Context(), ctx.Param("storage_name")); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}
