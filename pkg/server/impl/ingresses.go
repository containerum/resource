package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	kubtypesInternal "git.containerum.net/ch/kube-api/pkg/model"
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/cherrylog"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/pkg/models"
	"git.containerum.net/ch/resource-service/pkg/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

type IngressActionsDB struct {
	NamespaceDB models.NamespaceDBConstructor
	ServiceDB   models.ServiceDBConstructor
	IngressDB   models.IngressDBConstructor
	AccessDB    models.AccessDBConstructor
}

type IngressActionsImpl struct {
	*server.ResourceServiceClients
	*IngressActionsDB

	log *cherrylog.LogrusAdapter
}

func NewIngressActionsImpl(clients *server.ResourceServiceClients, constructors *IngressActionsDB) *IngressActionsImpl {
	return &IngressActionsImpl{
		ResourceServiceClients: clients,
		IngressActionsDB:       constructors,
		log:                    cherrylog.NewLogrusAdapter(logrus.WithField("component", "ingress_actions")),
	}
}

func (ia *IngressActionsImpl) CreateIngress(ctx context.Context, nsLabel string, req rstypes.CreateIngressRequest) error {
	userID := utils.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
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

	err := ia.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		if permErr := server.GetAndCheckPermission(ctx, ia.AccessDB(tx), userID, rstypes.KindNamespace, nsLabel, rstypes.PermissionStatusWrite); permErr != nil {
			return permErr
		}

		service, serviceType, getErr := ia.ServiceDB(tx).GetService(ctx, userID, nsLabel, req.Service)
		if getErr != nil {
			return getErr
		}

		if serviceType != rstypes.ServiceExternal {
			return rserrors.ErrServiceNotExternal()
		}

		paths, pathsErr := server.IngressPaths(service, req.Path, req.ServicePort)
		if pathsErr != nil {
			return pathsErr
		}

		if createErr := ia.IngressDB(tx).CreateIngress(ctx, userID, nsLabel, req); createErr != nil {
			return createErr
		}

		var ingress kubtypesInternal.IngressWithOwner
		ingress.Name = req.Domain
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
					Name: ingress.Name,
					Data: map[string]string{
						"tls.crt": req.TLS.Cert,
						"tls.key": req.TLS.Key,
					},
				},
				Owner: userID,
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

		return nil
	})

	return err
}

func (ia *IngressActionsImpl) GetUserIngresses(ctx context.Context, nsLabel string,
	params rstypes.GetIngressesQueryParams) (rstypes.GetIngressesResponse, error) {
	userID := utils.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Info("get user ingresses")

	resp, err := ia.IngressDB(ia.DB).GetUserIngresses(ctx, userID, nsLabel, params)

	return resp, err
}

func (ia *IngressActionsImpl) GetAllIngresses(ctx context.Context, params rstypes.GetIngressesQueryParams) (rstypes.GetIngressesResponse, error) {
	ia.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
	}).Info("get all ingresses")

	resp, err := ia.IngressDB(ia.DB).GetAllIngresses(ctx, params)

	return resp, err
}

func (ia *IngressActionsImpl) DeleteIngress(ctx context.Context, nsLabel, domain string) error {
	userID := utils.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
		"domain":   domain,
	}).Info("delete ingress")

	err := ia.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		if permErr := server.GetAndCheckPermission(ctx, ia.AccessDB(tx), userID, rstypes.KindNamespace, nsLabel, rstypes.PermissionStatusReadDelete); permErr != nil {
			return permErr
		}

		_, delErr := ia.IngressDB(tx).DeleteIngress(ctx, userID, nsLabel, domain)
		if delErr != nil {
			return delErr
		}

		return nil
	})

	return err
}
