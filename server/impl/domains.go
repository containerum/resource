package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"git.containerum.net/ch/resource-service/server"
)

func (rs *resourceServiceImpl) AddDomain(ctx context.Context, req rstypes.AddDomainRequest) error {
	rs.log.Info("add domain %#v", req)

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		return tx.AddDomain(ctx, req)
	})
	if err != nil {
		err = server.HandleDBError(err)
	}

	return err
}
