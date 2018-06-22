package handlers

import (
	"net/http"

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

// swagger:operation GET /domains Domain GetDomainsListHandler
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
		h.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// swagger:operation GET /domains/{domain} Domain GetDomainHandler
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
		h.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// swagger:operation POST /domains Domain AddDomainHandler
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
		h.BadRequest(ctx, err)
		return
	}

	domain, err := h.AddDomain(ctx.Request.Context(), req)
	if err != nil {
		h.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, domain)
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
		h.HandleError(ctx, err)
		return
	}

	ctx.Status(http.StatusAccepted)
}
