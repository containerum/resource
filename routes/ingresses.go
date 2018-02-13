package routes

import (
	"github.com/gin-gonic/gin"

	"reflect"

	"net/http"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"gopkg.in/go-playground/validator.v8"
)

func createIngressRequestValidate(v *validator.Validate, structLevel *validator.StructLevel) {
	req := structLevel.CurrentStruct.Interface().(rstypes.CreateIngressRequest)

	if req.Type == rstypes.IngressCustomHTTPS {
		if req.TLS == nil {
			structLevel.ReportError(reflect.ValueOf(req.TLS), "TLS", "tls", "exists")
		}
	}
}

func createIngressHandler(ctx *gin.Context) {
	var req rstypes.CreateIngressRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}

	if err := customValidator.Struct(req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}

	if err := srv.CreateIngress(ctx.Request.Context(), ctx.Param("ns_label"), req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func deleteIngressHandler(ctx *gin.Context) {
	if err := srv.DeleteIngress(ctx.Request.Context(), ctx.Param("ns_label"), ctx.Param("domain")); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}
