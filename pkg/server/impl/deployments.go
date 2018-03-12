package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	kubtypesInternal "git.containerum.net/ch/kube-api/pkg/model"
	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/pkg/models"
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

	return ret, err
}

func (rs *resourceServiceImpl) GetDeploymentByLabel(ctx context.Context, nsLabel, deplLabel string) (kubtypes.Deployment, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"deploy_label": deplLabel,
	}).Info("get deployment by label")

	ret, err := rs.DB.GetDeploymentByLabel(ctx, userID, nsLabel, deplLabel)

	return ret, err
}

func (rs *resourceServiceImpl) CreateDeployment(ctx context.Context, nsLabel string, deploy kubtypes.Deployment) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Info("create deployment")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		nsID, getErr := tx.GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		firstInNamespace, createErr := tx.CreateDeployment(ctx, userID, nsLabel, deploy)
		if createErr != nil {
			return createErr
		}

		if firstInNamespace {
			// TODO: activate volume in gluster
		}

		newEndpoints, epErr := tx.CreateGlusterEndpoints(ctx, userID, nsLabel)
		if epErr != nil {
			return epErr
		}

		for _, ep := range newEndpoints {
			// TODO: create new endpoint in kube
			// TODO: create gluster service in kube
			_ = ep
		}

		if confErr := tx.ConfirmGlusterEndpoints(ctx, userID, nsLabel); confErr != nil {
			return confErr
		}

		deployCreateReq := kubtypesInternal.DeploymentWithOwner{}
		deployCreateReq.Deployment = deploy
		deployCreateReq.Owner = userID
		if createErr := rs.Kube.CreateDeployment(ctx, nsID, deployCreateReq); createErr != nil {
			return createErr
		}

		return nil
	})

	return err
}

func (rs *resourceServiceImpl) DeleteDeployment(ctx context.Context, nsLabel, deplLabel string) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"deploy_label": deplLabel,
	}).Info("delete deployment")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		nsID, getErr := tx.GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		lastInNamespace, delErr := tx.DeleteDeployment(ctx, userID, nsLabel, deplLabel)
		if delErr != nil {
			return delErr
		}

		if delErr = rs.Kube.DeleteDeployment(ctx, nsID, deplLabel); delErr != nil {
			return delErr
		}

		if lastInNamespace {
			// TODO: deactivate volume in gluster
		}

		return nil
	})

	return err
}

func (rs *resourceServiceImpl) ReplaceDeployment(ctx context.Context, nsLabel, deplLabel string, deploy kubtypes.Deployment) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"deploy_label": deplLabel,
	}).Infof("replacing deployment with %#v", deploy)

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		nsID, getErr := tx.GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if replaceErr := tx.ReplaceDeployment(ctx, userID, nsLabel, deplLabel, deploy); replaceErr != nil {
			return replaceErr
		}

		deployReplaceReq := kubtypesInternal.DeploymentWithOwner{}
		deployReplaceReq.Deployment = deploy
		deployReplaceReq.Owner = userID
		if replaceErr := rs.Kube.ReplaceDeployment(ctx, nsID, deplLabel, deployReplaceReq); replaceErr != nil {
			return replaceErr
		}

		return nil
	})

	return err
}

func (rs *resourceServiceImpl) SetDeploymentReplicas(ctx context.Context, nsLabel, deplLabel string, req rstypes.SetReplicasRequest) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"deploy_label": deplLabel,
	}).Infof("set deployment replicas %#v", req)

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		nsID, getErr := tx.GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if setErr := tx.SetDeploymentReplicas(ctx, userID, nsLabel, deplLabel, req.Replicas); setErr != nil {
			return setErr
		}

		if setErr := rs.Kube.SetDeploymentReplicas(ctx, nsID, deplLabel, req.Replicas); setErr != nil {
			return setErr
		}

		return nil
	})

	return err
}

func (rs *resourceServiceImpl) SetContainerImage(ctx context.Context, nsLabel, deplLabel string, req rstypes.SetContainerImageRequest) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"deploy_label": deplLabel,
	}).Infof("set container image %#v", req)

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		nsID, getErr := tx.GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if setErr := tx.SetContainerImage(ctx, userID, nsLabel, deplLabel, req); setErr != nil {
			return setErr
		}

		setErr := rs.Kube.SetContainerImage(ctx, nsID, deplLabel, kubtypes.Container{Name: req.ContainerName, Image: req.Image})
		if setErr != nil {
			return setErr
		}

		return nil
	})

	return err
}
