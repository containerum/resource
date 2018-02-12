package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

func (rs *resourceServiceImpl) CreateIngress(ctx context.Context, nsLabel string, req rstypes.CreateIngressRequest) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Infof("create ingress %#v", req)

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if createErr := tx.CreateIngress(ctx, userID, nsLabel, req); createErr != nil {
			return createErr
		}

		// TODO: create ingress in kube

		return nil
	})
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	return nil
}
