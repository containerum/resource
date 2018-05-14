package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/pkg/models"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/containerum/cherry/adaptors/cherrylog"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type DeployActionsDB struct {
	DeployDB    models.DeployDBConstructor
	NamespaceDB models.NamespaceDBConstructor
	AccessDB    models.AccessDBConstructor
}

type DeployActionsImpl struct {
	*server.ResourceServiceClients
	*DeployActionsDB

	log *cherrylog.LogrusAdapter
}

func NewDeployActionsImpl(clients *server.ResourceServiceClients, constructors *DeployActionsDB) *DeployActionsImpl {
	return &DeployActionsImpl{
		ResourceServiceClients: clients,
		DeployActionsDB:        constructors,
		log:                    cherrylog.NewLogrusAdapter(logrus.WithField("component", "deploy_actions")),
	}
}

func (da *DeployActionsImpl) GetDeployments(ctx context.Context, nsLabel string) ([]kubtypes.Deployment, error) {
	userID := httputil.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Info("get deployments")

	ret, err := da.DeployDB(da.DB).GetDeployments(ctx, userID, nsLabel)
	for i := range ret {
		if calcErr := server.CalculateDeployResources(&ret[i]); calcErr != nil {
			return nil, calcErr
		}
	}

	return ret, err
}

func (da *DeployActionsImpl) GetDeploymentByLabel(ctx context.Context, nsLabel, deplName string) (kubtypes.Deployment, error) {
	userID := httputil.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_label":    nsLabel,
		"deploy_name": deplName,
	}).Info("get deployment by label")

	ret, err := da.DeployDB(da.DB).GetDeploymentByLabel(ctx, userID, nsLabel, deplName)
	if calcErr := server.CalculateDeployResources(&ret); calcErr != nil {
		return ret, calcErr
	}

	return ret, err
}

func (da *DeployActionsImpl) CreateDeployment(ctx context.Context, nsLabel string, deploy kubtypes.Deployment) error {
	userID := httputil.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Info("create deployment")

	err := da.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		ns, getErr := da.NamespaceDB(tx).GetUserNamespaceByLabel(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if permErr := server.GetAndCheckPermission(ctx, da.AccessDB(tx), userID, rstypes.KindNamespace, nsLabel, rstypes.PermissionStatusWrite); permErr != nil {
			return permErr
		}

		nsUsage, getErr := da.NamespaceDB(tx).GetNamespaceUsage(ctx, ns.Namespace)
		if getErr != nil {
			return getErr
		}

		if chkErr := server.CheckDeploymentCreateQuotas(ns.Namespace, nsUsage, deploy); chkErr != nil {
			return chkErr
		}

		_, createErr := da.DeployDB(tx).CreateDeployment(ctx, userID, nsLabel, deploy)
		if createErr != nil {
			return createErr
		}

		if createErr := da.Kube.CreateDeployment(ctx, ns.ID, deploy); createErr != nil {
			return createErr
		}

		return nil
	})

	return err
}

func (da *DeployActionsImpl) DeleteDeployment(ctx context.Context, nsLabel, deplName string) error {
	userID := httputil.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_label":    nsLabel,
		"deploy_name": deplName,
	}).Info("delete deployment")

	err := da.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		nsID, getErr := da.NamespaceDB(tx).GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if permErr := server.GetAndCheckPermission(ctx, da.AccessDB(tx), userID, rstypes.KindNamespace, nsLabel, rstypes.PermissionStatusReadDelete); permErr != nil {
			return permErr
		}

		_, delErr := da.DeployDB(tx).DeleteDeployment(ctx, userID, nsLabel, deplName)
		if delErr != nil {
			return delErr
		}

		if delErr = da.Kube.DeleteDeployment(ctx, nsID, deplName); delErr != nil {
			return delErr
		}

		return nil
	})

	return err
}

func (da *DeployActionsImpl) ReplaceDeployment(ctx context.Context, nsLabel string, deploy kubtypes.Deployment) error {
	userID := httputil.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_label":    nsLabel,
		"deploy_name": deploy.Name,
	}).Infof("replacing deployment with %#v", deploy)

	if err := server.CalculateDeployResources(&deploy); err != nil {
		return err
	}

	err := da.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		ns, getErr := da.NamespaceDB(tx).GetUserNamespaceByLabel(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if permErr := server.GetAndCheckPermission(ctx, da.AccessDB(tx), userID, rstypes.KindNamespace, nsLabel, rstypes.PermissionStatusWrite); permErr != nil {
			return permErr
		}

		nsUsage, getErr := da.NamespaceDB(tx).GetNamespaceUsage(ctx, ns.Namespace)
		if getErr != nil {
			return getErr
		}

		oldDeploy, getErr := da.DeployDB(tx).GetDeploymentByLabel(ctx, userID, nsLabel, deploy.Name)
		if getErr != nil {
			return getErr
		}

		if chkErr := server.CheckDeploymentReplaceQuotas(ns.Namespace, nsUsage, oldDeploy, deploy); chkErr != nil {
			return chkErr
		}

		if replaceErr := da.DeployDB(tx).ReplaceDeployment(ctx, userID, nsLabel, deploy); replaceErr != nil {
			return replaceErr
		}

		if replaceErr := da.Kube.ReplaceDeployment(ctx, ns.ID, deploy); replaceErr != nil {
			return replaceErr
		}

		return nil
	})

	return err
}

func (da *DeployActionsImpl) SetDeploymentReplicas(ctx context.Context, nsLabel, deplName string, req kubtypes.UpdateReplicas) error {
	userID := httputil.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_label":    nsLabel,
		"deploy_name": deplName,
	}).Infof("set deployment replicas %#v", req)

	err := da.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		ns, getErr := da.NamespaceDB(tx).GetUserNamespaceByLabel(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if permErr := server.GetAndCheckPermission(ctx, da.AccessDB(tx), userID, rstypes.KindNamespace, nsLabel, rstypes.PermissionStatusWrite); permErr != nil {
			return permErr
		}

		deploy, getErr := da.DeployDB(tx).GetDeploymentByLabel(ctx, userID, nsLabel, deplName)
		if getErr != nil {
			return getErr
		}

		nsUsage, getErr := da.NamespaceDB(tx).GetNamespaceUsage(ctx, ns.Namespace)
		if getErr != nil {
			return getErr
		}

		if chkErr := server.CheckDeploymentReplicasChangeQuotas(ns.Namespace, nsUsage, deploy, req.Replicas); chkErr != nil {
			return chkErr
		}

		if setErr := da.DeployDB(tx).SetDeploymentReplicas(ctx, userID, nsLabel, deplName, req.Replicas); setErr != nil {
			return setErr
		}

		if setErr := da.Kube.SetDeploymentReplicas(ctx, ns.ID, deplName, req.Replicas); setErr != nil {
			return setErr
		}

		return nil
	})

	return err
}

func (da *DeployActionsImpl) SetContainerImage(ctx context.Context, nsLabel, deplName string, req kubtypes.UpdateImage) error {
	userID := httputil.MustGetUserID(ctx)
	da.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_label":    nsLabel,
		"deploy_name": deplName,
	}).Infof("set container image %#v", req)

	err := da.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		nsID, getErr := da.NamespaceDB(tx).GetNamespaceID(ctx, userID, nsLabel)
		if getErr != nil {
			return getErr
		}

		if permErr := server.GetAndCheckPermission(ctx, da.AccessDB(tx), userID, rstypes.KindNamespace, nsLabel, rstypes.PermissionStatusWrite); permErr != nil {
			return permErr
		}

		if setErr := da.DeployDB(tx).SetContainerImage(ctx, userID, nsLabel, deplName, req); setErr != nil {
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
