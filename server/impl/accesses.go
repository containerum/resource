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

func (rs *resourceServiceImpl) SetUserVolumeAccess(ctx context.Context, label string, req *rstypes.SetNamespaceAccessRequest) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"to":           req.Username,
		"label":        label,
		"access_level": req.Access,
	}).Info("change user volume access level")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		vol, getErr := tx.GetUserVolumeByLabel(ctx, userID, label)
		if getErr != nil {
			return getErr
		}

		if vol.OwnerUserID != userID {
			return server.ErrResourceNotOwned
		}

		vol.PermissionRecord.UserID = "" // FIXME: retrieve from login via user-manager
		vol.PermissionRecord.AccessLevel = req.Access

		if setErr := tx.SetResourceAccess(ctx, &vol.PermissionRecord); setErr != nil {
			return setErr
		}

		// TODO: update auth

		return nil
	})
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	return nil
}

func (rs *resourceServiceImpl) SetUserNamespaceAccess(ctx context.Context, label string, req *rstypes.SetNamespaceAccessRequest) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"to":           req.Username,
		"label":        label,
		"access_level": req.Access,
	}).Info("change user volume access level")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		ns, getErr := tx.GetUserNamespaceByLabel(ctx, userID, label)
		if getErr != nil {
			return getErr
		}

		if ns.OwnerUserID != userID {
			return server.ErrResourceNotOwned
		}

		ns.PermissionRecord.UserID = "" // FIXME: retrieve from login via user-manager
		ns.PermissionRecord.AccessLevel = req.Access

		if setErr := tx.SetResourceAccess(ctx, &ns.PermissionRecord); setErr != nil {
			return setErr
		}

		// TODO: update auth

		return nil
	})
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	return nil
}

func (rs *resourceServiceImpl) GetUserNamespaceAccesses(ctx context.Context, label string) (rstypes.GetUserNamespaceAccessesResponse, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Info("get user namespace accesses")

	ret, err := rs.DB.GetNamespaceWithUserPermissions(ctx, userID, label)
	if err != nil {
		return rstypes.GetUserNamespaceAccessesResponse{}, server.HandleDBError(err)
	}

	return ret, nil
}

func (rs *resourceServiceImpl) GetUserVolumeAccesses(ctx context.Context, label string) (rstypes.VolumeWithUserPermissions, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Info("get user volume accesses")

	ret, err := rs.DB.GetVolumeWithUserPermissions(ctx, userID, label)
	if err != nil {
		err = server.HandleDBError(err)
		return rstypes.VolumeWithUserPermissions{}, err
	}

	return ret, nil
}
