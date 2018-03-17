package impl

import (
	"context"

	"strings"

	"fmt"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	kubtypesInternal "git.containerum.net/ch/kube-api/pkg/model"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/pkg/models"
	"git.containerum.net/ch/resource-service/pkg/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

type NamespaceActionsImpl struct {
	*server.ResourceServiceClients
	log *logrus.Entry
}

func NewNamespaceActionsImpl(clients *server.ResourceServiceClients) *NamespaceActionsImpl {
	return &NamespaceActionsImpl{
		ResourceServiceClients: clients,
		log: logrus.WithField("component", "namespace_actions"),
	}
}

func (na *NamespaceActionsImpl) CreateNamespace(ctx context.Context, req rstypes.CreateNamespaceRequest) error {
	userID := utils.MustGetUserID(ctx)
	isAdmin := server.IsAdminRole(ctx)
	na.log.WithFields(logrus.Fields{
		"tariff_id": req.TariffID,
		"label":     req.Label,
		"user_id":   userID,
		"admin":     isAdmin,
	}).Infof("creating namespace for user")

	tariff, err := na.Billing.GetNamespaceTariff(ctx, req.TariffID)
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

	err = na.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if createErr := tx.CreateNamespace(ctx, userID, req.Label, &newNamespace); createErr != nil {
			return createErr
		}

		nsCreateRequest := kubtypesInternal.NamespaceWithOwner{
			Namespace: kubtypes.Namespace{
				Label: newNamespace.ID, // in kube we will use namespace ID as name to prevent collisions
				Resources: kubtypes.Resources{
					Hard: kubtypes.Resource{
						CPU:    fmt.Sprintf("%dm", newNamespace.CPU),
						Memory: fmt.Sprintf("%dMi", newNamespace.RAM),
					},
				},
			},
			Owner: userID,
		}
		if createErr := na.Kube.CreateNamespace(ctx, nsCreateRequest); createErr != nil {
			return createErr
		}

		if tariff.VolumeSize > 0 {
			storage, selectErr := tx.ChooseAvailableStorage(ctx, tariff.VolumeSize)
			if selectErr != nil {
				return selectErr
			}

			newVolume := rstypes.Volume{
				Resource:    rstypes.Resource{TariffID: tariff.ID},
				Capacity:    tariff.VolumeSize,
				Replicas:    2, // FIXME
				StorageID:   storage.ID,
				NamespaceID: &newNamespace.ID,
				GlusterName: server.VolumeGlusterName(nsCreateRequest.Label, userID),
			}
			newVolume.Active = new(bool) // false

			if createErr := tx.CreateVolume(ctx, userID, server.VolumeLabel(req.Label), &newVolume); createErr != nil {
				return createErr
			}

			// TODO: create volume in gluster, do not return error
		}

		if createErr := na.Billing.Subscribe(ctx, userID, newNamespace.Resource, rstypes.KindNamespace); createErr != nil {
			return createErr
		}

		// TODO: tariff activation

		if updErr := na.UpdateAccess(ctx, tx, userID); updErr != nil {
			return updErr
		}

		// TODO: create non-persistent volume

		return nil
	})
	if err != nil {
		return err
	}

	if err := na.Mail.SendNamespaceCreated(ctx, userID, req.Label, tariff); err != nil {
		na.log.WithError(err).Error("send namespace created email failed")
	}

	return nil
}

func (na *NamespaceActionsImpl) GetUserNamespaces(ctx context.Context, filters string) (rstypes.GetAllNamespacesResponse, error) {
	userID := utils.MustGetUserID(ctx)
	na.log.WithFields(logrus.Fields{
		"user_id": userID,
		"filters": filters,
	}).Info("get user namespaces")

	filterstr := models.ParseNamespaceFilterParams(strings.Split(filters, ",")...)
	ret, err := na.DB.GetUserNamespaces(ctx, userID, &filterstr)

	return ret, err
}

func (na *NamespaceActionsImpl) GetUserNamespace(ctx context.Context, label string) (rstypes.GetUserNamespaceResponse, error) {
	userID := utils.MustGetUserID(ctx)
	na.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Info("get user namespace")

	ret, err := na.DB.GetUserNamespaceWithVolumesByLabel(ctx, userID, label)

	return ret, err
}

func (na *NamespaceActionsImpl) GetAllNamespaces(ctx context.Context,
	params rstypes.GetAllResourcesQueryParams) (rstypes.GetAllNamespacesResponse, error) {
	na.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
		"filters":  params.Filters,
	}).Info("get all namespaces")

	filters := models.ParseNamespaceFilterParams(strings.Split(params.Filters, ",")...)
	ret, err := na.DB.GetAllNamespaces(ctx, params.Page, params.PerPage, &filters)

	return ret, err
}

func (na *NamespaceActionsImpl) DeleteUserNamespace(ctx context.Context, label string) error {
	userID := utils.MustGetUserID(ctx)
	na.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Info("delete user namespace")

	var nsToDelete rstypes.Namespace
	err := na.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if ns, delNsErr := tx.DeleteUserNamespaceByLabel(ctx, userID, label); delNsErr != nil {
			return delNsErr
		} else {
			nsToDelete = ns
		}

		// TODO: stop volumes on volume service

		if delErr := na.Kube.DeleteNamespace(ctx, nsToDelete.ID); delErr != nil {
			return delErr
		}

		if unsubErr := na.Billing.Unsubscribe(ctx, userID, nsToDelete.Resource); unsubErr != nil {
			return unsubErr
		}

		if updErr := na.UpdateAccess(ctx, tx, userID); updErr != nil {
			return updErr
		}

		return nil
	})
	if err != nil {
		return err
	}

	if err := na.Mail.SendNamespaceDeleted(ctx, userID, label); err != nil {
		na.log.WithError(err).Error("send namespace deleted mail failed")
	}

	return nil
}

func (na *NamespaceActionsImpl) DeleteAllUserNamespaces(ctx context.Context) error {
	userID := utils.MustGetUserID(ctx)
	na.log.WithField("user_id", userID).Info("delete all user namespaces")

	err := na.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if _, delErr := na.DB.DeleteAllUserVolumes(ctx, userID, true); delErr != nil {
			return delErr
		}

		if delErr := na.DB.DeleteAllUserNamespaces(ctx, userID); delErr != nil {
			return delErr
		}

		// TODO: stop volumes on volume service

		// TODO: unsubscribe all on billing

		if updErr := na.UpdateAccess(ctx, tx, userID); updErr != nil {
			return updErr
		}
		return nil
	})
	if err != nil {
		return err
	}

	// TODO: send email

	return nil
}

func (na *NamespaceActionsImpl) RenameUserNamespace(ctx context.Context, oldLabel, newLabel string) error {
	userID := utils.MustGetUserID(ctx)
	na.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"old_label": oldLabel,
		"new_label": newLabel,
	}).Info("rename user namespace")

	err := na.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if renErr := tx.RenameNamespace(ctx, userID, oldLabel, newLabel); renErr != nil {
			return renErr
		}
		if updErr := na.UpdateAccess(ctx, tx, userID); updErr != nil {
			return updErr
		}
		return nil
	})

	return err
}

func (na *NamespaceActionsImpl) ResizeUserNamespace(ctx context.Context, label string, newTariffID string) error {
	userID := utils.MustGetUserID(ctx)
	isAdmin := server.IsAdminRole(ctx)
	na.log.WithFields(logrus.Fields{
		"user_id":       userID,
		"new_tariff_id": newTariffID,
		"label":         label,
		"admin":         isAdmin,
	}).Info("resize user namespace")

	err := na.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		ns, getErr := tx.GetUserNamespaceByLabel(ctx, userID, label)
		if getErr != nil {
			return getErr
		}

		if ns.TariffID == newTariffID {
			return rserrors.ErrTariffUnchanged().AddDetails("can`t change tariff to itself")
		}

		newTariff, getErr := na.Billing.GetNamespaceTariff(ctx, newTariffID)
		if getErr != nil {
			return getErr
		}

		oldTariff, getErr := na.Billing.GetNamespaceTariff(ctx, ns.TariffID)
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

		nsResizeReq := kubtypesInternal.NamespaceWithOwner{
			Namespace: kubtypes.Namespace{
				Label: ns.ID,
				Resources: kubtypes.Resources{
					Hard: kubtypes.Resource{
						CPU:    fmt.Sprintf("%dm", ns.CPU),
						Memory: fmt.Sprintf("%dMi", ns.RAM),
					},
				},
			},
			Owner: userID,
		}
		if updErr := na.Kube.SetNamespaceQuota(ctx, nsResizeReq); updErr != nil {
			return updErr
		}

		// if namespace has connected volume and new tariff don`t have volumes, remove it
		if oldTariff.VolumeSize > 0 && newTariff.VolumeSize <= 0 {
			unlinkedVol, unlinkErr := tx.DeleteUserVolumeByLabel(ctx, userID, server.VolumeLabel(ns.ResourceLabel))
			if unlinkErr != nil {
				return unlinkErr
			}

			_ = unlinkedVol
			// TODO: deactivate/delete volume in gluster
		}

		// if new namespace tariff has volumes and old don`t have, create it
		if newTariff.VolumeSize > 0 && oldTariff.VolumeSize <= 0 {
			storage, selectErr := tx.ChooseAvailableStorage(ctx, newTariff.VolumeSize)
			if selectErr != nil {
				return selectErr
			}
			newVolume := rstypes.Volume{
				Resource:    rstypes.Resource{TariffID: newTariff.ID},
				Capacity:    newTariff.VolumeSize,
				Replicas:    2, // FIXME
				StorageID:   storage.ID,
				NamespaceID: &ns.ID,
				GlusterName: server.VolumeGlusterName(ns.ResourceLabel, userID),
			}
			newVolume.Active = new(bool) // false

			if createErr := tx.CreateVolume(ctx, userID, server.VolumeLabel(ns.ResourceLabel), &newVolume); createErr != nil {
				return createErr
			}

			// TODO: create volume in gluster
		}

		return nil
	})
	if err != nil {
		return err
	}

	// TODO: send namespace resized email

	return nil
}
