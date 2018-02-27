package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
)

func (rs *resourceServiceImpl) CreateStorage(ctx context.Context, req rstypes.CreateStorageRequest) error {
	rs.log.Infof("create storage %#v", req)

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		return tx.CreateStorage(ctx, req)
	})

	return err
}

func (rs *resourceServiceImpl) GetStorages(ctx context.Context) ([]rstypes.Storage, error) {
	rs.log.Info("get storages")

	ret, err := rs.DB.GetStorages(ctx)

	return ret, err
}

func (rs *resourceServiceImpl) UpdateStorage(ctx context.Context, name string, req rstypes.UpdateStorageRequest) error {
	rs.log.WithField("name", name).Info("update storage to %#v", req)

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		return tx.UpdateStorage(ctx, name, req)
	})

	return err
}

func (rs *resourceServiceImpl) DeleteStorage(ctx context.Context, name string) error {
	rs.log.WithField("name", name).Info("delete storage")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if delErr := tx.DeleteStorage(ctx, name); delErr != nil {
			return delErr
		}

		// TODO: do something with attached volumes

		return nil
	})

	return err
}
