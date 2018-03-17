package impl

import (
	"context"

	"strings"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
	"git.containerum.net/ch/resource-service/pkg/models"
	"git.containerum.net/ch/resource-service/pkg/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

type VolumeActionsImpl struct {
	*server.ResourceServiceClients
	log *logrus.Entry
}

func NewVolumeActionsImpl(clients *server.ResourceServiceClients) *VolumeActionsImpl {
	return &VolumeActionsImpl{
		ResourceServiceClients: clients,
		log: logrus.WithField("component", "volume_actions"),
	}
}

func (va *VolumeActionsImpl) CreateVolume(ctx context.Context, req rstypes.CreateVolumeRequest) error {
	userID := utils.MustGetUserID(ctx)
	isAdmin := server.IsAdminRole(ctx)
	va.log.WithFields(logrus.Fields{
		"tariff_id": req.TariffID,
		"label":     req.Label,
		"user_id":   userID,
		"admin":     isAdmin,
	}).Infof("creating volume for user")

	tariff, err := va.Billing.GetVolumeTariff(ctx, req.TariffID)
	if err != nil {
		return err
	}
	if chkErr := server.CheckTariff(tariff.Tariff, isAdmin); chkErr != nil {
		return chkErr
	}

	newVolume := &rstypes.Volume{
		Resource:    rstypes.Resource{TariffID: tariff.ID},
		Capacity:    tariff.StorageLimit,
		Replicas:    tariff.ReplicasLimit,
		NamespaceID: nil, // make always persistent
		GlusterName: req.Label,
	}
	newVolume.Active = new(bool) // false

	err = va.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		storage, selectErr := tx.ChooseAvailableStorage(ctx, tariff.StorageLimit)
		if selectErr != nil {
			return selectErr
		}
		newVolume.StorageID = storage.ID

		if createErr := tx.CreateVolume(ctx, userID, req.Label, newVolume); createErr != nil {
			return createErr
		}

		if subErr := va.Billing.Subscribe(ctx, userID, newVolume.Resource, rstypes.KindVolume); subErr != nil {
			return subErr
		}

		// TODO: create volume gluster

		// TODO: tariff activation

		if updErr := va.UpdateAccess(ctx, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})
	if err != nil {
		return err
	}

	if err := va.Mail.SendVolumeCreated(ctx, userID, req.Label, tariff); err != nil {
		va.log.WithError(err).Error("create volume email send failed")
	}

	return err
}

func (va *VolumeActionsImpl) DeleteUserVolume(ctx context.Context, label string) error {
	userID := utils.MustGetUserID(ctx)
	va.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Info("delete user volume")

	var volToDelete rstypes.Volume
	err := va.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if vol, delVolErr := tx.DeleteUserVolumeByLabel(ctx, userID, label); delVolErr != nil {
			return delVolErr
		} else {
			volToDelete = vol
		}

		if unsubErr := va.Billing.Unsubscribe(ctx, userID, volToDelete.Resource); unsubErr != nil {
			return unsubErr
		}

		// TODO: delete from gluster

		if updErr := va.UpdateAccess(ctx, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})
	if err != nil {
		return err
	}

	if err := va.Mail.SendVolumeDeleted(ctx, userID, label); err != nil {
		va.log.WithError(err).Error("send volume deleted email failed")
	}

	return nil
}

func (va *VolumeActionsImpl) DeleteAllUserVolumes(ctx context.Context) error {
	userID := utils.MustGetUserID(ctx)
	va.log.WithField("user_id", userID).Info("delete all user volumes")

	err := va.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if _, delErr := tx.DeleteAllUserVolumes(ctx, userID, false); delErr != nil {
			return delErr
		}

		// TODO: unsibscribe all tariffs

		// TODO: delete all volumes in gluster

		if updErr := va.UpdateAccess(ctx, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (va *VolumeActionsImpl) GetUserVolumes(ctx context.Context, filters string) (rstypes.GetUserVolumesResponse, error) {
	userID := utils.MustGetUserID(ctx)
	va.log.WithFields(logrus.Fields{
		"user_id": userID,
		"filters": filters,
	}).Info("get user volumes")

	filterstr := models.ParseVolumeFilterParams(strings.Split(filters, ",")...)
	vols, err := va.DB.GetUserVolumes(ctx, userID, &filterstr)

	return vols, err
}

func (va *VolumeActionsImpl) GetUserVolume(ctx context.Context, label string) (rstypes.GetUserVolumeResponse, error) {
	userID := utils.MustGetUserID(ctx)
	va.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Info("get user volume")

	vol, err := va.DB.GetUserVolumeByLabel(ctx, userID, label)

	return vol, err
}

func (va *VolumeActionsImpl) GetVolumesLinkedWithUserNamespace(ctx context.Context, label string) (rstypes.GetUserVolumesResponse, error) {
	userID := utils.MustGetUserID(ctx)
	va.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Info("get volumes linked with user namespace")

	vols, err := va.DB.GetVolumesLinkedWithUserNamespace(ctx, userID, label)

	return vols, err
}

func (va *VolumeActionsImpl) GetAllVolumes(ctx context.Context,
	params rstypes.GetAllResourcesQueryParams) (rstypes.GetAllVolumesResponse, error) {
	va.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
		"filters":  params.Filters,
	}).Info("get all volumes")

	filters := models.ParseVolumeFilterParams(strings.Split(params.Filters, ",")...)
	vols, err := va.DB.GetAllVolumes(ctx, params.Page, params.PerPage, &filters)

	return vols, err
}

func (va *VolumeActionsImpl) RenameUserVolume(ctx context.Context, oldLabel, newLabel string) error {
	userID := utils.MustGetUserID(ctx)
	va.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"old_label": oldLabel,
		"new_label": newLabel,
	}).Info("rename user volume")

	err := va.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if renameErr := tx.RenameVolume(ctx, userID, oldLabel, newLabel); renameErr != nil {
			return renameErr
		}

		if updErr := va.UpdateAccess(ctx, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (va *VolumeActionsImpl) ResizeUserVolume(ctx context.Context, label string, newTariffID string) error {
	userID := utils.MustGetUserID(ctx)
	isAdmin := server.IsAdminRole(ctx)
	va.log.WithFields(logrus.Fields{
		"user_id":       userID,
		"new_tariff_id": newTariffID,
		"label":         label,
		"admin":         isAdmin,
	}).Info("resize user namespace")

	err := va.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		vol, getErr := tx.GetUserVolumeByLabel(ctx, userID, label)
		if getErr != nil {
			return getErr
		}

		if vol.TariffID == newTariffID {
			return rserrors.ErrTariffUnchanged().AddDetails("can`t change tariff to itself")
		}

		newTariff, getErr := va.Billing.GetVolumeTariff(ctx, newTariffID)
		if getErr != nil {
			return getErr
		}

		if chkErr := server.CheckTariff(newTariff.Tariff, isAdmin); chkErr != nil {
			return chkErr
		}

		// TODO: maybe check if user will have exceeded quota
		vol.TariffID = newTariff.ID
		vol.Replicas = newTariff.ReplicasLimit
		vol.Capacity = newTariff.StorageLimit

		if updErr := tx.ResizeVolume(ctx, &vol.Volume); updErr != nil {
			return updErr
		}

		// TODO: resize volume in gluster

		return nil
	})
	if err != nil {
		return err
	}

	// TODO: send volume resize email

	return nil
}
