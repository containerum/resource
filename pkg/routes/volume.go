package routes

import (
	"net/http"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/containerum/utils/httputil"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type VolumeHandlers struct {
	server.VolumeActions
	*TranslateValidate
}

func (h *VolumeHandlers) CreateVolumeHandler(ctx *gin.Context) {
	var req rstypes.CreateVolumeRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	if err := h.CreateVolume(ctx.Request.Context(), req); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *VolumeHandlers) DeleteUserVolumeHandler(ctx *gin.Context) {
	if err := h.DeleteUserVolume(ctx.Request.Context(), ctx.Param("vol_label")); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *VolumeHandlers) DeleteAllUserVolumesHandler(ctx *gin.Context) {
	if err := h.DeleteAllUserVolumes(ctx.Request.Context()); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *VolumeHandlers) GetUserVolumesHandler(ctx *gin.Context) {
	vols, err := h.GetUserVolumes(ctx.Request.Context(), ctx.Query("filters"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	httputil.MaskForNonAdmin(ctx, vols)

	ctx.JSON(http.StatusOK, vols)
}

func (h *VolumeHandlers) GetUserVolumeHandler(ctx *gin.Context) {
	vol, err := h.GetUserVolume(ctx.Request.Context(), ctx.Param("vol_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	httputil.MaskForNonAdmin(ctx, &vol)

	ctx.JSON(http.StatusOK, vol)
}

func (h *VolumeHandlers) GetAllVolumesHandler(ctx *gin.Context) {
	var params rstypes.GetAllResourcesQueryParams
	if err := ctx.ShouldBindWith(&params, binding.Form); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	vols, err := h.GetAllVolumes(ctx.Request.Context(), params)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, vols)
}

func (h *VolumeHandlers) RenameUserVolumeHandler(ctx *gin.Context) {
	var req rstypes.RenameVolumeRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}
	if err := h.RenameUserVolume(ctx.Request.Context(), ctx.Param("vol_label"), req.NewLabel); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *VolumeHandlers) ResizeUserVolumeHandler(ctx *gin.Context) {
	var req rstypes.ResizeVolumeRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
	}
	if err := h.ResizeUserVolume(ctx.Request.Context(), ctx.Param("vol_label"), req.NewTariffID); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *VolumeHandlers) GetVolumesLinkedWithUserNamespaceHandler(ctx *gin.Context) {
	resp, err := h.GetVolumesLinkedWithUserNamespace(ctx.Request.Context(), ctx.Param("vol_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	httputil.MaskForNonAdmin(ctx, &resp)

	ctx.JSON(http.StatusOK, resp)
}
