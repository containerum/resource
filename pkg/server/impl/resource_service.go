package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/cherrylog"
	"git.containerum.net/ch/resource-service/pkg/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

type ResourceServiceImpl struct {
	server.AccessActions
	server.DeployActions
	server.DomainActions
	server.IngressActions
	server.NamespaceActions
	server.ServiceActions
	server.StorageActions
	server.VolumeActions

	*server.ResourceServiceClients
	log *cherrylog.LogrusAdapter
}

// NewResourceServiceImpl creates a resource-service
func NewResourceServiceImpl(clients *server.ResourceServiceClients) *ResourceServiceImpl {
	return &ResourceServiceImpl{
		AccessActions:    NewAccessActionsImpl(clients),
		DeployActions:    NewDeployActionsImpl(clients),
		DomainActions:    NewDomainActionsImpl(clients),
		IngressActions:   NewIngressActionsImpl(clients),
		NamespaceActions: NewNamespaceActionsImpl(clients),
		ServiceActions:   NewServiceActionsImpl(clients),
		StorageActions:   NewStorageActionsImpl(clients),
		VolumeActions:    NewVolumeActionsImpl(clients),

		ResourceServiceClients: clients,
		log: cherrylog.NewLogrusAdapter(logrus.WithField("component", "resource_service")),
	}
}

func (rs *ResourceServiceImpl) GetResourcesCount(ctx context.Context) (rstypes.GetResourcesCountResponse, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithField("user_id", userID).Info("get resources count")

	ret, err := rs.DB.GetResourcesCount(ctx, userID)

	return ret, err
}
