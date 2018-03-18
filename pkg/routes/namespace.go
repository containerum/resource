package routes

import (
	"net/http"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/pkg/server"
	"git.containerum.net/ch/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type NamespaceHandlers struct {
	server.NamespaceActions
	*TranslateValidate
}

func (h *NamespaceHandlers) CreateNamespaceHandler(ctx *gin.Context) {
	var req rstypes.CreateNamespaceRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	if err := h.CreateNamespace(ctx.Request.Context(), req); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *NamespaceHandlers) GetUserNamespacesHandler(ctx *gin.Context) {
	resp, err := h.GetUserNamespaces(ctx.Request.Context(), ctx.Query("filters"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	utils.MaskForNonAdmin(ctx, resp)

	ctx.JSON(http.StatusOK, resp)
}

func (h *NamespaceHandlers) GetUserNamespaceHandler(ctx *gin.Context) {
	resp, err := h.GetUserNamespace(ctx.Request.Context(), ctx.Param("ns_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	utils.MaskForNonAdmin(ctx, &resp)

	ctx.JSON(http.StatusOK, resp)
}

func (h *NamespaceHandlers) GetAllNamespacesHandler(ctx *gin.Context) {
	var params rstypes.GetAllResourcesQueryParams
	if err := ctx.ShouldBindWith(&params, binding.Form); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	resp, err := h.GetAllNamespaces(ctx, params)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *NamespaceHandlers) DeleteUserNamespaceHandler(ctx *gin.Context) {
	if err := h.DeleteUserNamespace(ctx.Request.Context(), ctx.Param("ns_label")); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *NamespaceHandlers) DeleteAllUserNamespacesHandler(ctx *gin.Context) {
	if err := h.DeleteAllUserNamespaces(ctx.Request.Context()); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *NamespaceHandlers) RenameUserNamespaceHandler(ctx *gin.Context) {
	var req rstypes.RenameNamespaceRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	if err := h.RenameUserNamespace(ctx.Request.Context(), ctx.Param("ns_label"), req.NewLabel); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *NamespaceHandlers) ResizeUserNamespaceHandler(ctx *gin.Context) {
	var req rstypes.ResizeNamespaceRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	if err := h.ResizeUserNamespace(ctx.Request.Context(), ctx.Param("ns_label"), req.NewTariffID); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}
