package routes

import (
	"net/http"

	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"github.com/gin-gonic/gin"
)

func createServiceHandler(ctx *gin.Context) {
	var req kubtypes.Service
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}

	if err := srv.CreateService(ctx.Request.Context(), ctx.Param("ns_label"), req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func getServicesHandler(ctx *gin.Context) {
	resp, err := srv.GetServices(ctx.Request.Context(), ctx.Param("ns_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}
