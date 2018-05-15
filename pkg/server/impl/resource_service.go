package impl

import (
	"context"

	"git.containerum.net/ch/resource-service/pkg/db"
	rstypes "git.containerum.net/ch/resource-service/pkg/model"
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

func (rs *ResourceCountActionsImpl) GetResourcesCount(ctx context.Context) (*rstypes.GetResourcesCountResponse, error) {
	userID := httputil.MustGetUserID(ctx)
	rs.log.WithField("user_id", userID).Info("get resources count")

	ingresses, err := rs.mongo.CountIngresses(userID)
	if err != nil {
		return nil, err
	}
	deploys, err := rs.mongo.CountDeployments(userID)
	if err != nil {
		return nil, err
	}
	services, err := rs.mongo.CountService(userID)
	if err != nil {
		return nil, err
	}
	pods, err := rs.mongo.CountReplicas(userID)
	if err != nil {
		return nil, err
	}

	ret := rstypes.GetResourcesCountResponse{
		Ingresses:   ingresses,
		Deployments: deploys,
		ExtServices: services.External,
		IntServices: services.Internal,
		Pods:        pods,
	}

	return &ret, nil
}
