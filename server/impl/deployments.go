package impl

import (
	"context"

	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/models"
	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

func (rs *resourceServiceImpl) GetDeployments(ctx context.Context, nsLabel string) ([]kubtypes.Deployment, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Info("get deployments")

	ret, err := rs.DB.GetDeployments(ctx, userID, nsLabel)
	if err != nil {
		err = server.HandleDBError(err)
		return nil, err
	}

	return ret, nil
}

func (rs *resourceServiceImpl) GetDeploymentByLabel(ctx context.Context, nsLabel, deplLabel string) (kubtypes.Deployment, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"deploy_label": deplLabel,
	}).Info("get deployment by label")

	ret, err := rs.DB.GetDeploymentByLabel(ctx, userID, nsLabel, deplLabel)
	if err != nil {
		err = server.HandleDBError(err)
		return kubtypes.Deployment{}, err
	}

	return ret, nil
}

func (rs *resourceServiceImpl) CreateDeployment(ctx context.Context, nsLabel string, deploy kubtypes.Deployment) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Info("create deployment")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		firstInNamespace, createErr := tx.CreateDeployment(ctx, userID, nsLabel, deploy)
		if createErr != nil {
			return createErr
		}

		if firstInNamespace {
			// TODO: activate volume in gluster
		}

		// TODO: create deployment in kube

		return nil
	})
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	return nil
}

func (rs *resourceServiceImpl) DeleteDeployment(ctx context.Context, nsLabel, deplLabel string) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"deploy_label": deplLabel,
	}).Info("delete deployment")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		lastInNamespace, delErr := tx.DeleteDeployment(ctx, userID, nsLabel, deplLabel)
		if delErr != nil {
			return delErr
		}

		// TODO: delete deployment in kube

		if lastInNamespace {
			// TODO: deactivate volume in gluster
		}

		return nil
	})
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	return nil
}

func (rs *resourceServiceImpl) ReplaceDeployment(ctx context.Context, nsLabel, deplLabel string, deploy kubtypes.Deployment) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"deploy_label": deplLabel,
	}).Infof("replacing deployment with %#v", deploy)

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if replaceErr := tx.ReplaceDeployment(ctx, userID, nsLabel, deplLabel, deploy); replaceErr != nil {
			return replaceErr
		}

		// TODO: replace deploy in kube

		return nil
	})
	if err != nil {
		err = server.HandleDBError(err)
		return err
	}

	return nil
}
