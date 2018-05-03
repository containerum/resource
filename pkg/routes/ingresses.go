package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"net/http"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/pkg/server"
	kubtypes "github.com/containerum/kube-client/pkg/model"
)

type IngressHandlers struct {
	server.IngressActions
	*TranslateValidate
}

func (h *IngressHandlers) CreateIngressHandler(ctx *gin.Context) {
	var req kubtypes.Ingress
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	if err := h.CreateIngress(ctx.Request.Context(), ctx.Param("ns_label"), req); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func (h *IngressHandlers) GetUserIngressesHandler(ctx *gin.Context) {
	var params rstypes.GetIngressesQueryParams
	if err := ctx.ShouldBindWith(&params, binding.Form); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	resp, err := h.GetUserIngresses(ctx.Request.Context(), ctx.Param("ns_label"), params)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *IngressHandlers) GetAllIngressesHandler(ctx *gin.Context) {
	var params rstypes.GetIngressesQueryParams

	if err := ctx.ShouldBindWith(&params, binding.Form); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	resp, err := h.GetAllIngresses(ctx.Request.Context(), params)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *IngressHandlers) DeleteIngressHandler(ctx *gin.Context) {
	if err := h.DeleteIngress(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("domain")); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}
