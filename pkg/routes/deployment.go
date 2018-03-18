package routes

import (
	"net/http"

	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type DeployHandlers struct {
	server.DeployActions
	*TranslateValidate
}

func (h *DeployHandlers) CreateDeploymentHandler(ctx *gin.Context) {
	var req kubtypes.Deployment

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	if err := h.CreateDeployment(ctx.Request.Context(), ctx.Param("ns_label"), req); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}
	ctx.Status(http.StatusOK)
}

func (h *DeployHandlers) GetDeploymentsHandler(ctx *gin.Context) {
	resp, err := h.GetDeployments(ctx.Request.Context(), ctx.Param("ns_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *DeployHandlers) GetDeploymentByLabelHandler(ctx *gin.Context) {
	resp, err := h.GetDeploymentByLabel(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("deploy_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *DeployHandlers) DeleteDeploymentByLabelHandler(ctx *gin.Context) {
	err := h.DeleteDeployment(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("deploy_label"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *DeployHandlers) SetContainerImageHandler(ctx *gin.Context) {
	var req kubtypes.UpdateImage
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	err := h.SetContainerImage(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("deploy_label"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *DeployHandlers) ReplaceDeploymentHandler(ctx *gin.Context) {
	var req kubtypes.Deployment
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	req.Name = ctx.Param("deploy_label")
	err := h.ReplaceDeployment(ctx.Request.Context(), ctx.Param("ns_label"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (h *DeployHandlers) SetReplicasHandler(ctx *gin.Context) {
	var req kubtypes.UpdateReplicas
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}
	err := h.SetDeploymentReplicas(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("deploy_label"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}
