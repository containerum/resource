package handlers

import (
	"net/http"

	m "git.containerum.net/ch/resource-service/pkg/router/middleware"
	"git.containerum.net/ch/resource-service/pkg/server"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type DeployHandlers struct {
	server.DeployActions
	*m.TranslateValidate
}

func (h *DeployHandlers) GetDeploymentsListHandler(ctx *gin.Context) {
	resp, err := h.GetDeploymentsList(ctx.Request.Context(), ctx.Param("ns_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *DeployHandlers) GetDeploymentHandler(ctx *gin.Context) {
	resp, err := h.GetDeployment(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("deploy_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *DeployHandlers) CreateDeploymentHandler(ctx *gin.Context) {
	var req kubtypes.Deployment

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	deploy, err := h.CreateDeployment(ctx.Request.Context(), ctx.Param("ns_label"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}
	ctx.JSON(http.StatusOK, deploy)
}

func (h *DeployHandlers) UpdateDeploymentHandler(ctx *gin.Context) {
	var req kubtypes.Deployment
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	req.Name = ctx.Param("deploy_label")
	updDeploy, err := h.UpdateDeployment(ctx.Request.Context(), ctx.Param("ns_label"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusAccepted, updDeploy)
}

func (h *DeployHandlers) SetContainerImageHandler(ctx *gin.Context) {
	var req kubtypes.UpdateImage
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	updatedDeploy, err := h.SetDeploymentContainerImage(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("deploy_label"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusAccepted, updatedDeploy)
}

func (h *DeployHandlers) SetReplicasHandler(ctx *gin.Context) {
	var req kubtypes.UpdateReplicas
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}
	updatedDeploy, err := h.SetDeploymentReplicas(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("deploy_label"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusAccepted, updatedDeploy)
}

func (h *DeployHandlers) DeleteDeploymentByLabelHandler(ctx *gin.Context) {
	err := h.DeleteDeployment(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("deploy_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}
