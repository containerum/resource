package impl

import (
	"context"

	"strings"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

func (rs *resourceServiceImpl) CreateVolume(ctx context.Context, req *rstypes.CreateVolumeRequest) error {
	userID := utils.MustGetUserID(ctx)
	isAdmin := server.IsAdminRole(ctx)
	rs.log.WithFields(logrus.Fields{
		"tariff_id": req.TariffID,
		"label":     req.Label,
		"user_id":   userID,
		"admin":     isAdmin,
	}).Infof("creating volume for user")

	tariff, err := rs.Billing.GetVolumeTariff(ctx, req.TariffID)
	if err != nil {
		return err
	}
	if chkErr := checkTariff(tariff.Tariff, isAdmin); chkErr != nil {
		return chkErr
	}

	newVolume := &rstypes.Volume{
		Resource:   rstypes.Resource{TariffID: tariff.ID},
		Active:     new(bool),
		Capacity:   tariff.StorageLimit,
		Replicas:   tariff.ReplicasLimit,
		Persistent: true,
	}

	err = rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if createErr := tx.CreateVolume(ctx, userID, req.Label, newVolume); createErr != nil {
			return createErr
		}

		if subErr := rs.Billing.Subscribe(ctx, userID, newVolume.Resource, rstypes.KindVolume); subErr != nil {
			return subErr
		}

		// TODO: create volume gluster

		// TODO: tariff activation

		// TODO: update user access

		return nil
	})
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	go func() {
		if err := rs.Mail.SendVolumeCreated(ctx, userID, req.Label, tariff); err != nil {
			rs.log.WithError(err).Error("create volume email send failed")
		}
	}()

	return err
}

func (rs *resourceServiceImpl) DeleteUserVolume(ctx context.Context, label string) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Info("delete user volume")

	var volToDelete rstypes.Volume
	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if vol, delVolErr := tx.DeleteUserVolumeByLabel(ctx, userID, label); delVolErr != nil {
			return delVolErr
		} else {
			volToDelete = vol
		}

		if unsubErr := rs.Billing.Unsubscribe(ctx, userID, volToDelete.Resource); unsubErr != nil {
			return unsubErr
		}

		// TODO: delete from gluster

		// TODO: update auth

		return nil
	})
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	go func() {
		if err := rs.Mail.SendVolumeDeleted(ctx, userID, label); err != nil {
			rs.log.WithError(err).Error("send volume deleted email failed")
		}
	}()

	return nil
}

func (rs *resourceServiceImpl) DeleteAllUserVolumes(ctx context.Context) (err error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithField("user_id", userID).Info("delete all user volumes")

	err = rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if delErr := tx.DeleteAllUserVolumes(ctx, userID, true); err != nil {
			return delErr
		}

		// TODO: unsibscribe all tariffs

		// TODO: delete all volumes in gluster

		// TODO: update auth
		return nil
	})
	if err != nil {
		err = server.HandleDBError(err)
	}

	return
}

func (rs *resourceServiceImpl) GetUserVolumes(ctx context.Context, filters string) (rstypes.GetUserVolumesResponse, error) {
	userID := utils.MustGetUserID(ctx)
	isAdmin := server.IsAdminRole(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id": userID,
		"admin":   isAdmin,
		"filters": filters,
	}).Info("get user volumes")

	filterstr := models.ParseVolumeFilterParams(strings.Split(filters, ",")...)
	vols, err := rs.DB.GetUserVolumes(ctx, userID, &filterstr)
	if err != nil {
		err = server.HandleDBError(err)
		return nil, err
	}

	rs.filterVolumes(isAdmin, vols)

	return vols, nil
}

func (rs *resourceServiceImpl) GetUserVolume(ctx context.Context, label string) (rstypes.GetUserVolumeResponse, error) {
	userID := utils.MustGetUserID(ctx)
	isAdmin := server.IsAdminRole(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id": userID,
		"admin":   isAdmin,
		"label":   label,
	}).Info("get user volume")

	vol, err := rs.DB.GetUserVolumeByLabel(ctx, userID, label)
	if err != nil {
		err = server.HandleDBError(err)
		return rstypes.VolumeWithPermission{}, err
	}

	rs.filterVolume(isAdmin, &vol)

	return vol, nil
}

func (rs *resourceServiceImpl) GetAllVolumes(ctx context.Context,
	params *rstypes.GetAllResourcesQueryParams) (rstypes.GetAllVolumesResponse, error) {
	rs.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
		"filters":  params.Filters,
	}).Info("get all volumes")

	filters := models.ParseVolumeFilterParams(strings.Split(params.Filters, ",")...)
	vols, err := rs.DB.GetAllVolumes(ctx, params.Page, params.PerPage, &filters)
	if err != nil {
		err = server.HandleDBError(err)
		return nil, err
	}

	return vols, nil
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

func (rs *resourceServiceImpl) RenameUserVolume(ctx context.Context, oldLabel, newLabel string) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"old_label": oldLabel,
		"new_label": newLabel,
	}).Info("rename user volume")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		return tx.RenameVolume(ctx, userID, oldLabel, newLabel)
	})
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	return nil
}

func (rs *resourceServiceImpl) SetUserVolumeAccess(ctx context.Context, label string, newAccessLevel rstypes.PermissionStatus) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":          userID,
		"label":            label,
		"new_access_level": newAccessLevel,
	}).Info("change user volume access level")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		return tx.SetVolumeAccess(ctx, userID, label, newAccessLevel)
	})
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	return nil
}
