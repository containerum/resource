package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/cherrylog"
	"git.containerum.net/ch/resource-service/pkg/models"
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
	ResourceCountDB models.ResourceCountDBConstructor

	log *cherrylog.LogrusAdapter
}

// NewResourceServiceImpl creates a resource-service
func NewResourceServiceImpl(clients *server.ResourceServiceClients, constructors *server.ResourceServiceConstructors) *ResourceServiceImpl {
	return &ResourceServiceImpl{
		AccessActions: NewAccessActionsImpl(clients, &AccessActionsDB{
			AccessDB:    constructors.AccessDB,
			NamespaceDB: constructors.NamespaceDB,
			VolumeDB:    constructors.VolumeDB,
		}),
		DeployActions: NewDeployActionsImpl(clients, &DeployActionsDB{
			DeployDB:    constructors.DeployDB,
			NamespaceDB: constructors.NamespaceDB,
			EndpointsDB: constructors.EndpointsDB,
		}),
		DomainActions: NewDomainActionsImpl(clients, &DomainActionsDB{
			DomainDB: constructors.DomainDB,
		}),
		IngressActions: NewIngressActionsImpl(clients, &IngressActionsDB{
			NamespaceDB: constructors.NamespaceDB,
			ServiceDB:   constructors.ServiceDB,
			IngressDB:   constructors.IngressDB,
		}),
		NamespaceActions: NewNamespaceActionsImpl(clients, &NamespaceActionsDB{
			NamespaceDB: constructors.NamespaceDB,
			StorageDB:   constructors.StorageDB,
			VolumeDB:    constructors.VolumeDB,
			AccessDB:    constructors.AccessDB,
		}),
		ServiceActions: NewServiceActionsImpl(clients, &ServiceActionsDB{
			ServiceDB:   constructors.ServiceDB,
			NamespaceDB: constructors.NamespaceDB,
			DomainDB:    constructors.DomainDB,
		}),
		StorageActions: NewStorageActionsImpl(clients, &StorageActionsDB{
			StorageDB: constructors.StorageDB,
		}),
		VolumeActions: NewVolumeActionsImpl(clients, &VolumeActionsDB{
			VolumeDB:  constructors.VolumeDB,
			StorageDB: constructors.StorageDB,
			AccessDB:  constructors.AccessDB,
		}),

		ResourceServiceClients: clients,
		ResourceCountDB:        constructors.ResourceCountDB,
		log:                    cherrylog.NewLogrusAdapter(logrus.WithField("component", "resource_service")),
	}
}

func (rs *ResourceServiceImpl) GetResourcesCount(ctx context.Context) (rstypes.GetResourcesCountResponse, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithField("user_id", userID).Info("get resources count")

	ret, err := rs.ResourceCountDB(rs.DB).GetResourcesCount(ctx, userID)

	return ret, err
}
