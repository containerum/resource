package impl

import (
	"context"

	"strings"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	kubtypesInternal "git.containerum.net/ch/kube-api/pkg/model"
	"git.containerum.net/ch/resource-service/pkg/models"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/containerum/kube-client/pkg/cherry/resource-service"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
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

func (ia *IngressActionsImpl) CreateIngress(ctx context.Context, nsLabel string, req kubtypes.Ingress) error {
	userID := httputil.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Infof("create ingress %#v", req)

	err := ia.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		nsID, getErr := ia.NamespaceDB(tx).GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if permErr := server.GetAndCheckPermission(ctx, ia.AccessDB(tx), userID, rstypes.KindNamespace, nsLabel, rstypes.PermissionStatusWrite); permErr != nil {
			return permErr
		}

		if req.Rules[0].Path[0].Path == "" {
			req.Rules[0].Path[0].Path = "/"
		}

		service, serviceType, getErr := ia.ServiceDB(tx).GetService(ctx, userID, nsLabel, req.Rules[0].Path[0].ServiceName)
		if getErr != nil {
			return getErr
		}

		if serviceType != rstypes.ServiceExternal {
			return rserrors.ErrServiceNotExternal()
		}

		_, getErr = ia.IngressDB(tx).GetIngress(ctx, userID, nsLabel, service.Name)
		switch {
		case getErr == nil:
			return rserrors.ErrResourceAlreadyExists().AddDetailF("ingress for service %s already exists", service.Name)
		case cherry.Equals(getErr, rserrors.ErrResourceNotExists()):
			// pass
		default:
			return getErr
		}

		var ingressType rstypes.IngressType
		switch {
		case req.Rules[0].TLSSecret == nil:
			ingressType = rstypes.IngressHTTP
		case strings.HasPrefix(*req.Rules[0].TLSSecret, "letsencrypt"):
			ingressType = rstypes.IngressHTTPS
		default:
			ingressType = rstypes.IngressCustomHTTPS
		}

		if createErr := ia.IngressDB(tx).CreateIngress(ctx, userID, nsLabel, rstypes.CreateIngressRequest{
			Ingress: rstypes.Ingress{
				Domain:      req.Rules[0].Host,
				Type:        ingressType,
				Service:     req.Rules[0].Path[0].ServiceName,
				Path:        req.Rules[0].Path[0].Path,
				ServicePort: req.Rules[0].Path[0].ServicePort,
			},
		}); createErr != nil {
			return createErr
		}

		if createErr := ia.Kube.CreateIngress(ctx, nsID, kubtypesInternal.IngressWithOwner{Ingress: req, Owner: userID}); createErr != nil {
			return createErr
		}

		return nil
	})

	return err
}

func (ia *IngressActionsImpl) GetUserIngresses(ctx context.Context, nsLabel string,
	params rstypes.GetIngressesQueryParams) (rstypes.GetIngressesResponse, error) {
	userID := httputil.MustGetUserID(ctx)
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
	userID := httputil.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
		"domain":   domain,
	}).Info("delete ingress")

	err := ia.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		nsID, getErr := ia.NamespaceDB(tx).GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if permErr := server.GetAndCheckPermission(ctx, ia.AccessDB(tx), userID, rstypes.KindNamespace, nsLabel, rstypes.PermissionStatusReadDelete); permErr != nil {
			return permErr
		}

		_, delErr := ia.IngressDB(tx).DeleteIngress(ctx, userID, nsLabel, domain)
		if delErr != nil {
			return delErr
		}

		ingressName := domain
		if delErr := ia.Kube.DeleteIngress(ctx, nsID, ingressName); delErr != nil {
			return delErr
		}

		return nil
	})

	return err
}
