package impl

import (
	"context"

	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/models"
	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

func determineServiceType(req kubtypes.Service) string {
	serviceType := "external"
	for _, port := range req.Ports {
		if port.TargetPort != nil {
			serviceType = "internal"
			break
		}
	}
	return serviceType
}

func (rs *resourceServiceImpl) CreateService(ctx context.Context, nsLabel string, req kubtypes.Service) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Infof("create service %#v", req)

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		serviceType := determineServiceType(req)

		if serviceType == "external" {
			domain, selectErr := tx.ChooseRandomDomain(ctx)
			if selectErr != nil {
				return selectErr
			}

			// TODO: choose free port of domain

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

func (rs *resourceServiceImpl) GetServices(ctx context.Context, nsLabel string) ([]kubtypes.Service, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Info("get services")

	ret, err := rs.DB.GetServices(ctx, userID, nsLabel)
	if err != nil {
		err = server.HandleDBError(err)
		return make([]kubtypes.Service, 0), err
	}

	return ret, nil
}

func (rs *resourceServiceImpl) GetService(ctx context.Context, nsLabel, serviceLabel string) (kubtypes.Service, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":       userID,
		"ns_label":      nsLabel,
		"service_label": serviceLabel,
	}).Info("get service")

	ret, err := rs.DB.GetService(ctx, userID, nsLabel, serviceLabel)
	if err != nil {
		err = models.WrapDBError(err)
	}

	return ret, err
}

func (rs *resourceServiceImpl) UpdateService(ctx context.Context, nsLabel, serviceLabel string, req kubtypes.Service) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":       userID,
		"ns_label":      nsLabel,
		"service_label": serviceLabel,
	}).Info("update service")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		serviceType := determineServiceType(req)

		if serviceType == "external" {
			domain, selectErr := tx.ChooseRandomDomain(ctx)
			if selectErr != nil {
				return selectErr
			}

			// TODO: choose free port of domain

			req.Domain = domain.Domain
			req.IPs = domain.IP
		}

		if updErr := tx.UpdateService(ctx, userID, nsLabel, serviceLabel, serviceType, req); updErr != nil {
			return updErr
		}

		// TODO: update service in kube

		return nil
	})
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	return nil
}
