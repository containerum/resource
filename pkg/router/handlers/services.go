package handlers

import (
	"net/http"

	m "git.containerum.net/ch/resource-service/pkg/router/middleware"
	"git.containerum.net/ch/resource-service/pkg/server"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type ServiceHandlers struct {
	server.ServiceActions
	*m.TranslateValidate
}

func (h *ServiceHandlers) GetServicesListHandler(ctx *gin.Context) {
	resp, err := h.GetServices(ctx.Request.Context(), ctx.Param("ns_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *ServiceHandlers) GetServiceHandler(ctx *gin.Context) {
	resp, err := h.GetService(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("service_label"))

	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *ServiceHandlers) CreateServiceHandler(ctx *gin.Context) {
	var req kubtypes.Service
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	createdService, err := h.CreateService(ctx.Request.Context(), ctx.Param("ns_label"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusCreated, createdService)
}

func (h *ServiceHandlers) UpdateServiceHandler(ctx *gin.Context) {
	var req kubtypes.Service
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	req.Name = ctx.Param("service_label")
	updatedService, err := h.UpdateService(ctx.Request.Context(), ctx.Param("ns_label"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, updatedService)
}

func (h *ServiceHandlers) DeleteServiceHandler(ctx *gin.Context) {
	err := h.DeleteService(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("service_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}
