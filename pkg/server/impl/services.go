package impl

import (
	"context"

	"git.containerum.net/ch/resource-service/pkg/db"
	"git.containerum.net/ch/resource-service/pkg/models/service"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/containerum/cherry/adaptors/cherrylog"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type ServiceActionsImpl struct {
	mongo *db.MongoStorage
	log   *cherrylog.LogrusAdapter
}

func NewServiceActionsImpl(mongo *db.MongoStorage) *ServiceActionsImpl {
	return &ServiceActionsImpl{
		mongo: mongo,
		log:   cherrylog.NewLogrusAdapter(logrus.WithField("component", "service_actions")),
	}
}

func (sa *ServiceActionsImpl) CreateService(ctx context.Context, nsID string, req kubtypes.Service) (*service.Service, error) {
	userID := httputil.MustGetUserID(ctx)
	sa.log.WithFields(logrus.Fields{
		"user_id": userID,
		"ns_id":   nsID,
	}).Infof("create service %#v", req)

	_, err := sa.mongo.GetDeploymentByName(nsID, req.Deploy)
	if err != nil {
		return nil, err
	}

	serviceType := server.DetermineServiceType(req)

	if serviceType == service.ServiceExternal {
		domain, err := sa.mongo.GetRandomDomain()
		if err != nil {
			return nil, err
		}

		req.Domain = domain.Domain
		req.IPs = domain.IP
		for i := range req.Ports {
			//TODO Select port randomly
			// port, err := domainDB.ChooseDomainFreePort(ctx, domain.Domain, req.Ports[i].Protocol)
			//if portSelectErr != nil {
			//	return portSelectErr
			//}
			port := 1000
			req.Ports[i].Port = &port
		}
	}

	//TODO
	/*
		nsUsage, getErr := sa.NamespaceDB(tx).GetNamespaceUsage(ctx, ns.Namespace)
		if getErr != nil {
			return getErr
		}

		if chkErr := server.CheckServiceCreateQuotas(ns.Namespace, nsUsage, serviceType); chkErr != nil {
			return chkErr
		}

		if createErr := sa.Kube.CreateService(ctx, ns.ID, req); createErr != nil {
			return createErr
		}
	*/

	createdService, err := sa.mongo.CreateService(service.ServiceFromKube(nsID, userID, req))
	if err != nil {
		return nil, err
	}

	return &createdService, nil
}

func (sa *ServiceActionsImpl) GetServices(ctx context.Context, nsID string) ([]service.Service, error) {
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

func (sa *ServiceActionsImpl) UpdateService(ctx context.Context, nsID string, req kubtypes.Service) (*service.Service, error) {
	userID := httputil.MustGetUserID(ctx)
	sa.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"namespace":    nsID,
		"service_name": req.Name,
	}).Info("update service")

	serviceType := server.DetermineServiceType(kubtypes.Service(req))

	if serviceType == service.ServiceExternal {
		domain, err := sa.mongo.GetRandomDomain()
		if err != nil {
			return nil, err
		}

		req.Domain = domain.Domain
		req.IPs = domain.IP
		for i := range req.Ports {
			//TODO Select port randomly
			// port, err := domainDB.ChooseDomainFreePort(ctx, domain.Domain, req.Ports[i].Protocol)
			//if portSelectErr != nil {
			//	return portSelectErr
			//}
			port := 1000
			req.Ports[i].Port = &port
		}
	}

	//TODO
	/*
		if updErr := sa.Kube.UpdateService(ctx, nsID, kubtypes.Service(req)); updErr != nil {
			return updErr
		}
	*/

	createdService, err := sa.mongo.CreateService(service.ServiceFromKube(nsID, userID, req))
	if err != nil {
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

	/*
		TODO Check ingresses
			_, getErr = sa.IngressDB(tx).GetIngress(ctx, userID, nsLabel, serviceName)
			switch {
			case getErr == nil:
				return rserrors.ErrServiceHasIngresses()
			case cherry.Equals(getErr, rserrors.ErrResourceNotExists()):
				// pass
			default:
				return getErr
			}

			if delErr := sa.Kube.DeleteService(ctx, nsID, serviceName); delErr != nil {
				return delErr
			}
	*/

	err := sa.mongo.DeleteService(nsID, serviceName)
	if err != nil {
		return err
	}

	return nil
}
