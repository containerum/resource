package middleware

import (
	"net/textproto"

	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/gonic"
	"github.com/containerum/utils/httputil"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/universal-translator"
	"gopkg.in/go-playground/validator.v9"
)

type TranslateValidate struct {
	*ut.UniversalTranslator
	*validator.Validate
}

func (tv *TranslateValidate) HandleError(ctx *gin.Context, err error) {
	ctx.Error(err)
	switch err.(type) {
	case *cherry.Err:
		e := err.(*cherry.Err)
		gonic.Gonic(e, ctx)
	default:
		gonic.Gonic(rserrors.ErrInternal().AddDetailsErr(err), ctx)
	}
}

func (tv *TranslateValidate) BadRequest(ctx *gin.Context, err error) {
	ctx.Error(err)
	if validationErr, ok := err.(validator.ValidationErrors); ok {
		ret := rserrors.ErrValidation()
		for _, fieldErr := range validationErr {
			if fieldErr == nil {
				continue
			}
			t, _ := tv.FindTranslator(httputil.GetAcceptedLanguages(ctx.Request.Context())...)
			ret.AddDetailF("Field %s: %s", fieldErr.Namespace(), fieldErr.Translate(t))
		}
		gonic.Gonic(ret, ctx)
		return
	}
	gonic.Gonic(rserrors.ErrValidation().AddDetailsErr(err), ctx)
}

func (tv *TranslateValidate) ValidateHeaders(headerTagMap map[string]string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		headerErr := make(map[string]validator.ValidationErrors)
		for header, tag := range headerTagMap {
			ferr := tv.VarCtx(ctx.Request.Context(), ctx.GetHeader(textproto.CanonicalMIMEHeaderKey(header)), tag)
			if ferr != nil {
				headerErr[header] = ferr.(validator.ValidationErrors)
			}
		}
		if len(headerErr) > 0 {
			ret := rserrors.ErrValidation()
			for header, fieldErrs := range headerErr {
				for _, fieldErr := range fieldErrs {
					if fieldErr == nil {
						continue
					}
					t, _ := tv.FindTranslator(httputil.GetAcceptedLanguages(ctx.Request.Context())...)
					ret.AddDetailF("Header %s: %s", header, fieldErr.Translate(t))
				}
			}
			gonic.Gonic(ret, ctx)
			return
		}
	}
}
