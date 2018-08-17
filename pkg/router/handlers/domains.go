package handlers

import (
	"net/http"

	"errors"

	"git.containerum.net/ch/resource-service/pkg/models/domain"
	m "git.containerum.net/ch/resource-service/pkg/router/middleware"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type DomainHandlers struct {
	server.DomainActions
	*m.TranslateValidate
}

// swagger:operation GET /domains Domain GetDomainsList
// Get domains list.
//
// ---
// x-method-visibility: public
// parameters:
//  - $ref: '#/parameters/UserRoleHeader'
//  - name: page
//    in: query
//    type: string
//    required: false
//  - name: per_page
//    in: query
//    type: string
//    required: false
// responses:
//  '200':
//    description: domains list
//    schema:
//      $ref: '#/definitions/DomainsResponse'
//  default:
//    $ref: '#/responses/error'
func (h *DomainHandlers) GetDomainsListHandler(ctx *gin.Context) {
	resp, err := h.GetDomainsList(ctx.Request.Context(), ctx.Query("page"), ctx.Query("per_page"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// swagger:operation GET /domains/{domain} Domain GetDomain
// Get domain.
//
// ---
// x-method-visibility: public
// parameters:
//  - $ref: '#/parameters/UserRoleHeader'
//  - name: domain
//    in: path
//    type: string
//    required: true
// responses:
//  '200':
//    description: domains
//    schema:
//      $ref: '#/definitions/Domain'
//  default:
//    $ref: '#/responses/error'
func (h *DomainHandlers) GetDomainHandler(ctx *gin.Context) {
	resp, err := h.GetDomain(ctx.Request.Context(), ctx.Param("domain"))
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// swagger:operation POST /domains Domain AddDomain
// Add domain.
//
// ---
// x-method-visibility: public
// parameters:
//  - $ref: '#/parameters/UserRoleHeader'
//  - name: body
//    in: body
//    schema:
//      $ref: '#/definitions/Domain'
// responses:
//  '201':
//    description: domains
//    schema:
//      $ref: '#/definitions/Domain'
//  default:
//    $ref: '#/responses/error'
func (h *DomainHandlers) AddDomainHandler(ctx *gin.Context) {
	var req domain.Domain
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, err))
		return
	}

	if len(req.IP) == 0 {
		ctx.AbortWithStatusJSON(h.BadRequest(ctx, errors.New("at least 1 IP is required")))
		return
	}

	if req.Domain == "" {
		req.Domain = req.IP[0]
	}

	ret, err := h.AddDomain(ctx.Request.Context(), req)
	if err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.JSON(http.StatusCreated, ret)
}

// swagger:operation DELETE /domains/{domain} Domain DeleteDomainHandler
// Add domain.
//
// ---
// x-method-visibility: public
// parameters:
//  - $ref: '#/parameters/UserRoleHeader'
//  - name: domain
//    in: path
//    type: string
//    required: true
// responses:
//  '202':
//    description: domain deleted
//  default:
//    $ref: '#/responses/error'
func (h *DomainHandlers) DeleteDomainHandler(ctx *gin.Context) {
	if err := h.DeleteDomain(ctx.Request.Context(), ctx.Param("domain")); err != nil {
		ctx.AbortWithStatusJSON(h.HandleError(err))
		return
	}

	ctx.Status(http.StatusAccepted)
}
