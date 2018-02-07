package routes

import (
	"net/http"

	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"github.com/gin-gonic/gin"
)

func createDeploymentHandler(ctx *gin.Context) {
	var req kubtypes.Deployment
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}

	if err := srv.CreateDeployment(ctx.Request.Context(), ctx.Param("ns_label"), req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}
	ctx.Status(http.StatusOK)
}

func getDeploymentsHandler(ctx *gin.Context) {
	resp, err := srv.GetDeployments(ctx.Request.Context(), ctx.Param("ns_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func getDeploymentByLabelHandler(ctx *gin.Context) {
	resp, err := srv.GetDeploymentByLabel(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("deploy_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func deleteDeploymentByLabelHandler(ctx *gin.Context) {
	// TODO
}

func setContainerImageHandler(ctx *gin.Context) {
	// TODO
}

func replaceDeploymentHandler(ctx *gin.Context) {
	// TODO
}

func setReplicasHandler(ctx *gin.Context) {
	// TODO
}
