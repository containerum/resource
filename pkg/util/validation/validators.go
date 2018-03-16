package validation

import (
	"fmt"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/pkg/server"
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

	registerCustomTags(ret)
	registerCustomTagsENTranslation(ret, enTranslator)
	registerCustomTagsENTranslation(ret, enUSTranslator)

	ret.RegisterStructValidation(createIngressRequestValidate, rstypes.CreateIngressRequest{})
	ret.RegisterStructValidation(serviceValidate, kubtypes.Service{})
	ret.RegisterStructValidation(updateServiceValidate, server.UpdateServiceRequest{})
	ret.RegisterStructValidation(deploymentValidate, kubtypes.Deployment{})
	ret.RegisterStructValidation(containerVolumeValidate, kubtypes.ContainerVolume{})
	ret.RegisterStructValidation(containerPortValidate, kubtypes.ContainerPort{})

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

func serviceValidate(structLevel validator.StructLevel) {
	req := structLevel.Current().Interface().(kubtypes.Service)

	v := structLevel.Validator()

	if err := v.Var(req.Name, "dns"); err != nil {
		structLevel.ReportValidationErrors("Name", "", err.(validator.ValidationErrors))
	}

	if err := v.Var(req.Deploy, "dns"); err != nil {
		structLevel.ReportValidationErrors("Deploy", "", err.(validator.ValidationErrors))
	}

	for i, port := range req.Ports {
		if err := v.Var(port.Protocol, "eq=TCP|eq=UDP"); err != nil {
			structLevel.ReportValidationErrors(fmt.Sprintf("Ports[%d].Protocol", i), "", err.(validator.ValidationErrors))
		}

		if err := v.Var(port.Port, "omitempty,min=1,max=65535"); err != nil {
			structLevel.ReportValidationErrors(fmt.Sprintf("Ports[%d].Port", i), "", err.(validator.ValidationErrors))
		}

		if err := v.Var(port.TargetPort, "min=1,max=65535"); err != nil {
			structLevel.ReportValidationErrors(fmt.Sprintf("Ports[%d].TargetPort", i), "", err.(validator.ValidationErrors))
		}
	}

	for i, ip := range req.IPs {
		if err := v.Var(ip, "ip"); err != nil {
			structLevel.ReportValidationErrors(fmt.Sprintf("IPs[%d]", i), "", err.(validator.ValidationErrors))
		}
	}
}

func updateServiceValidate(structLevel validator.StructLevel) {
	req := structLevel.Current().Interface().(kubtypes.Service)

	v := structLevel.Validator()

	for i, port := range req.Ports {
		if err := v.Var(port.Protocol, "eq=TCP|eq=UDP"); err != nil {
			structLevel.ReportValidationErrors(fmt.Sprintf("Ports[%d].Protocol", i), "", err.(validator.ValidationErrors))
		}

		if err := v.Var(port.Port, "min=1,max=65535"); err != nil {
			structLevel.ReportValidationErrors(fmt.Sprintf("Ports[%d].Port", i), "", err.(validator.ValidationErrors))
		}

		if err := v.Var(port.TargetPort, "omitempty,min=1,max=65535"); err != nil {
			structLevel.ReportValidationErrors(fmt.Sprintf("Ports[%d].TargetPort", i), "", err.(validator.ValidationErrors))
		}

	}

	for i, ip := range req.IPs {
		if err := v.Var(ip, "ip"); err != nil {
			structLevel.ReportValidationErrors(fmt.Sprintf("IPs[%d]", i), "", err.(validator.ValidationErrors))
		}
	}
}

func deploymentValidate(structLevel validator.StructLevel) {
	req := structLevel.Current().Interface().(kubtypes.Deployment)

	v := structLevel.Validator()

	if err := v.Var(req.Name, "dns"); err != nil {
		structLevel.ReportValidationErrors("Name", "", err.(validator.ValidationErrors))
	}

	if err := v.Var(req.Replicas, "min=1,max=15"); err != nil {
		structLevel.ReportValidationErrors("Replicas", "", err.(validator.ValidationErrors))
	}

	if err := v.Var(req.Containers, "min=1"); err != nil { // at least 1 container
		structLevel.ReportValidationErrors("Containers", "", err.(validator.ValidationErrors))
	}

	for i, container := range req.Containers {
		if err := v.Var(container.Name, "dns"); err != nil {
			structLevel.ReportValidationErrors(fmt.Sprintf("Containers[%d].Name", i), "", err.(validator.ValidationErrors))
		}

		if err := v.Var(container.Image, "docker_image"); err != nil {
			structLevel.ReportValidationErrors(fmt.Sprintf("Containers[%d].Image", i), "", err.(validator.ValidationErrors))
		}

		for j, cm := range container.ConfigMaps {
			if err := v.Struct(cm); err != nil {
				structLevel.ReportValidationErrors(fmt.Sprintf("Containers[%d].ConfigMaps[%d]", i, j), "", err.(validator.ValidationErrors))
			}
		}

		for j, vm := range container.VolumeMounts {
			if err := v.Struct(vm); err != nil {
				structLevel.ReportValidationErrors(fmt.Sprintf("Containers[%d].VolumeMounts[%d]", i, j), "", err.(validator.ValidationErrors))
			}
		}

		for j, cp := range container.Ports {
			if err := v.Struct(cp); err != nil {
				structLevel.ReportValidationErrors(fmt.Sprintf("Containers[%d].Ports[%d]", i, j), "", err.(validator.ValidationErrors))
			}
		}

		if err := v.Var(container.Limits.Memory, "kube_quantity"); err != nil {
			structLevel.ReportValidationErrors(fmt.Sprintf("Containers[%d].Limits.Memory", i), "", err.(validator.ValidationErrors))
		}

		if err := v.Var(container.Limits.CPU, "kube_quantity"); err != nil {
			structLevel.ReportValidationErrors(fmt.Sprintf("Containers[%d].Limits.CPU", i), "", err.(validator.ValidationErrors))
		}
	}
}

func containerVolumeValidate(structLevel validator.StructLevel) {
	req := structLevel.Current().Interface().(kubtypes.ContainerVolume)

	v := structLevel.Validator()

	if err := v.Var(req.Name, "dns"); err != nil {
		structLevel.ReportValidationErrors("Name", "", err.(validator.ValidationErrors))
	}
}

func containerPortValidate(structLevel validator.StructLevel) {
	req := structLevel.Current().Interface().(kubtypes.ContainerPort)

	v := structLevel.Validator()

	if err := v.Var(req.Name, "dns"); err != nil {
		structLevel.ReportValidationErrors("Name", "", err.(validator.ValidationErrors))
	}

	if err := v.Var(req.Port, "min=1,max=65535"); err != nil {
		structLevel.ReportValidationErrors("Port", "", err.(validator.ValidationErrors))
	}

	if err := v.Var(req.Protocol, "eq=TCP|eq=UDP"); err != nil {
		structLevel.ReportValidationErrors("Protocol", "", err.(validator.ValidationErrors))
	}
}
