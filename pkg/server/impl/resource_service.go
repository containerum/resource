package impl

import (
	"context"

	"git.containerum.net/ch/resource-service/pkg/db"
	rstypes "git.containerum.net/ch/resource-service/pkg/model"
	"github.com/containerum/cherry/adaptors/cherrylog"
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
	/*userID := httputil.MustGetUserID(ctx)
	rs.log.WithField("user_id", userID).Info("get resources count")

	ret, err := rs.ResourceCountDB(rs.DB).GetResourcesCount(ctx, userID)

	return ret, err*/
	return nil, nil
}
