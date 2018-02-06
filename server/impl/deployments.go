package impl

import (
	"context"

	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
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
