package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/pkg/models"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/sirupsen/logrus"
)

type StorageActionsImpl struct {
	*server.ResourceServiceClients
	log *logrus.Entry
}

func NewStorageActionsImpl(clients *server.ResourceServiceClients) *StorageActionsImpl {
	return &StorageActionsImpl{
		ResourceServiceClients: clients,
		log: logrus.WithField("component", "storage_actions"),
	}
}

func (sa *StorageActionsImpl) CreateStorage(ctx context.Context, req rstypes.CreateStorageRequest) error {
	sa.log.Infof("create storage %#v", req)

	err := sa.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		return tx.CreateStorage(ctx, req)
	})

	return err
}

func (sa *StorageActionsImpl) GetStorages(ctx context.Context) ([]rstypes.Storage, error) {
	sa.log.Info("get storages")

	ret, err := sa.DB.GetStorages(ctx)

	return ret, err
}

func (sa *StorageActionsImpl) UpdateStorage(ctx context.Context, name string, req rstypes.UpdateStorageRequest) error {
	sa.log.WithField("name", name).Info("update storage to %#v", req)

	err := sa.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		return tx.UpdateStorage(ctx, name, req)
	})

	return err
}

func (sa *StorageActionsImpl) DeleteStorage(ctx context.Context, name string) error {
	sa.log.WithField("name", name).Info("delete storage")

	err := sa.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if delErr := tx.DeleteStorage(ctx, name); delErr != nil {
			return delErr
		}

		// TODO: do something with attached volumes

		return nil
	})

	return err
}
