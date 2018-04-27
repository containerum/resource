package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/pkg/models"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/sirupsen/logrus"
)

type StorageActionsDB struct {
	StorageDB models.StorageDBConstructor
}

type StorageActionsImpl struct {
	*server.ResourceServiceClients
	*StorageActionsDB

	log *cherrylog.LogrusAdapter
}

func NewStorageActionsImpl(clients *server.ResourceServiceClients, constructors *StorageActionsDB) *StorageActionsImpl {
	return &StorageActionsImpl{
		ResourceServiceClients: clients,
		log: cherrylog.NewLogrusAdapter(logrus.WithField("component", "storage_actions")),
	}
}

func (sa *StorageActionsImpl) CreateStorage(ctx context.Context, req rstypes.CreateStorageRequest) error {
	sa.log.Infof("create storage %#v", req)

	err := sa.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		return sa.StorageDB(tx).CreateStorage(ctx, req)
	})

	return err
}

func (sa *StorageActionsImpl) GetStorages(ctx context.Context) ([]rstypes.Storage, error) {
	sa.log.Info("get storages")

	ret, err := sa.StorageDB(sa.DB).GetStorages(ctx)

	return ret, err
}

func (sa *StorageActionsImpl) UpdateStorage(ctx context.Context, name string, req rstypes.UpdateStorageRequest) error {
	sa.log.WithField("name", name).Info("update storage to %#v", req)

	err := sa.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		return sa.StorageDB(tx).UpdateStorage(ctx, name, req)
	})

	return err
}

func (sa *StorageActionsImpl) DeleteStorage(ctx context.Context, name string) error {
	sa.log.WithField("name", name).Info("delete storage")

	err := sa.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		if delErr := sa.StorageDB(tx).DeleteStorage(ctx, name); delErr != nil {
			return delErr
		}

		// TODO: do something with attached volumes

		return nil
	})

	return err
}
