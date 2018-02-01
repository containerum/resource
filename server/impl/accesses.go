package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

func (rs *resourceServiceImpl) SetUserAccesses(ctx context.Context, accessLevel rstypes.PermissionStatus) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"access_level": accessLevel,
	}).Info("set user resources access level")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		// TODO: update auth
		return tx.SetAllResourcesAccess(ctx, userID, accessLevel)
	})
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	return nil
}
