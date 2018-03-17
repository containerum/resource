package impl

import (
	"context"

	kubtypesInternal "git.containerum.net/ch/kube-api/pkg/model"
	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/pkg/models"
	"git.containerum.net/ch/resource-service/pkg/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

type DeployActionsImpl struct {
	*server.ResourceServiceClients
	log *logrus.Entry
}

func NewDeployActionsImpl(clients *server.ResourceServiceClients) *DeployActionsImpl {
	return &DeployActionsImpl{
		ResourceServiceClients: clients,
		log: logrus.WithField("component", "deploy_actions"),
	}
}

func (da *DeployActionsImpl) GetDeployments(ctx context.Context, nsLabel string) ([]kubtypes.Deployment, error) {
	userID := utils.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Info("get deployments")

	ret, err := da.DB.GetDeployments(ctx, userID, nsLabel)

	return ret, err
}

func (da *DeployActionsImpl) GetDeploymentByLabel(ctx context.Context, nsLabel, deplName string) (kubtypes.Deployment, error) {
	userID := utils.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_label":    nsLabel,
		"deploy_name": deplName,
	}).Info("get deployment by label")

	ret, err := da.DB.GetDeploymentByLabel(ctx, userID, nsLabel, deplName)

	return ret, err
}

func (da *DeployActionsImpl) CreateDeployment(ctx context.Context, nsLabel string, deploy kubtypes.Deployment) error {
	userID := utils.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Info("create deployment")

	err := da.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
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
		if createErr := da.Kube.CreateDeployment(ctx, nsID, deployCreateReq); createErr != nil {
			return createErr
		}

		return nil
	})

	return err
}

func (da *DeployActionsImpl) DeleteDeployment(ctx context.Context, nsLabel, deplName string) error {
	userID := utils.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_label":    nsLabel,
		"deploy_name": deplName,
	}).Info("delete deployment")

	err := da.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		nsID, getErr := tx.GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		lastInNamespace, delErr := tx.DeleteDeployment(ctx, userID, nsLabel, deplName)
		if delErr != nil {
			return delErr
		}

		if delErr = da.Kube.DeleteDeployment(ctx, nsID, deplName); delErr != nil {
			return delErr
		}

		if lastInNamespace {
			// TODO: deactivate volume in gluster
		}

		return nil
	})

	return err
}

func (da *DeployActionsImpl) ReplaceDeployment(ctx context.Context, nsLabel string, deploy kubtypes.Deployment) error {
	userID := utils.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_label":    nsLabel,
		"deploy_name": deploy.Name,
	}).Infof("replacing deployment with %#v", deploy)

	err := da.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		nsID, getErr := tx.GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if replaceErr := tx.ReplaceDeployment(ctx, userID, nsLabel, deploy); replaceErr != nil {
			return replaceErr
		}

		deployReplaceReq := kubtypesInternal.DeploymentWithOwner{}
		deployReplaceReq.Deployment = deploy
		deployReplaceReq.Owner = userID
		if replaceErr := da.Kube.ReplaceDeployment(ctx, nsID, deployReplaceReq); replaceErr != nil {
			return replaceErr
		}

		return nil
	})

	return err
}

func (da *DeployActionsImpl) SetDeploymentReplicas(ctx context.Context, nsLabel, deplName string, req kubtypes.UpdateReplicas) error {
	userID := utils.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_label":    nsLabel,
		"deploy_name": deplName,
	}).Infof("set deployment replicas %#v", req)

	err := da.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		nsID, getErr := tx.GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if setErr := tx.SetDeploymentReplicas(ctx, userID, nsLabel, deplName, req.Replicas); setErr != nil {
			return setErr
		}

		if setErr := da.Kube.SetDeploymentReplicas(ctx, nsID, deplName, req.Replicas); setErr != nil {
			return setErr
		}

		return nil
	})

	return err
}

func (da *DeployActionsImpl) SetContainerImage(ctx context.Context, nsLabel, deplName string, req kubtypes.UpdateImage) error {
	userID := utils.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_label":    nsLabel,
		"deploy_name": deplName,
	}).Infof("set container image %#v", req)

	err := da.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		nsID, getErr := tx.GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if setErr := tx.SetContainerImage(ctx, userID, nsLabel, deplName, req); setErr != nil {
			return setErr
		}

		setErr := da.Kube.SetContainerImage(ctx, nsID, deplName, req)
		if setErr != nil {
			return setErr
		}

		return nil
	})

	return err
}
