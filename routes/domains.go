package routes

import (
	"net/http"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"github.com/gin-gonic/gin"
)

func addDomainHandler(ctx *gin.Context) {
	var req rstypes.AddDomainRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
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
	if err := ctx.ShouldBindQuery(&params); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
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
