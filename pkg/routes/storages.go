package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"net/http"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/pkg/server"
)

type StorageHandlers struct {
	server.StorageActions
	*TranslateValidate
}

func (h *StorageHandlers) CreateStorageHandler(ctx *gin.Context) {
	var req rstypes.CreateStorageRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	if err := h.CreateStorage(ctx.Request.Context(), req); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func (h *StorageHandlers) GetStoragesHandler(ctx *gin.Context) {
	resp, err := h.GetStorages(ctx.Request.Context())
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *StorageHandlers) UpdateStorageHandler(ctx *gin.Context) {
	var req rstypes.UpdateStorageRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	if err := h.UpdateStorage(ctx.Request.Context(), ctx.Param("storage_name"), req); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *StorageHandlers) DeleteStorageHandler(ctx *gin.Context) {
	if err := h.DeleteStorage(ctx.Request.Context(), ctx.Param("storage_name")); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}
