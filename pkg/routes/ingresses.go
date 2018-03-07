package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"net/http"

	rstypes "git.containerum.net/ch/json-types/resource-service"
)

func createIngressHandler(ctx *gin.Context) {
	var req rstypes.CreateIngressRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(badRequest(ctx, err))
		return
	}

	if err := srv.CreateIngress(ctx.Request.Context(), ctx.Param("ns_label"), req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func getUserIngressesHandler(ctx *gin.Context) {
	var params rstypes.GetIngressesQueryParams
	if err := ctx.ShouldBindWith(&params, binding.Form); err != nil {
		ctx.AbortWithStatusJSON(badRequest(ctx, err))
		return
	}

	resp, err := srv.GetUserIngresses(ctx.Request.Context(), ctx.Param("ns_label"), params)
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func getAllIngressesHandler(ctx *gin.Context) {
	var params rstypes.GetIngressesQueryParams

	if err := ctx.ShouldBindWith(&params, binding.Form); err != nil {
		ctx.AbortWithStatusJSON(badRequest(ctx, err))
		return
	}

	resp, err := srv.GetAllIngresses(ctx.Request.Context(), params)
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func deleteIngressHandler(ctx *gin.Context) {
	if err := srv.DeleteIngress(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("domain")); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}
