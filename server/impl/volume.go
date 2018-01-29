package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

func (rs *resourceServiceImpl) CreateVolume(ctx context.Context, req *rstypes.CreateVolumeRequest) (err error) {
	userID := utils.MustGetUserID(ctx)
	isAdmin := server.IsAdminRole(ctx)
	rs.log.WithFields(logrus.Fields{
		"tariff_id": req.TariffID,
		"label":     req.Label,
		"user_id":   userID,
		"admin":     isAdmin,
	}).Infof("creating namespace for user")

	tariff, err := rs.Billing.GetVolumeTariff(ctx, req.TariffID)
	if err != nil {
		return
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

		// TODO: create volume in kube

		// TODO: tariff activation

		// TODO: update user access

		return nil
	})
	if err != nil {
		err = server.HandleDBError(err)
	}

	go func() {
		if err := rs.Mail.SendVolumeCreated(ctx, userID, req.Label, tariff); err != nil {
			rs.log.WithError(err).Error("create volume email send failed")
		}
	}()

	return
}
