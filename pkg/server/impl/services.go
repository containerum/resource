package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/pkg/models"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type ServiceActionsDB struct {
	ServiceDB   models.ServiceDBConstructor
	NamespaceDB models.NamespaceDBConstructor
	DomainDB    models.DomainDBConstructor
	AccessDB    models.AccessDBConstructor
	IngressDB   models.IngressDBConstructor
}

type ServiceActionsImpl struct {
	*server.ResourceServiceClients
	*ServiceActionsDB

	log *cherrylog.LogrusAdapter
}

func NewServiceActionsImpl(clients *server.ResourceServiceClients, constructors *ServiceActionsDB) *ServiceActionsImpl {
	return &ServiceActionsImpl{
		ResourceServiceClients: clients,
		ServiceActionsDB:       constructors,
		log:                    cherrylog.NewLogrusAdapter(logrus.WithField("component", "service_actions")),
	}
}

func (sa *ServiceActionsImpl) CreateService(ctx context.Context, nsLabel string, req kubtypes.Service) error {
	userID := httputil.MustGetUserID(ctx)
	sa.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Infof("create service %#v", req)

	err := sa.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		if permErr := server.GetAndCheckPermission(ctx, sa.AccessDB(tx), userID, rstypes.KindNamespace, nsLabel, rstypes.PermissionStatusWrite); permErr != nil {
			return permErr
		}

		serviceType := server.DetermineServiceType(req)

		if serviceType == rstypes.ServiceExternal {
			domainDB := sa.DomainDB(tx)
			domain, selectErr := domainDB.ChooseRandomDomain(ctx)
			if selectErr != nil {
				return selectErr
			}

			req.Domain = domain.Domain
			req.IPs = domain.IP
			// TODO: SQL queries in loop is not good solution
			for i := range req.Ports {
				port, portSelectErr := domainDB.ChooseDomainFreePort(ctx, domain.Domain, req.Ports[i].Protocol)
				if portSelectErr != nil {
					return portSelectErr
				}
				req.Ports[i].Port = &port
			}
		}

		ns, getErr := sa.NamespaceDB(tx).GetUserNamespaceByLabel(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		nsUsage, getErr := sa.NamespaceDB(tx).GetNamespaceUsage(ctx, ns.Namespace)
		if getErr != nil {
			return getErr
		}

		if chkErr := server.CheckServiceCreateQuotas(ns.Namespace, nsUsage, serviceType); chkErr != nil {
			return chkErr
		}

		if createErr := sa.ServiceDB(tx).CreateService(ctx, userID, nsLabel, serviceType, req); createErr != nil {
			return createErr
		}

		if createErr := sa.Kube.CreateService(ctx, ns.ID, req); createErr != nil {
			return createErr
		}

		return nil
	})

	return err
}

func (sa *ServiceActionsImpl) GetServices(ctx context.Context, nsLabel string) ([]kubtypes.Service, error) {
	userID := httputil.MustGetUserID(ctx)
	sa.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Info("get services")

	ret, err := sa.ServiceDB(sa.DB).GetServices(ctx, userID, nsLabel)

	return ret, err
}

func (sa *ServiceActionsImpl) GetService(ctx context.Context, nsLabel, serviceName string) (kubtypes.Service, error) {
	userID := httputil.MustGetUserID(ctx)
	sa.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"service_name": serviceName,
	}).Info("get service")

	ret, _, err := sa.ServiceDB(sa.DB).GetService(ctx, userID, nsLabel, serviceName)

	return ret, err
}

func (sa *ServiceActionsImpl) UpdateService(ctx context.Context, nsLabel string, req server.UpdateServiceRequest) error {
	userID := httputil.MustGetUserID(ctx)
	sa.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"service_name": req.Name,
	}).Info("update service")

	err := sa.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		if permErr := server.GetAndCheckPermission(ctx, sa.AccessDB(tx), userID, rstypes.KindNamespace, nsLabel, rstypes.PermissionStatusWrite); permErr != nil {
			return permErr
		}

		serviceType := server.DetermineServiceType(kubtypes.Service(req))

		if serviceType == rstypes.ServiceExternal {
			domainDB := sa.DomainDB(tx)
			domain, selectErr := domainDB.ChooseRandomDomain(ctx)
			if selectErr != nil {
				return selectErr
			}

			req.Domain = domain.Domain
			req.IPs = domain.IP
			for i := range req.Ports {
				port, portSelectErr := domainDB.ChooseDomainFreePort(ctx, domain.Domain, req.Ports[i].Protocol)
				if portSelectErr != nil {
					return portSelectErr
				}
				req.Ports[i].Port = &port
			}
		}

		nsID, getErr := sa.NamespaceDB(tx).GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if updErr := sa.ServiceDB(tx).UpdateService(ctx, userID, nsLabel, serviceType, kubtypes.Service(req)); updErr != nil {
			return updErr
		}

		if updErr := sa.Kube.UpdateService(ctx, nsID, kubtypes.Service(req)); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (sa *ServiceActionsImpl) DeleteService(ctx context.Context, nsLabel, serviceName string) error {
	userID := httputil.MustGetUserID(ctx)
	sa.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"service_name": serviceName,
	}).Info("delete service")

	err := sa.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		nsID, getErr := sa.NamespaceDB(tx).GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if permErr := server.GetAndCheckPermission(ctx, sa.AccessDB(tx), userID, rstypes.KindNamespace, nsLabel, rstypes.PermissionStatusWrite); permErr != nil {
			return permErr
		}

		_, getErr = sa.IngressDB(tx).GetIngress(ctx, userID, nsLabel, serviceName)
		switch {
		case getErr == nil:
			return rserrors.ErrServiceHasIngresses()
		case cherry.Equals(getErr, rserrors.ErrResourceNotExists()):
			// pass
		default:
			return getErr
		}

		if delErr := sa.ServiceDB(tx).DeleteService(ctx, userID, nsLabel, serviceName); delErr != nil {
			return delErr
		}

		if delErr := sa.Kube.DeleteService(ctx, nsID, serviceName); delErr != nil {
			return delErr
		}

		return nil
	})

	return err
}
