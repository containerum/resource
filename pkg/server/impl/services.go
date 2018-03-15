package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	kubtypesInternal "git.containerum.net/ch/kube-api/pkg/model"
	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/pkg/models"
	"git.containerum.net/ch/resource-service/pkg/server"
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
		serviceType := server.DetermineServiceType(req)

		kubeRequest := kubtypesInternal.ServiceWithOwner{
			Service: req,
			Owner:   userID,
		}

		if serviceType == rstypes.ServiceExternal {
			domain, selectErr := tx.ChooseRandomDomain(ctx)
			if selectErr != nil {
				return selectErr
			}

			kubeRequest.Domain = domain.Domain
			kubeRequest.IPs = domain.IP
			// TODO: SQL queries in loop is not good solution
			for i := range kubeRequest.Ports {
				port, portSelectErr := tx.ChooseDomainFreePort(ctx, domain.Domain, kubeRequest.Ports[i].Protocol)
				if portSelectErr != nil {
					return portSelectErr
				}
				kubeRequest.Ports[i].Port = port
			}
		}

		nsID, getErr := tx.GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if createErr := tx.CreateService(ctx, userID, nsLabel, serviceType, kubeRequest.Service); createErr != nil {
			return createErr
		}

		if createErr := rs.Kube.CreateService(ctx, nsID, kubeRequest); createErr != nil {
			return createErr
		}

		return nil
	})

	return err
}

func (rs *resourceServiceImpl) GetServices(ctx context.Context, nsLabel string) ([]kubtypes.Service, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Info("get services")

	ret, err := rs.DB.GetServices(ctx, userID, nsLabel)

	return ret, err
}

func (rs *resourceServiceImpl) GetService(ctx context.Context, nsLabel, serviceName string) (kubtypes.Service, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"service_name": serviceName,
	}).Info("get service")

	ret, err := rs.DB.GetService(ctx, userID, nsLabel, serviceName)

	return ret, err
}

func (rs *resourceServiceImpl) UpdateService(ctx context.Context, nsLabel, serviceName string, req kubtypes.Service) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"service_name": serviceName,
	}).Info("update service")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		serviceType := server.DetermineServiceType(req)

		kubeRequest := kubtypesInternal.ServiceWithOwner{
			Service: req,
			Owner:   userID,
		}

		if serviceType == rstypes.ServiceExternal {
			domain, selectErr := tx.ChooseRandomDomain(ctx)
			if selectErr != nil {
				return selectErr
			}

			kubeRequest.Domain = domain.Domain
			kubeRequest.IPs = domain.IP
			for i := range kubeRequest.Ports {
				port, portSelectErr := tx.ChooseDomainFreePort(ctx, domain.Domain, kubeRequest.Ports[i].Protocol)
				if portSelectErr != nil {
					return portSelectErr
				}
				kubeRequest.Ports[i].Port = port
			}
		}

		nsID, getErr := tx.GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if updErr := tx.UpdateService(ctx, userID, nsLabel, serviceName, serviceType, kubeRequest.Service); updErr != nil {
			return updErr
		}

		if updErr := rs.Kube.UpdateService(ctx, nsID, serviceName, kubeRequest); updErr != nil {
			return updErr
		}

		return nil
	})

	return err
}

func (rs *resourceServiceImpl) DeleteService(ctx context.Context, nsLabel, serviceName string) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"service_name": serviceName,
	}).Info("delete service")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		nsID, getErr := tx.GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if delErr := tx.DeleteService(ctx, userID, nsLabel, serviceName); delErr != nil {
			return delErr
		}

		if delErr := rs.Kube.DeleteService(ctx, nsID, serviceName); delErr != nil {
			return delErr
		}

		return nil
	})

	return err
}
