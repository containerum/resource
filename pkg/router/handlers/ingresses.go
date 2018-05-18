package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"net/http"

	m "git.containerum.net/ch/resource-service/pkg/router/middleware"
	"git.containerum.net/ch/resource-service/pkg/server"
	kubtypes "github.com/containerum/kube-client/pkg/model"
)

type IngressHandlers struct {
	server.IngressActions
	*m.TranslateValidate
}

// swagger:operation GET /namespaces/{namespace}/ingresses Ingress GetIngressesListHandler
// Get ingresses list.
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
//    description: ingresses list
//    schema:
//      $ref: '#/definitions/IngressList'
//  default:
//    $ref: '#/responses/error'
func (h *IngressHandlers) GetIngressesListHandler(ctx *gin.Context) {
	resp, err := h.GetIngressesList(ctx.Request.Context(), ctx.Param("namespace"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// swagger:operation GET /namespaces/{namespace}/ingresses/{ingress} Ingress GetIngressHandler
// Get ingresses list.
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
//  - name: ingress
//    in: path
//    type: string
//    required: true
// responses:
//  '200':
//    description: ingresses
//    schema:
//      $ref: '#/definitions/Ingress'
//  default:
//    $ref: '#/responses/error'
func (h *IngressHandlers) GetIngressHandler(ctx *gin.Context) {
	resp, err := h.GetIngress(ctx.Request.Context(), ctx.Param("namespace"), ctx.Param("ingress"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// swagger:operation POST /namespaces/{namespace}/ingresses Ingress CreateIngressHandler
// Create ingress.
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
//      $ref: '#/definitions/Ingress'
// responses:
//  '201':
//    description: ingress created
//    schema:
//      $ref: '#/definitions/Ingress'
//  default:
//    $ref: '#/responses/error'
func (h *IngressHandlers) CreateIngressHandler(ctx *gin.Context) {
	var req kubtypes.Ingress
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	createdIngress, err := h.CreateIngress(ctx.Request.Context(), ctx.Param("namespace"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusCreated, createdIngress)
}

// swagger:operation PUT /namespaces/{namespace}/ingresses/{ingress} Ingress UpdateIngressHandler
// Update ingress.
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
//  - name: ingress
//    in: path
//    type: string
//    required: true
//  - name: body
//    in: body
//    schema:
//      $ref: '#/definitions/Ingress'
// responses:
//  '202':
//    description: ingress updated
//    schema:
//      $ref: '#/definitions/Ingress'
//  default:
//    $ref: '#/responses/error'
func (h *IngressHandlers) UpdateIngressHandler(ctx *gin.Context) {
	var req kubtypes.Ingress
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	req.Name = ctx.Param("ingress")
	updatedIngress, err := h.UpdateIngress(ctx.Request.Context(), ctx.Param("namespace"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusAccepted, updatedIngress)
}

// swagger:operation DELETE /namespaces/{namespace}/ingresses/{ingress} Ingress DeleteIngressHandler
// Delete ingress.
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
//  - name: ingress
//    in: path
//    type: string
//    required: true
// responses:
//  '202':
//    description: ingress deleted
//  default:
//    $ref: '#/responses/error'
func (h *IngressHandlers) DeleteIngressHandler(ctx *gin.Context) {
	if err := h.DeleteIngress(ctx.Request.Context(), ctx.Param("namespace"), ctx.Param("ingress")); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}

// swagger:operation DELETE /namespaces/{namespace}/ingresses Ingress DeleteAllIngressesHandler
// Delete all ingresses.
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
//    description: all ingresses in namespace deleted
//  default:
//    $ref: '#/responses/error'
func (h *IngressHandlers) DeleteAllIngressesHandler(ctx *gin.Context) {
	if err := h.DeleteAllIngresses(ctx.Request.Context(), ctx.Param("namespace")); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}
