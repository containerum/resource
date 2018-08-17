package impl

import (
	"context"

	"git.containerum.net/ch/resource-service/pkg/db"
	"git.containerum.net/ch/resource-service/pkg/models/resources"
	"git.containerum.net/ch/resource-service/pkg/rserrors"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type ResourcesActionsImpl struct {
	mongo *db.MongoStorage
	log   *cherrylog.LogrusAdapter
}

func NewResourcesActionsImpl(mongo *db.MongoStorage) *ResourcesActionsImpl {
	return &ResourcesActionsImpl{
		mongo: mongo,
		log:   cherrylog.NewLogrusAdapter(logrus.WithField("component", "resource_service")),
	}
}

func (rs *ResourcesActionsImpl) GetResourcesCount(ctx context.Context) (*resources.GetResourcesCountResponse, error) {
	userID := httputil.MustGetUserID(ctx)
	rs.log.WithField("user_id", userID).Info("get resources count")

	ingresses, err := rs.mongo.CountIngresses(userID)
	if err != nil {
		rs.log.Debug(err)
		return nil, rserrors.ErrUnableCountResources()
	}
	deploys, err := rs.mongo.CountDeployments(userID)
	if err != nil {
		rs.log.Debug(err)
		return nil, rserrors.ErrUnableCountResources()
	}
	services, err := rs.mongo.CountServices(userID)
	if err != nil {
		rs.log.Debug(err)
		return nil, rserrors.ErrUnableCountResources()
	}
	pods, err := rs.mongo.CountReplicas(userID)
	if err != nil {
		rs.log.Debug(err)
		return nil, rserrors.ErrUnableCountResources()
	}
	cms, err := rs.mongo.CountConfigMaps(userID)
	if err != nil {
		rs.log.Debug(err)
		return nil, rserrors.ErrUnableCountResources()
	}

	ret := resources.GetResourcesCountResponse{
		Ingresses:   ingresses,
		Deployments: deploys,
		ExtServices: services.External,
		IntServices: services.Internal,
		Pods:        pods,
		ConfigMaps:  cms,
	}

	return &ret, nil
}

func (rs *ResourcesActionsImpl) GetAllResourcesCount(ctx context.Context) (*resources.GetResourcesCountResponse, error) {
	ingresses, err := rs.mongo.CountAllIngresses()
	if err != nil {
		rs.log.Debug(err)
		return nil, rserrors.ErrUnableCountResources()
	}
	deploys, err := rs.mongo.CountAllDeployments()
	if err != nil {
		rs.log.Debug(err)
		return nil, rserrors.ErrUnableCountResources()
	}
	services, err := rs.mongo.CountAllServices()
	if err != nil {
		rs.log.Debug(err)
		return nil, rserrors.ErrUnableCountResources()
	}
	pods, err := rs.mongo.CountAllReplicas()
	if err != nil {
		rs.log.Debug(err)
		return nil, rserrors.ErrUnableCountResources()
	}
	cms, err := rs.mongo.CountAllConfigMaps()
	if err != nil {
		rs.log.Debug(err)
		return nil, rserrors.ErrUnableCountResources()
	}

	ret := resources.GetResourcesCountResponse{
		Ingresses:   ingresses,
		Deployments: deploys,
		ExtServices: services.External,
		IntServices: services.Internal,
		Pods:        pods,
		ConfigMaps:  cms,
	}

	return &ret, nil
}

func (rs *ResourcesActionsImpl) DeleteAllResourcesInNamespace(ctx context.Context, nsID string) error {
	rs.log.WithField("namespace_id", nsID).Info("deleting all resources")
	if err := rs.mongo.DeleteAllIngressesInNamespace(nsID); err != nil {
		return err
	}
	if err := rs.mongo.DeleteAllServicesInNamespace(nsID); err != nil {
		return err
	}
	if err := rs.mongo.DeleteAllDeploymentsInNamespace(nsID); err != nil {
		return err
	}
	if err := rs.mongo.DeleteAllConfigMapsInNamespace(nsID); err != nil {
		return err
	}
	return nil
}

func (rs *ResourcesActionsImpl) DeleteAllUserResources(ctx context.Context) error {
	userID := httputil.MustGetUserID(ctx)
	rs.log.WithField("user_id", userID).Info("deleting all user resources")
	if err := rs.mongo.DeleteAllIngressesByOwner(userID); err != nil {
		return err
	}
	if err := rs.mongo.DeleteAllServicesByOwner(userID); err != nil {
		return err
	}
	if err := rs.mongo.DeleteAllDeploymentsByOwner(userID); err != nil {
		return err
	}
	if err := rs.mongo.DeleteAllConfigMapsByOwner(userID); err != nil {
		return err
	}
	return nil
}
