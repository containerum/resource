package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/pkg/models"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type ResourceCountActionsDB struct {
	ResourceCountDB models.ResourceCountDBConstructor
}

type ResourceCountActionsImpl struct {
	*server.ResourceServiceClients
	*ResourceCountActionsDB

	log *cherrylog.LogrusAdapter
}

func NewResourceCountActionsImpl(clients *server.ResourceServiceClients, constructors *ResourceCountActionsDB) *ResourceCountActionsImpl {
	return &ResourceCountActionsImpl{
		ResourceServiceClients: clients,
		ResourceCountActionsDB: constructors,
		log: cherrylog.NewLogrusAdapter(logrus.WithField("component", "resource_service")),
	}
}

func (rs *ResourceCountActionsImpl) GetResourcesCount(ctx context.Context) (rstypes.GetResourcesCountResponse, error) {
	userID := httputil.MustGetUserID(ctx)
	rs.log.WithField("user_id", userID).Info("get resources count")

	ret, err := rs.ResourceCountDB(rs.DB).GetResourcesCount(ctx, userID)

	return ret, err
}
