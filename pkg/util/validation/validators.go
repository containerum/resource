package validation

import (
	rstypes "git.containerum.net/ch/json-types/resource-service"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/en_US"
	"github.com/go-playground/universal-translator"
	"gopkg.in/go-playground/validator.v9"
	enTranslations "gopkg.in/go-playground/validator.v9/translations/en"
)

func StandardResourceValidator(uni *ut.UniversalTranslator) (ret *validator.Validate) {
	ret = validator.New()
	ret.SetTagName("binding")

	enTranslator, _ := uni.GetTranslator(en.New().Locale())
	enUSTranslator, _ := uni.GetTranslator(en_US.New().Locale())

	enTranslations.RegisterDefaultTranslations(ret, enTranslator)
	enTranslations.RegisterDefaultTranslations(ret, enUSTranslator)

	ret.RegisterStructValidation(createIngressRequestValidate, rstypes.CreateIngressRequest{})
	rstypes.RegisterCustomTags(ret)

	return
}

func createIngressRequestValidate(structLevel validator.StructLevel) {
	req := structLevel.Current().Interface().(rstypes.CreateIngressRequest)

	if req.Type == rstypes.IngressCustomHTTPS {
		if req.TLS == nil {
			structLevel.ReportError(req.TLS, "TLS", "tls", "exists", "")
		}
	}
}
