package impl

import (
	"context"

	"strings"

	"git.containerum.net/ch/json-types/misc"
	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

func (rs *resourceServiceImpl) CreateNamespace(ctx context.Context, req *rstypes.CreateNamespaceRequest) error {
	userID := utils.MustGetUserID(ctx)
	isAdmin := server.IsAdminRole(ctx)
	rs.log.WithFields(logrus.Fields{
		"tariff_id": req.TariffID,
		"label":     req.Label,
		"user_id":   userID,
		"admin":     isAdmin,
	}).Infof("creating namespace for user")

	tariff, err := rs.Billing.GetNamespaceTariff(ctx, req.TariffID)
	if err != nil {
		return err
	}

	if chkErr := server.CheckTariff(tariff.Tariff, isAdmin); chkErr != nil {
		return chkErr
	}

	newNamespace := rstypes.Namespace{
		Resource:            rstypes.Resource{TariffID: tariff.ID},
		RAM:                 tariff.MemoryLimit,
		CPU:                 tariff.CPULimit,
		MaxExternalServices: tariff.ExternalServices,
		MaxIntServices:      tariff.InternalServices,
		MaxTraffic:          tariff.Traffic,
	}

	err = rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if createErr := tx.CreateNamespace(ctx, userID, req.Label, &newNamespace); createErr != nil {
			return createErr
		}

		if createErr := rs.Kube.CreateNamespace(ctx, req.Label, rstypes.NamespaceWithPermission{
			Namespace:        newNamespace,
			PermissionRecord: rstypes.PermissionRecord{AccessLevel: rstypes.PermissionStatusOwner},
		}); createErr != nil {
			return createErr
		}

		if tariff.VolumeSize > 0 {
			newVolume := rstypes.Volume{
				Resource:   rstypes.Resource{TariffID: tariff.ID},
				Active:     misc.WrapBool(false),
				Capacity:   tariff.VolumeSize,
				Replicas:   2, // FIXME
				Persistent: false,
			}

			if createErr := tx.CreateVolume(ctx, userID, server.VolumeLabelFromNamespaceLabel(req.Label), &newVolume); createErr != nil {
				return createErr
			}

			// TODO: create volume in gluster, do not return error
		}

		if createErr := rs.Billing.Subscribe(ctx, userID, newNamespace.Resource, rstypes.KindNamespace); createErr != nil {
			return createErr
		}

		// TODO: tariff activation

		// TODO: update user access

		// TODO: create non-persistent volume

		return nil
	})
	if err = server.HandleDBError(err); err != nil {
		return err
	}

	if err := rs.Mail.SendNamespaceCreated(ctx, userID, req.Label, tariff); err != nil {
		logrus.WithError(err).Error("send namespace created email failed")
	}

	return nil
}

func (rs *resourceServiceImpl) GetUserNamespaces(ctx context.Context, filters string) (rstypes.GetAllNamespacesResponse, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id": userID,
		"filters": filters,
	}).Info("get user namespaces")

	filterstr := models.ParseNamespaceFilterParams(strings.Split(filters, ",")...)
	ret, err := rs.DB.GetUserNamespaces(ctx, userID, &filterstr)
	if err != nil {
		return nil, server.HandleDBError(err)
	}

	return ret, nil
}

func (rs *resourceServiceImpl) GetUserNamespace(ctx context.Context, label string) (rstypes.GetUserNamespaceResponse, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Info("get user namespace")

	ret, err := rs.DB.GetUserNamespaceWithVolumesByLabel(ctx, userID, label)
	if err != nil {
		return rstypes.NamespaceWithVolumes{}, server.HandleDBError(err)
	}

	return ret, nil
}

func (rs *resourceServiceImpl) GetAllNamespaces(ctx context.Context,
	params *rstypes.GetAllResourcesQueryParams) (rstypes.GetAllNamespacesResponse, error) {
	rs.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
		"filters":  params.Filters,
	}).Info("get all namespaces")

	filters := models.ParseNamespaceFilterParams(strings.Split(params.Filters, ",")...)
	ret, err := rs.DB.GetAllNamespaces(ctx, params.Page, params.PerPage, &filters)
	if err != nil {
		return nil, server.HandleDBError(err)
	}

	return ret, nil
}

func (rs *resourceServiceImpl) DeleteUserNamespace(ctx context.Context, label string) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Info("delete user namespace")

	var nsToDelete rstypes.Namespace
	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if ns, delNsErr := tx.DeleteUserNamespaceByLabel(ctx, userID, label); delNsErr != nil {
			return delNsErr
		} else {
			nsToDelete = ns
		}

		// TODO: stop volumes on volume service

		if delErr := rs.Kube.DeleteNamespace(ctx, nsToDelete); delErr != nil {
			return delErr
		}

		if unsubErr := rs.Billing.Unsubscribe(ctx, userID, nsToDelete.Resource); unsubErr != nil {
			return unsubErr
		}

		// TODO: update user access on auth service

		return nil
	})
	if err != nil {
		return server.HandleDBError(err)
	}

	if err := rs.Mail.SendNamespaceDeleted(ctx, userID, label); err != nil {
		logrus.WithError(err).Error("send namespace deleted mail failed")
	}

	return nil
}

func (rs *resourceServiceImpl) DeleteAllUserNamespaces(ctx context.Context) error {
	userID := utils.MustGetUserID(ctx)
	logrus.WithField("user_id", userID).Info("delete all user namespaces")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if _, delErr := rs.DB.DeleteAllUserVolumes(ctx, userID, true); delErr != nil {
			return delErr
		}

		if delErr := rs.DB.DeleteAllUserNamespaces(ctx, userID); delErr != nil {
			return delErr
		}

		// TODO: stop volumes on volume service

		// TODO: unsubscribe all on billing

		// TODO: update user access on auth
		return nil
	})
	if err != nil {
		return server.HandleDBError(err)
	}

	// TODO: send email

	return nil
}

func (rs *resourceServiceImpl) RenameUserNamespace(ctx context.Context, oldLabel, newLabel string) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"old_label": oldLabel,
		"new_label": newLabel,
	}).Info("rename user namespace")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		return tx.RenameNamespace(ctx, userID, oldLabel, newLabel)
	})
	if err != nil {
		return server.HandleDBError(err)
	}

	return nil
}

func (rs *resourceServiceImpl) ResizeUserNamespace(ctx context.Context, label string, newTariffID string) error {
	userID := utils.MustGetUserID(ctx)
	isAdmin := server.IsAdminRole(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":       userID,
		"new_tariff_id": newTariffID,
		"label":         label,
		"admin":         isAdmin,
	}).Info("resize user namespace")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		ns, getErr := tx.GetUserNamespaceByLabel(ctx, userID, label)
		if getErr != nil {
			return getErr
		}

		if ns.TariffID == newTariffID {
			return server.ErrTariffIsSame
		}

		newTariff, getErr := rs.Billing.GetNamespaceTariff(ctx, newTariffID)
		if getErr != nil {
			return getErr
		}

		oldTariff, getErr := rs.Billing.GetNamespaceTariff(ctx, ns.TariffID)
		if getErr != nil {
			return getErr
		}

		if chkErr := server.CheckTariff(newTariff.Tariff, isAdmin); chkErr != nil {
			return chkErr
		}

		// TODO: maybe check if user will have exceeded quota
		ns.TariffID = newTariff.ID
		ns.CPU = newTariff.CPULimit
		ns.RAM = newTariff.MemoryLimit
		ns.MaxExternalServices = newTariff.ExternalServices
		ns.MaxIntServices = newTariff.InternalServices
		ns.MaxTraffic = newTariff.Traffic

		if updErr := tx.ResizeNamespace(ctx, &ns.Namespace); updErr != nil {
			return updErr
		}

		if updErr := rs.Kube.SetNamespaceQuota(ctx, label, ns); updErr != nil {
			return updErr
		}

		// if namespace has connected volume and new tariff don`t have volumes, remove it
		if oldTariff.VolumeSize > 0 && newTariff.VolumeSize <= 0 {
			unlinkedVol, unlinkErr := tx.DeleteUserVolumeByLabel(ctx, userID, server.VolumeLabelFromNamespaceLabel(ns.ResourceLabel))
			if unlinkErr != nil {
				return unlinkErr
			}

			_ = unlinkedVol
			// TODO: deactivate/delete volume in gluster
		}

		// if new namespace tariff has volumes and old don`t have, create it
		if newTariff.VolumeSize > 0 && oldTariff.VolumeSize <= 0 {
			newVolume := rstypes.Volume{
				Resource:   rstypes.Resource{TariffID: newTariff.ID},
				Active:     misc.WrapBool(false),
				Capacity:   newTariff.VolumeSize,
				Replicas:   2, // FIXME
				Persistent: false,
			}

			if createErr := tx.CreateVolume(ctx, userID, server.VolumeLabelFromNamespaceLabel(ns.ResourceLabel), &newVolume); createErr != nil {
				return createErr
			}

			// TODO: create volume in gluster
		}

		return nil
	})
	if err != nil {
		return server.HandleDBError(err)
	}

	// TODO: send namespace resized email

	return nil
}
