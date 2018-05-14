package handlers

import (
	"net/http"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	m "git.containerum.net/ch/resource-service/pkg/router/middleware"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type DomainHandlers struct {
	server.DomainActions
	*m.TranslateValidate
}

func (h *DomainHandlers) AddDomainHandler(ctx *gin.Context) {
	var req rstypes.AddDomainRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	if err := h.AddDomain(ctx.Request.Context(), req); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func (h *DomainHandlers) GetAllDomainsHandler(ctx *gin.Context) {
	var params rstypes.GetAllDomainsQueryParams
	if err := ctx.ShouldBindWith(&params, binding.Form); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	resp, err := h.GetAllDomains(ctx.Request.Context(), params)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *DomainHandlers) GetDomainHandler(ctx *gin.Context) {
	resp, err := h.GetDomain(ctx.Request.Context(), ctx.Param("domain"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *DomainHandlers) DeleteDomainHandler(ctx *gin.Context) {
	if err := h.DeleteDomain(ctx.Request.Context(), ctx.Param("domain")); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}
