package handlers

import (
	"net/http"

	rstypes "git.containerum.net/ch/resource-service/pkg/model"
	"git.containerum.net/ch/resource-service/pkg/models/domain"
	m "git.containerum.net/ch/resource-service/pkg/router/middleware"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type DomainHandlers struct {
	server.DomainActions
	*m.TranslateValidate
}

func (h *DomainHandlers) GetDomainsListHandler(ctx *gin.Context) {
	var params rstypes.GetAllDomainsQueryParams
	if err := ctx.ShouldBindWith(&params, binding.Form); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	resp, err := h.GetDomainsList(ctx.Request.Context())
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

func (h *DomainHandlers) AddDomainHandler(ctx *gin.Context) {
	var req domain.Domain
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	domain, err := h.AddDomain(ctx.Request.Context(), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusCreated, domain)
}

func (h *DomainHandlers) DeleteDomainHandler(ctx *gin.Context) {
	if err := h.DeleteDomain(ctx.Request.Context(), ctx.Param("domain")); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}
