package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"net/http"

	m "git.containerum.net/ch/resource-service/pkg/router/middleware"
	"git.containerum.net/ch/resource-service/pkg/server"
	kubtypes "github.com/containerum/kube-client/pkg/model"
)

type IngressHandlers struct {
	server.IngressActions
	*m.TranslateValidate
}

func (h *IngressHandlers) GetIngressesListHandler(ctx *gin.Context) {
	resp, err := h.GetIngressesList(ctx.Request.Context(), ctx.Param("ns_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *IngressHandlers) GetIngressHandler(ctx *gin.Context) {
	resp, err := h.GetIngress(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("ingress"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *IngressHandlers) CreateIngressHandler(ctx *gin.Context) {
	var req kubtypes.Ingress
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	createdIngress, err := h.CreateIngress(ctx.Request.Context(), ctx.Param("ns_label"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusCreated, createdIngress)
}

func (h *IngressHandlers) UpdateIngressHandler(ctx *gin.Context) {
	var req kubtypes.Ingress
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	updatedIngress, err := h.UpdateIngress(ctx.Request.Context(), ctx.Param("ns_label"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusAccepted, updatedIngress)
}

func (h *IngressHandlers) DeleteIngressHandler(ctx *gin.Context) {
	if err := h.DeleteIngress(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("ingress")); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}
