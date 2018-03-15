package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	kubtypesInternal "git.containerum.net/ch/kube-api/pkg/model"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/pkg/models"
	"git.containerum.net/ch/resource-service/pkg/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

func (rs *resourceServiceImpl) CreateIngress(ctx context.Context, nsLabel string, req rstypes.CreateIngressRequest) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Infof("create ingress %#v", req)

	// path should be "/" or start with "/"
	if req.Path == "" {
		req.Path = "/"
	}
	if req.Path[0] != '/' {
		req.Path = "/" + req.Path
	}

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		nsID, getErr := tx.GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		service, getErr := tx.GetService(ctx, userID, nsLabel, req.Service)
		if getErr != nil {
			return getErr
		}

		paths, pathsErr := server.IngressPaths(service, req.Path, req.ServicePort)
		if pathsErr != nil {
			return pathsErr
		}

		if createErr := tx.CreateIngress(ctx, userID, nsLabel, req); createErr != nil {
			return createErr
		}

		var ingress kubtypesInternal.IngressWithOwner
		ingress.Name = server.IngressName(req.Domain)
		ingress.Owner = userID
		switch req.Type {
		case rstypes.IngressHTTPS:
			ingress.Rules = append(ingress.Rules, kubtypes.Rule{
				Host:      req.Domain,
				TLSSecret: &req.Service, // if we pass non-existing secret "let`s encrypt" will be used.
				Path:      paths,
			})
		case rstypes.IngressCustomHTTPS:
			// TLS certificate and key stored in "secret" in kube.
			// So before creating ingress we need to create "secret".
			secret := kubtypesInternal.SecretWithOwner{
				Secret: kubtypes.Secret{
					Name: server.SecretName(ingress.Name),
					Data: map[string]string{
						"tls.crt": req.TLS.Cert,
						"tls.key": req.TLS.Key,
					},
				},
				Owner: userID,
			}
			if createErr := rs.Kube.CreateSecret(ctx, nsID, secret); createErr != nil {
				return createErr
			}

			ingress.Rules = append(ingress.Rules, kubtypes.Rule{
				Host:      req.Domain,
				TLSSecret: &secret.Name,
				Path:      paths,
			})
		case rstypes.IngressHTTP:
			ingress.Rules = append(ingress.Rules, kubtypes.Rule{
				Host: req.Domain,
				Path: paths,
			})
		default:
			return rserrors.ErrValidation().AddDetailF("invalid ingress type %s", req.TLS)
		}

		if createErr := rs.Kube.CreateIngress(ctx, nsID, ingress); createErr != nil {
			return createErr
		}

		return nil
	})

	return err
}

func (rs *resourceServiceImpl) GetUserIngresses(ctx context.Context, nsLabel string,
	params rstypes.GetIngressesQueryParams) (rstypes.GetIngressesResponse, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Info("get user ingresses")

	resp, err := rs.DB.GetUserIngresses(ctx, userID, nsLabel, params)

	return resp, err
}

func (rs *resourceServiceImpl) GetAllIngresses(ctx context.Context, params rstypes.GetIngressesQueryParams) (rstypes.GetIngressesResponse, error) {
	rs.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
	}).Info("get all ingresses")

	resp, err := rs.DB.GetAllIngresses(ctx, params)

	return resp, err
}

func (rs *resourceServiceImpl) DeleteIngress(ctx context.Context, nsLabel, domain string) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
		"domain":   domain,
	}).Info("delete ingress")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		nsID, getErr := tx.GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		ingressType, delErr := tx.DeleteIngress(ctx, userID, nsLabel, domain)
		if delErr != nil {
			return delErr
		}

		ingressName := server.IngressName(domain)
		if delErr := rs.Kube.DeleteIngress(ctx, nsID, ingressName); delErr != nil {
			return delErr
		}

		// in CreateIngress() we created secret for "custom_https" ingress so delete it.
		if ingressType == rstypes.IngressCustomHTTPS {
			if delErr := rs.Kube.DeleteSecret(ctx, nsID, server.SecretName(ingressName)); delErr != nil {
				return delErr
			}
		}

		return nil
	})

	return err
}
