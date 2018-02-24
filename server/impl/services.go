package impl

import (
	"context"

	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/models"
	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

func (rs *resourceServiceImpl) CreateService(ctx context.Context, nsLabel string, req kubtypes.Service) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Infof("create service %#v", req)

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		serviceType := "external"
		for _, port := range req.Ports {
			if port.TargetPort != nil {
				serviceType = "internal"
				break
			}
		}

		if serviceType == "external" {
			domain, selectErr := tx.ChooseRandomDomain(ctx)
			if selectErr != nil {
				return selectErr
			}

			req.Domain = domain.Domain
			req.IPs = domain.IP
		}

		if createErr := tx.CreateService(ctx, userID, nsLabel, serviceType, req); createErr != nil {
			return createErr
		}

		// TODO: create service in kube

		return nil
	})
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	return nil
}
