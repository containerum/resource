package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	umtypes "git.containerum.net/ch/json-types/user-manager"
	"git.containerum.net/ch/resource-service/pkg/server"
	"git.containerum.net/ch/utils"
	"github.com/gin-gonic/gin/binding"
)

type AccessHandlers struct {
	server.AccessActions
	*TranslateValidate
}

func (h *AccessHandlers) GetUserResourceAccessesHandler(ctx *gin.Context) {
	resp, err := h.GetUserAccesses(ctx.Request.Context())
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *AccessHandlers) SetUserResourceAccessesHandler(ctx *gin.Context) {
	var req rstypes.SetResourcesAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	if err := h.SetUserAccesses(ctx.Request.Context(), req.Access); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *AccessHandlers) SetUserNamespaceAccessHandler(ctx *gin.Context) {
	var req rstypes.SetNamespaceAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	if err := h.SetUserNamespaceAccess(ctx.Request.Context(), ctx.Param("ns_label"), &req); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *AccessHandlers) SetUserVolumeAccessHandler(ctx *gin.Context) {
	var req rstypes.SetVolumeAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}
	if err := h.SetUserVolumeAccess(ctx.Request.Context(), ctx.Param("vol_label"), &req); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *AccessHandlers) GetUserNamespaceAccessesHandler(ctx *gin.Context) {
	resp, err := h.GetUserNamespaceAccesses(ctx.Request.Context(), ctx.Param("ns_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	utils.MaskForNonAdmin(ctx, resp)

	if ctx.GetHeader(umtypes.UserRoleHeader) != "admin" && resp.NewAccessLevel != "owner" {
		resp.Users = nil
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *AccessHandlers) GetUserVolumeAccessesHandler(ctx *gin.Context) {
	resp, err := h.GetUserVolumeAccesses(ctx.Request.Context(), ctx.Param("vol_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	utils.MaskForNonAdmin(ctx, &resp)

	ctx.JSON(http.StatusOK, resp)
}

func (h *AccessHandlers) DeleteUserNamespaceAccessHandler(ctx *gin.Context) {
	var req rstypes.DeleteNamespaceAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	if err := h.DeleteUserNamespaceAccess(ctx.Request.Context(), ctx.Param("ns_label"), req); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *AccessHandlers) DeleteUserVolumeAccessHandler(ctx *gin.Context) {
	var req rstypes.DeleteVolumeAccessRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	if err := h.DeleteUserVolumeAccess(ctx.Request.Context(), ctx.Param("vol_label"), req); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}
