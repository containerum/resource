package handlers

import (
	"net/http"

	m "git.containerum.net/ch/resource-service/pkg/router/middleware"
	"git.containerum.net/ch/resource-service/pkg/server"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
)

type ConfigMapHandlers struct {
	server.ConfigMapActions
	*m.TranslateValidate
}

// swagger:operation GET /namespaces/{namespace}/configmaps ConfigMap GetConfigMapsListHandler
// Get configmaps list.
//
// ---
// x-method-visibility: public
// parameters:
//  - $ref: '#/parameters/UserIDHeader'
//  - $ref: '#/parameters/UserRoleHeader'
//  - $ref: '#/parameters/UserNamespaceHeader'
//  - name: namespace
//    in: path
//    type: string
//    required: true
// responses:
//  '200':
//    description: configmaps list
//    schema:
//      $ref: '#/definitions/ConfigMapsResponse'
//  default:
//    $ref: '#/responses/error'
func (h *ConfigMapHandlers) GetConfigMapsListHandler(ctx *gin.Context) {
	resp, err := h.GetConfigMapsList(ctx.Request.Context(), ctx.Param("namespace"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// swagger:operation GET /namespaces/{namespace}/configmaps/{configmap} ConfigMap GetConfigMapHandler
// Get configmaps list.
//
// ---
// x-method-visibility: public
// parameters:
//  - $ref: '#/parameters/UserIDHeader'
//  - $ref: '#/parameters/UserRoleHeader'
//  - $ref: '#/parameters/UserNamespaceHeader'
//  - name: namespace
//    in: path
//    type: string
//    required: true
//  - name: configmap
//    in: path
//    type: string
//    required: true
// responses:
//  '200':
//    description: configmap
//    schema:
//     $ref: '#/definitions/ConfigMapResource'
//  default:
//    $ref: '#/responses/error'
func (h *ConfigMapHandlers) GetConfigMapHandler(ctx *gin.Context) {
	resp, err := h.GetConfigMap(ctx.Request.Context(), ctx.Param("namespace"), ctx.Param("configmap"))

	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// swagger:operation POST /namespaces/{namespace}/configmaps ConfigMap CreateConfigMapHandler
// Create configmap.
//
// ---
// x-method-visibility: public
// parameters:
//  - $ref: '#/parameters/UserIDHeader'
//  - $ref: '#/parameters/UserRoleHeader'
//  - $ref: '#/parameters/UserNamespaceHeader'
//  - name: namespace
//    in: path
//    type: string
//    required: true
//  - name: body
//    in: body
//    schema:
//     $ref: '#/definitions/ConfigMap'
// responses:
//  '201':
//    description: configmap created
//    schema:
//     $ref: '#/definitions/ConfigMapResource'
//  default:
//    $ref: '#/responses/error'
func (h *ConfigMapHandlers) CreateConfigMapHandler(ctx *gin.Context) {
	var req kubtypes.ConfigMap
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	createdCM, err := h.CreateConfigMap(ctx.Request.Context(), ctx.Param("namespace"), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusCreated, createdCM)
}

func (h *ConfigMapHandlers) ImportConfigMapsHandler(ctx *gin.Context) {
	var req kubtypes.ConfigMapsList
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	for _, cm := range req.ConfigMaps {
		if err := h.ImportConfigMap(ctx.Request.Context(), cm.Namespace, cm); err != nil {
			logrus.Warn(err)
		}
	}

	ctx.Status(http.StatusOK)
}

// swagger:operation DELETE /namespaces/{namespace}/configmaps/{configmap} ConfigMap DeleteConfigMapHandler
// Delete configmap.
//
// ---
// x-method-visibility: public
// parameters:
//  - $ref: '#/parameters/UserIDHeader'
//  - $ref: '#/parameters/UserRoleHeader'
//  - $ref: '#/parameters/UserNamespaceHeader'
//  - name: namespace
//    in: path
//    type: string
//    required: true
//  - name: configmap
//    in: path
//    type: string
//    required: true
// responses:
//  '202':
//    description: configmap deleted
//  default:
//    $ref: '#/responses/error'
func (h *ConfigMapHandlers) DeleteConfigMapHandler(ctx *gin.Context) {
	err := h.DeleteConfigMap(ctx.Request.Context(), ctx.Param("namespace"), ctx.Param("configmap"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}

// swagger:operation DELETE /namespaces/{namespace}/configmaps ConfigMap DeleteAllConfigMapsHandler
// Delete configmap.
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
//    description: all configmaps in namespace deleted
//  default:
//    $ref: '#/responses/error'
func (h *ConfigMapHandlers) DeleteAllConfigMapsHandler(ctx *gin.Context) {
	err := h.DeleteAllConfigMaps(ctx.Request.Context(), ctx.Param("namespace"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}
