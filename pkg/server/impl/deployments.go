package impl

import (
	"context"

	"git.containerum.net/ch/resource-service/pkg/clients"
	"git.containerum.net/ch/resource-service/pkg/db"
	"git.containerum.net/ch/resource-service/pkg/models/deployment"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/containerum/cherry/adaptors/cherrylog"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type DeployActionsImpl struct {
	kube        clients.Kube
	permissions clients.Permissions
	mongo       *db.MongoStorage
	log         *cherrylog.LogrusAdapter
}

func NewDeployActionsImpl(mongo *db.MongoStorage, permissions *clients.Permissions, kube *clients.Kube) *DeployActionsImpl {
	return &DeployActionsImpl{
		kube:        *kube,
		mongo:       mongo,
		permissions: *permissions,
		log:         cherrylog.NewLogrusAdapter(logrus.WithField("component", "deploy_actions")),
	}
}

func (da *DeployActionsImpl) GetDeploymentsList(ctx context.Context, nsID string) (deployment.DeploymentList, error) {
	userID := httputil.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"namespace": nsID,
	}).Info("get deployments")

	return da.mongo.GetDeploymentList(nsID)
}

func (da *DeployActionsImpl) GetDeployment(ctx context.Context, nsID, deplName string) (*deployment.Deployment, error) {
	userID := httputil.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_id":       nsID,
		"deploy_name": deplName,
	}).Info("get deployment by label")

	ret, err := da.mongo.GetDeploymentByName(nsID, deplName)

	return &ret, err
}

func (da *DeployActionsImpl) CreateDeployment(ctx context.Context, nsID string, deploy kubtypes.Deployment) (*deployment.Deployment, error) {
	userID := httputil.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id": userID,
		"ns_id":   nsID,
	}).Info("create deployment")

	nsLimits, err := da.permissions.GetNamespaceLimits(ctx, nsID)
	if err != nil {
		return nil, err
	}

	nsUsage, err := da.mongo.GetNamespaceResourcesLimits(nsID)
	if err != nil {
		return nil, err
	}

	if err := server.CheckDeploymentCreateQuotas(nsLimits, nsUsage, deploy); err != nil {
		return nil, err
	}

	if err := da.kube.CreateDeployment(ctx, nsID, deploy); err != nil {
		return nil, err
	}

	server.CalculateDeployResources(&deploy)

	createdDeploy, err := da.mongo.CreateDeployment(deployment.DeploymentFromKube(nsID, userID, deploy))
	if err != nil {
		return nil, err
	}
	return &createdDeploy, nil
}

func (da *DeployActionsImpl) UpdateDeployment(ctx context.Context, nsID string, deploy kubtypes.Deployment) (*deployment.Deployment, error) {
	userID := httputil.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_id":       nsID,
		"deploy_name": deploy.Name,
	}).Infof("replacing deployment with %#v", deploy)

	server.CalculateDeployResources(&deploy)

	nsLimits, err := da.permissions.GetNamespaceLimits(ctx, nsID)
	if err != nil {
		return nil, err
	}

	nsUsage, err := da.mongo.GetNamespaceResourcesLimits(nsID)
	if err != nil {
		return nil, err
	}

	oldDeploy, err := da.mongo.GetDeploymentByName(nsID, deploy.Name)
	if err != nil {
		return nil, err
	}

	if err := server.CheckDeploymentReplaceQuotas(nsLimits, nsUsage, oldDeploy.Deployment, deploy); err != nil {
		return nil, err
	}

	if err := da.kube.UpdateDeployment(ctx, nsID, deploy); err != nil {
		return nil, err
	}

	if err := da.mongo.UpdateDeployment(deployment.DeploymentFromKube(nsID, userID, deploy)); err != nil {
		return nil, err
	}

	updatedDeploy, err := da.mongo.GetDeploymentByName(nsID, deploy.Name)
	if err != nil {
		return nil, err
	}

	return &updatedDeploy, nil
}

func (da *DeployActionsImpl) SetDeploymentReplicas(ctx context.Context, nsID, deplName string, req kubtypes.UpdateReplicas) (*deployment.Deployment, error) {
	userID := httputil.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_id":       nsID,
		"deploy_name": deplName,
	}).Infof("set deployment replicas %#v", req)

	nsLimits, err := da.permissions.GetNamespaceLimits(ctx, nsID)
	if err != nil {
		return nil, err
	}

	nsUsage, err := da.mongo.GetNamespaceResourcesLimits(nsID)
	if err != nil {
		return nil, err
	}

	oldDeploy, err := da.mongo.GetDeploymentByName(nsID, deplName)
	if err != nil {
		return nil, err
	}

	newDeploy := oldDeploy
	newDeploy.Replicas = req.Replicas

	if err := server.CheckDeploymentReplicasChangeQuotas(nsLimits, nsUsage, oldDeploy.Deployment, req.Replicas); err != nil {
		return nil, err
	}

	if err := da.kube.SetDeploymentReplicas(ctx, nsID, newDeploy.Name, req.Replicas); err != nil {
		return nil, err
	}

	if err := da.mongo.UpdateDeployment(newDeploy); err != nil {
		return nil, err
	}

	updatedDeploy, err := da.mongo.GetDeploymentByName(nsID, deplName)
	if err != nil {
		return nil, err
	}

	return &updatedDeploy, nil
}

func (da *DeployActionsImpl) SetDeploymentContainerImage(ctx context.Context, nsID, deplName string, req kubtypes.UpdateImage) (*deployment.Deployment, error) {
	userID := httputil.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_id":       nsID,
		"deploy_name": deplName,
	}).Infof("set container image %#v", req)

	oldDeploy, err := da.mongo.GetDeploymentByName(nsID, deplName)
	if err != nil {
		return nil, err
	}

	updated := false
	for i, c := range oldDeploy.Containers {
		if c.Name == req.Container {
			oldDeploy.Containers[i].Image = req.Image
			updated = true
			break
		}

	}
	if !updated {
		return nil, rserrors.ErrNoContainer()
	}

	if err := da.kube.SetContainerImage(ctx, nsID, oldDeploy.Name, req); err != nil {
		return nil, err
	}

	err = da.mongo.UpdateDeployment(oldDeploy)
	if err != nil {
		return nil, err
	}

	updatedDeploy, err := da.mongo.GetDeploymentByName(nsID, deplName)
	if err != nil {
		return nil, err
	}

	return &updatedDeploy, nil
}

func (da *DeployActionsImpl) DeleteDeployment(ctx context.Context, nsID, deplName string) error {
	userID := httputil.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_id":       nsID,
		"deploy_name": deplName,
	}).Info("delete deployment")

	if err := da.kube.DeleteDeployment(ctx, nsID, deplName); err != nil {
		return err
	}

	if err := da.mongo.DeleteDeployment(nsID, deplName); err != nil {
		return err
	}
	return nil
}

func (da *DeployActionsImpl) DeleteAllDeployments(ctx context.Context, nsID string) error {
	da.log.WithFields(logrus.Fields{
		"ns_id": nsID,
	}).Info("delete all deployments")

	if err := da.mongo.DeleteAllDeployments(nsID); err != nil {
		return err
	}
	return nil
}
