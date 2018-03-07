package routes

import (
	"net/http"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func addDomainHandler(ctx *gin.Context) {
	var req rstypes.AddDomainRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(badRequest(ctx, err))
		return
	}

	if err := srv.AddDomain(ctx.Request.Context(), req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func getAllDomainsHandler(ctx *gin.Context) {
	var params rstypes.GetAllDomainsQueryParams
	if err := ctx.ShouldBindWith(&params, binding.Form); err != nil {
		ctx.AbortWithStatusJSON(badRequest(ctx, err))
		return
	}

	resp, err := srv.GetAllDomains(ctx.Request.Context(), params)
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func getDomainHandler(ctx *gin.Context) {
	resp, err := srv.GetDomain(ctx.Request.Context(), ctx.Param("domain"))
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func deleteDomainHandler(ctx *gin.Context) {
	if err := srv.DeleteDomain(ctx.Request.Context(), ctx.Param("domain")); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}
