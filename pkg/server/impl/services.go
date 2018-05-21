package impl

import (
	"context"

	"git.containerum.net/ch/resource-service/pkg/clients"
	"git.containerum.net/ch/resource-service/pkg/db"
	"git.containerum.net/ch/resource-service/pkg/models/service"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type ServiceActionsImpl struct {
	kube        clients.Kube
	permissions clients.Permissions
	mongo       *db.MongoStorage
	log         *cherrylog.LogrusAdapter
}

func NewServiceActionsImpl(mongo *db.MongoStorage, permissions *clients.Permissions, kube *clients.Kube) *ServiceActionsImpl {
	return &ServiceActionsImpl{
		kube:        *kube,
		mongo:       mongo,
		permissions: *permissions,
		log:         cherrylog.NewLogrusAdapter(logrus.WithField("component", "service_actions")),
	}
}

func (sa *ServiceActionsImpl) GetServices(ctx context.Context, nsID string) (service.ServiceList, error) {
	userID := httputil.MustGetUserID(ctx)
	sa.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"namespace": nsID,
	}).Info("get services")

	return sa.mongo.GetServiceList(nsID)
}

func (sa *ServiceActionsImpl) GetService(ctx context.Context, nsID, serviceName string) (*service.Service, error) {
	userID := httputil.MustGetUserID(ctx)
	sa.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"namespace":    nsID,
		"service_name": serviceName,
	}).Info("get service")

	ret, err := sa.mongo.GetService(nsID, serviceName)

	return &ret, err
}

func (sa *ServiceActionsImpl) CreateService(ctx context.Context, nsID string, req kubtypes.Service) (*service.Service, error) {
	userID := httputil.MustGetUserID(ctx)
	sa.log.WithFields(logrus.Fields{
		"user_id": userID,
		"ns_id":   nsID,
	}).Infof("create service %#v", req)

	_, err := sa.mongo.GetDeployment(nsID, req.Deploy)
	if err != nil {
		sa.log.Error(err)
		return nil, rserrors.ErrResourceNotExists().AddDetailF("deployment '%s' not exists", req.Deploy)
	}

	serviceType := server.DetermineServiceType(req)

	if serviceType == service.ServiceExternal {
		domain, err := sa.mongo.GetRandomDomain()
		if err != nil {
			return nil, err
		}

		req.Domain = domain.Domain
		req.IPs = domain.IP
		for i, port := range req.Ports {
			externalPort, err := sa.mongo.GetFreePort(domain.Domain, port.Protocol)
			if err != nil {
				return nil, err
			}
			req.Ports[i].Port = &externalPort
		}
	}

	nsLimits, err := sa.permissions.GetNamespaceLimits(ctx, nsID)
	if err != nil {
		return nil, err
	}

	nsUsage, err := sa.mongo.CountServicesInNamespace(nsID)
	if err != nil {
		return nil, err
	}

	if err := server.CheckServiceCreateQuotas(nsLimits, nsUsage, serviceType); err != nil {
		return nil, err
	}

	createdService, err := sa.mongo.CreateService(service.ServiceFromKube(nsID, userID, req))
	if err != nil {
		return nil, err
	}

	if err := sa.kube.CreateService(ctx, nsID, req); err != nil {
		sa.log.Debug("Kube-API error! Deleting service from DB.")
		if err := sa.mongo.DeleteService(nsID, req.Name); err != nil {
			return nil, err
		}
		return nil, err
	}

	return &createdService, nil
}

func (sa *ServiceActionsImpl) UpdateService(ctx context.Context, nsID string, req kubtypes.Service) (*service.Service, error) {
	userID := httputil.MustGetUserID(ctx)
	sa.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"namespace":    nsID,
		"service_name": req.Name,
	}).Info("update service")

	oldService, err := sa.mongo.GetService(nsID, req.Name)
	if err != nil {
		return nil, err
	}

	serviceType := server.DetermineServiceType(kubtypes.Service(req))

	if serviceType == service.ServiceExternal {
		domain, err := sa.mongo.GetRandomDomain()
		if err != nil {
			return nil, err
		}

		req.Domain = domain.Domain
		req.IPs = domain.IP
		for i, port := range req.Ports {
			var externalPort int
			if oldService.Ports[i].Port != nil {
				externalPort = *oldService.Ports[i].Port
			} else {
				externalPort, err = sa.mongo.GetFreePort(domain.Domain, port.Protocol)
				if err != nil {
					return nil, err
				}
			}
			req.Ports[i].Port = &externalPort
		}
	}

	createdService, err := sa.mongo.UpdateService(service.ServiceFromKube(nsID, userID, req))
	if err != nil {
		return nil, err
	}

	if err := sa.kube.UpdateService(ctx, nsID, req); err != nil {
		sa.log.Debug("Kube-API error! Reverting changes.")
		if _, err := sa.mongo.UpdateService(oldService); err != nil {
			return nil, err
		}
		return nil, err
	}

	return &createdService, nil
}

func (sa *ServiceActionsImpl) DeleteService(ctx context.Context, nsID, serviceName string) error {
	userID := httputil.MustGetUserID(ctx)
	sa.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_id":        nsID,
		"service_name": serviceName,
	}).Info("delete service")

	_, err := sa.mongo.GetIngress(nsID, serviceName)
	switch {
	case err == nil:
		return rserrors.ErrServiceHasIngresses()
	case cherry.Equals(err, rserrors.ErrResourceNotExists()):
		// pass
	default:
		return err
	}

	if err := sa.mongo.DeleteService(nsID, serviceName); err != nil {
		return err
	}

	if err := sa.kube.DeleteService(ctx, nsID, serviceName); err != nil {
		sa.log.Debug("Kube-API error! Reverting changes.")
		if err := sa.mongo.RestoreService(nsID, serviceName); err != nil {
			return err
		}
		return err
	}

	return nil
}

func (sa *ServiceActionsImpl) DeleteAllServices(ctx context.Context, nsID string) error {
	sa.log.WithFields(logrus.Fields{
		"ns_id": nsID,
	}).Info("delete all services")

	if err := sa.mongo.DeleteAllServicesInNamespace(nsID); err != nil {
		return err
	}

	return nil
}
