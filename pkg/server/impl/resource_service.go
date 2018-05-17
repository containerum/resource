package impl

import (
	"context"

	"git.containerum.net/ch/resource-service/pkg/db"
	"git.containerum.net/ch/resource-service/pkg/models/resources"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type ResourceCountActionsImpl struct {
	mongo *db.MongoStorage
	log   *cherrylog.LogrusAdapter
}

func NewResourceCountActionsImpl(mongo *db.MongoStorage) *ResourceCountActionsImpl {
	return &ResourceCountActionsImpl{
		mongo: mongo,
		log:   cherrylog.NewLogrusAdapter(logrus.WithField("component", "resource_service")),
	}
}

func (rs *ResourceCountActionsImpl) GetResourcesCount(ctx context.Context) (*resources.GetResourcesCountResponse, error) {
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

	ret := resources.GetResourcesCountResponse{
		Ingresses:   ingresses,
		Deployments: deploys,
		ExtServices: services.External,
		IntServices: services.Internal,
		Pods:        pods,
	}

	return &ret, nil
}
