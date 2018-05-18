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

// swagger:operation GET /namespaces/{namespace}/deployments Deployment GetDeploymentsListHandler
// Get deployments list.
//
// ---
// x-method-visibility: public
// parameters:
//  - $ref: '#/parameters/UserIDHeader'
//  - $ref: '#/parameters/UserRoleHeader'
//  - $ref: '#/parameters/UserNamespaceHeader'
//  - $ref: '#/parameters/UserVolumeHeader'
//  - name: namespace
//    in: path
//    type: string
//    required: true
// responses:
//  '200':
//    description: deployments list
//    schema:
//      $ref: '#/definitions/DeploymentList'
//  default:
//    $ref: '#/responses/error'
func (h *DeployHandlers) GetDeploymentsListHandler(ctx *gin.Context) {
	resp, err := h.GetDeploymentsList(ctx.Request.Context(), ctx.Param("namespace"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// swagger:operation GET /namespaces/{namespace}/deployments/{deployment} Deployment GetDeploymentHandler
// Get deployment.
//
// ---
// x-method-visibility: public
// parameters:
//  - $ref: '#/parameters/UserIDHeader'
//  - $ref: '#/parameters/UserRoleHeader'
//  - $ref: '#/parameters/UserNamespaceHeader'
//  - $ref: '#/parameters/UserVolumeHeader'
//  - name: namespace
//    in: path
//    type: string
//    required: true
//  - name: deployment
//    in: path
//    type: string
//    required: true
// responses:
//  '200':
//    description: deployment
//    schema:
//      $ref: '#/definitions/Deployment'
//  default:
//    $ref: '#/responses/error'
func (h *DeployHandlers) GetDeploymentHandler(ctx *gin.Context) {
	resp, err := h.GetDeployment(ctx.Request.Context(), ctx.Param("namespace"), ctx.Param("deployment"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// swagger:operation POST /namespaces/{namespace}/deployments Deployment CreateDeploymentHandler
// Create deployment.
//
// ---
// x-method-visibility: public
// parameters:
//  - $ref: '#/parameters/UserIDHeader'
//  - $ref: '#/parameters/UserRoleHeader'
//  - $ref: '#/parameters/UserNamespaceHeader'
//  - $ref: '#/parameters/UserVolumeHeader'
//  - name: namespace
//    in: path
//    type: string
//    required: true
//  - name: body
//    in: body
//    schema:
//      $ref: '#/definitions/Deployment'
// responses:
//  '201':
//    description: deployment created
//    schema:
//      $ref: '#/definitions/Deployment'
//  default:
//    $ref: '#/responses/error'
func (h *DeployHandlers) CreateDeploymentHandler(ctx *gin.Context) {
	var req kubtypes.Deployment

	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	deploy, err := h.CreateDeployment(ctx.Request.Context(), ctx.Param("namespace"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}
	ctx.JSON(http.StatusCreated, deploy)
}

// swagger:operation PUT /namespaces/{namespace}/deployments/{deployment} Deployment UpdateDeployment
// Update deployment.
//
// ---
// x-method-visibility: public
// parameters:
//  - $ref: '#/parameters/UserIDHeader'
//  - $ref: '#/parameters/UserRoleHeader'
//  - $ref: '#/parameters/UserNamespaceHeader'
//  - $ref: '#/parameters/UserVolumeHeader'
//  - name: namespace
//    in: path
//    type: string
//    required: true
//  - name: deployment
//    in: path
//    type: string
//    required: true
//  - name: body
//    in: body
//    schema:
//      $ref: '#/definitions/Deployment'
// responses:
//  '202':
//    description: deployment updated
//    schema:
//      $ref: '#/definitions/Deployment'
//  default:
//    $ref: '#/responses/error'
func (h *DeployHandlers) UpdateDeploymentHandler(ctx *gin.Context) {
	var req kubtypes.Deployment
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	req.Name = ctx.Param("deployment")
	updDeploy, err := h.UpdateDeployment(ctx.Request.Context(), ctx.Param("namespace"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusAccepted, updDeploy)
}

// swagger:operation PUT /namespaces/{namespace}/deployments/{deployment}/image Deployment SetContainerImageHandler
// Update image in deployments container.
//
// ---
// x-method-visibility: public
// parameters:
//  - $ref: '#/parameters/UserIDHeader'
//  - $ref: '#/parameters/UserRoleHeader'
//  - $ref: '#/parameters/UserNamespaceHeader'
//  - $ref: '#/parameters/UserVolumeHeader'
//  - name: namespace
//    in: path
//    type: string
//    required: true
//  - name: deployment
//    in: path
//    type: string
//    required: true
//  - name: body
//    in: body
//    schema:
//      $ref: '#/definitions/UpdateImage'
// responses:
//  '202':
//    description: deployment updated
//    schema:
//      $ref: '#/definitions/Deployment'
//  default:
//    $ref: '#/responses/error'
func (h *DeployHandlers) SetContainerImageHandler(ctx *gin.Context) {
	var req kubtypes.UpdateImage
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	updatedDeploy, err := h.SetDeploymentContainerImage(ctx.Request.Context(), ctx.Param("namespace"), ctx.Param("deployment"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusAccepted, updatedDeploy)
}

// swagger:operation PUT /namespaces/{namespace}/deployments/{deployment}/replicas Deployment SetReplicasHandler
// Update deployments replicas count.
//
// ---
// x-method-visibility: public
// parameters:
//  - $ref: '#/parameters/UserIDHeader'
//  - $ref: '#/parameters/UserRoleHeader'
//  - $ref: '#/parameters/UserNamespaceHeader'
//  - $ref: '#/parameters/UserVolumeHeader'
//  - name: namespace
//    in: path
//    type: string
//    required: true
//  - name: deployment
//    in: path
//    type: string
//    required: true
//  - name: body
//    in: body
//    schema:
//      $ref: '#/definitions/UpdateReplicas'
// responses:
//  '202':
//    description: deployment updated
//    schema:
//      $ref: '#/definitions/Deployment'
//  default:
//    $ref: '#/responses/error'
func (h *DeployHandlers) SetReplicasHandler(ctx *gin.Context) {
	var req kubtypes.UpdateReplicas
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}
	updatedDeploy, err := h.SetDeploymentReplicas(ctx.Request.Context(), ctx.Param("namespace"), ctx.Param("deployment"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusAccepted, updatedDeploy)
}

// swagger:operation DELETE /namespaces/{namespace}/deployments/{deployment} Deployment DeleteDeploymentHandler
// Delete deployment.
//
// ---
// x-method-visibility: public
// parameters:
//  - $ref: '#/parameters/UserIDHeader'
//  - $ref: '#/parameters/UserRoleHeader'
//  - $ref: '#/parameters/UserNamespaceHeader'
//  - $ref: '#/parameters/UserVolumeHeader'
//  - name: namespace
//    in: path
//    type: string
//    required: true
//  - name: deployment
//    in: path
//    type: string
//    required: true
// responses:
//  '202':
//    description: deployment deleted
//  default:
//    $ref: '#/responses/error'
func (h *DeployHandlers) DeleteDeploymentHandler(ctx *gin.Context) {
	err := h.DeleteDeployment(ctx.Request.Context(), ctx.Param("namespace"), ctx.Param("deployment"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}

// swagger:operation DELETE /namespaces/{namespace}/deployments Deployment DeleteAllDeploymentsHandler
// Delete all deployments in namespace.
//
// ---
// x-method-visibility: private
// parameters:
//  - name: namespace
//    in: path
//    type: string
//    required: true
// responses:
//  '202':
//    description: all deployments in namespace deleted
//  default:
//    $ref: '#/responses/error'
func (h *DeployHandlers) DeleteAllDeploymentsHandler(ctx *gin.Context) {
	err := h.DeleteAllDeployments(ctx.Request.Context(), ctx.Param("namespace"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}
