package postgres

import (
	"context"

	"database/sql"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/models"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type volumeMountWithName struct {
	Name string `db:"resource_label"`
	rstypes.VolumeMount
}

func (db *pgDB) getContainersVolumes(ctx context.Context,
	containerIDs []string) (volMap map[string][]volumeMountWithName, err error) {
	db.log.Debugf("get containers volumes %v", containerIDs)

	volMap = make(map[string][]volumeMountWithName)
	vols := make([]volumeMountWithName, 0)
	query, args, _ := sqlx.In( /* language=sql */
		`SELECT p.resource_label, vm.* 
				FROM volume_mounts vm
				JOIN permissions p ON vm.volume_id = p.resource_id AND p.kind = 'volume'
				WHERE vm.container_id IN (?)`,
		containerIDs)
	err = sqlx.SelectContext(ctx, db.extLog, &vols, db.extLog.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
	default:
		err = models.WrapDBError(err)
		return
	}

	for _, vol := range vols {
		tmp := volMap[vol.ContainerID]
		tmp = append(tmp, vol)
		volMap[vol.ContainerID] = tmp
	}

	return
}

func (db *pgDB) getContainersEnvironments(ctx context.Context,
	containerIDs []string) (envMap map[string][]rstypes.EnvironmentVariable, err error) {
	db.log.Debugf("get containers envs %v", containerIDs)

	envMap = make(map[string][]rstypes.EnvironmentVariable)
	envs := make([]rstypes.EnvironmentVariable, 0)
	query, args, _ := sqlx.In( /* language=sql */ `SELECT * FROM env_vars WHERE container_id IN (?)`, containerIDs)
	err = sqlx.SelectContext(ctx, db.extLog, &envs, db.extLog.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
	default:
		err = models.WrapDBError(err)
		return
	}

	for _, env := range envs {
		containerEnvs := envMap[env.ContainerID]
		containerEnvs = append(containerEnvs, env)
		envMap[env.ContainerID] = containerEnvs
	}

	return
}

func (db *pgDB) getDeploymentsContainers(ctx context.Context,
	deplIDs []string) (contIDs []string, contMap map[string][]rstypes.Container, err error) {
	db.log.Debugf("get deployments containers %v", deplIDs)

	contIDs = make([]string, 0)
	contMap = make(map[string][]rstypes.Container)

	conts := make([]rstypes.Container, 0)
	query, args, _ := sqlx.In( /* language=sql */ `SELECT * FROM containers WHERE depl_id IN (?)`, deplIDs)
	err = sqlx.SelectContext(ctx, db.extLog, &conts, db.extLog.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
	default:
		err = models.WrapDBError(err)
		return
	}

	for _, cont := range conts {
		contIDs = append(contIDs, cont.ID)
		tmp := contMap[cont.DeployID]
		tmp = append(tmp, cont)
		contMap[cont.DeployID] = tmp
	}

	return
}

func (db *pgDB) getRawDeployments(ctx context.Context,
	userID, nsLabel string) (deplIDs []string, deployments []rstypes.Deployment, err error) {
	params := map[string]interface{}{
		"user_id":  userID,
		"ns_label": nsLabel,
	}
	db.log.WithFields(params).Debug("get raw deployments")

	deplIDs = make([]string, 0)
	deployments = make([]rstypes.Deployment, 0)

	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT d.* 
		FROM deployments d
		JOIN namespaces ns ON d.ns_id = ns.id
		JOIN permissions p ON ns.id = p.resource_id AND p.kind = 'namespace'
		WHERE p.resource_label = :ns_label AND p.user_id = :user_id`,
		params)
	err = sqlx.SelectContext(ctx, db.extLog, &deployments, db.extLog.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
	default:
		err = models.WrapDBError(err)
		return
	}

	for _, deploy := range deployments {
		deplIDs = append(deplIDs, deploy.ID)
	}

	return
}

func convertEnv(envs []rstypes.EnvironmentVariable) (ret []kubtypes.Env) {
	for _, envVar := range envs {
		ret = append(ret, kubtypes.Env{
			Name:  envVar.Name,
			Value: envVar.Value,
		})
	}
	return
}

func convertVols(vols []volumeMountWithName) (ret []kubtypes.Volume) {
	for _, volume := range vols {
		var volumeResp kubtypes.Volume
		volumeResp.Name = volume.Name
		volumeResp.MountPath = volume.MountPath
		if volume.SubPath.Valid {
			volumeResp.SubPath = &volume.SubPath.String
		}
		ret = append(ret, volumeResp)
	}
	return
}

func (db *pgDB) GetDeployments(ctx context.Context, userID, nsLabel string) (ret []kubtypes.Deployment, err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Debug("get deployments")

	deplIDs, deployments, err := db.getRawDeployments(ctx, userID, nsLabel)
	if err != nil {
		return
	}

	containerIDs, containerMap, err := db.getDeploymentsContainers(ctx, deplIDs)
	if err != nil {
		return
	}

	containerEnv, err := db.getContainersEnvironments(ctx, containerIDs)
	if err != nil {
		return
	}

	containerVols, err := db.getContainersVolumes(ctx, containerIDs)
	if err != nil {
		return
	}

	for _, deploy := range deployments {
		var deployResp kubtypes.Deployment
		deployResp.Name = deploy.Name
		deployResp.Replicas = deploy.Replicas
		for _, container := range containerMap[deploy.ID] {
			var containerResp kubtypes.Container
			containerResp.Name = container.Name
			containerResp.Image = container.Image
			// TODO: add resources description when model will be updated

			env := convertEnv(containerEnv[container.ID])
			containerResp.Env = &env

			vols := convertVols(containerVols[container.ID])
			containerResp.Volume = &vols
		}

		ret = append(ret, deployResp)
	}

	return
}

func (db *pgDB) getDeploymentContainers(ctx context.Context,
	deploy rstypes.Deployment) (ret []rstypes.Container, ids []string, err error) {
	db.log.WithField("deploy_id", deploy.ID).Debug("get deployment containers")

	query, args, _ := sqlx.Named( /* language=sql */ `SELECT * FROM containers WHERE depl_id = :id`, deploy)
	err = sqlx.GetContext(ctx, db.extLog, &ret, db.extLog.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
	default:
		err = models.WrapDBError(err)
		return
	}
	var containerIDs []string
	for _, v := range ret {
		containerIDs = append(containerIDs, v.ID)
	}

	return
}

func (db *pgDB) GetDeploymentByLabel(ctx context.Context, userID, nsLabel, deplLabel string) (ret kubtypes.Deployment, err error) {
	params := map[string]interface{}{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"deploy_label": deplLabel,
	}
	db.log.WithFields(params).Debug("get deployment by label")

	var rawDeploy rstypes.Deployment
	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT *
		FROM deployments d
		JOIN permissions p ON p.resource_id = d.ns_id AND p.kind = 'namespace'
		WHERE d.name := :deploy_label AND 
				p.user_id = :user_id AND 
				p.resource_label = :ns_label`,
		params)
	err = sqlx.GetContext(ctx, db.extLog, &rawDeploy, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = models.ErrLabeledResourceNotExists
		return
	default:
		err = models.WrapDBError(err)
		return
	}
	ret.Name = rawDeploy.Name
	ret.Replicas = rawDeploy.Replicas
	// TODO: add resources description when model will be updated

	rawContainers, containerIDs, err := db.getDeploymentContainers(ctx, rawDeploy)
	if err != nil {
		return
	}

	containerVols, err := db.getContainersVolumes(ctx, containerIDs)
	if err != nil {
		return
	}

	containerEnv, err := db.getContainersEnvironments(ctx, containerIDs)
	if err != nil {
		return
	}

	for _, container := range rawContainers {
		var containerResp kubtypes.Container
		containerResp.Name = container.Name
		containerResp.Image = container.Image

		env := convertEnv(containerEnv[container.ID])
		containerResp.Env = &env

		vols := convertVols(containerVols[container.ID])
		containerResp.Volume = &vols

		ret.Containers = append(ret.Containers, containerResp)
	}

	return
}
