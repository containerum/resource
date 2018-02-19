package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"git.containerum.net/ch/resource-service/server"
)

func (rs *resourceServiceImpl) CreateStorage(ctx context.Context, req rstypes.CreateStorageRequest) error {
	rs.log.Infof("create storage %#v", req)

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		return tx.CreateStorage(ctx, req)
	})
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	return nil
}

func (rs *resourceServiceImpl) GetStorages(ctx context.Context) ([]rstypes.Storage, error) {
	rs.log.Info("get storages")

	ret, err := rs.DB.GetStorages(ctx)
	if err != nil {
		err = server.HandleDBError(err)
		return make([]rstypes.Storage, 0), err
	}

	return ret, nil
}

func (rs *resourceServiceImpl) UpdateStorage(ctx context.Context, name string, req rstypes.UpdateStorageRequest) error {
	rs.log.WithField("name", name).Info("update storage to %#v", req)

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		return tx.UpdateStorage(ctx, name, req)
	})
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	return nil
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
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	return nil
}
