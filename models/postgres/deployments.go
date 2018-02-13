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
		err = nil
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
		err = nil
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
		err = nil
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
	if len(envs) == 0 {
		ret = make([]kubtypes.Env, 0)
		return
	}
	for _, envVar := range envs {
		ret = append(ret, kubtypes.Env{
			Name:  envVar.Name,
			Value: envVar.Value,
		})
	}
	return
}

func convertVols(vols []volumeMountWithName) (ret []kubtypes.Volume) {
	if len(vols) == 0 {
		ret = make([]kubtypes.Volume, 0)
		return
	}
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
		deployResp.Containers = make([]kubtypes.Container, 0)
		for _, container := range containerMap[deploy.ID] {
			var containerResp kubtypes.Container
			containerResp.Name = container.Name
			containerResp.Image = container.Image
			// TODO: add resources description when model will be updated

			env := convertEnv(containerEnv[container.ID])
			containerResp.Env = &env

			vols := convertVols(containerVols[container.ID])
			containerResp.Volume = &vols

			deployResp.Containers = append(deployResp.Containers, containerResp)
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

func (db *pgDB) getDeployID(ctx context.Context, nsID, deplLabel string) (id string, err error) {
	params := map[string]interface{}{
		"ns_id":        nsID,
		"deploy_label": deplLabel,
	}
	db.log.WithFields(params).Debug("get deploy id")

	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT id
		FROM deployments
		WHERE ns_id = :ns_id AND name = :deploy_label`,
		params)
	err = sqlx.GetContext(ctx, db.extLog, &id, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = nil
		id = ""
	default:
		err = models.WrapDBError(err)
	}

	return
}

func (db *pgDB) createRawDeployment(ctx context.Context, nsID string,
	deployment kubtypes.Deployment) (id string, firstInNamespace bool, err error) {
	db.log.WithField("ns_id", nsID).Debugf("create raw deployment %#v", deployment)

	query, args, _ := sqlx.Named( /* language=sql */
		`WITH ns_deploys AS (
			SELECT * FROM deployments WHERE ns_id = :ns_id
		)
		INSERT INTO deployments
		(ns_id, name, replicas)
		VALUES (:ns_id, :name, :replicas)
		RETURNING id, NOT EXISTS(SELECT * from ns_deploys)`,
		rstypes.Deployment{
			NamespaceID: nsID,
			Name:        deployment.Name,
			Replicas:    deployment.Replicas,
		})
	err = db.extLog.QueryRowxContext(ctx, db.extLog.Rebind(query), args...).Scan(&id, &firstInNamespace)
	if err != nil {
		err = models.WrapDBError(err)
	}

	return
}

func (db *pgDB) createDeploymentContainers(ctx context.Context, deplID string,
	containers []kubtypes.Container) (contMap map[string]kubtypes.Container, err error) {
	db.log.WithField("deploy_id", deplID).Debugf("create deployment containers %#v", containers)

	stmt, err := db.preparer.PrepareNamed( /* language=sql */
		`INSERT INTO containers
		(depl_id, name, image, cpu, ram)
		VALUES (:depl_id, :name, :image, :cpu, :ram)
		RETURNING id`)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	defer stmt.Close()

	contMap = make(map[string]kubtypes.Container)
	for _, container := range containers {
		var containerID string
		err = stmt.GetContext(ctx, &containerID, rstypes.Container{
			DeployID: deplID,
			Name:     container.Name,
			Image:    container.Image,
			CPU:      1, // FIXME
			RAM:      1, // FIXME
		})
		if err != nil {
			err = models.WrapDBError(err)
			return
		}
		contMap[containerID] = container
	}

	return
}

func (db *pgDB) createContainersEnvs(ctx context.Context, contMap map[string]kubtypes.Container) (err error) {
	db.log.Debugf("create containers environments %#v", contMap)

	stmt, err := db.preparer.PrepareNamed( /* language=sql */
		`INSERT INTO env_vars
		(container_id, name, value)
		VALUES (:container_id, :name, :value)`)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	defer stmt.Close()

	for id, container := range contMap {
		if container.Env == nil {
			continue
		}
		for _, env := range *container.Env {
			_, err = stmt.ExecContext(ctx, rstypes.EnvironmentVariable{
				ContainerID: id,
				Name:        env.Name,
				Value:       env.Value,
			})
			if err != nil {
				err = models.WrapDBError(err)
				return
			}
		}
	}

	return
}

func (db *pgDB) createContainersVolumes(ctx context.Context, userID string, contMap map[string]kubtypes.Container) (err error) {
	params := map[string]interface{}{"user_id": userID}
	db.log.WithFields(params).Debugf("create containers volumes %#v", contMap)

	stmt, err := db.preparer.PrepareNamed( /* language=sql */
		`WITH vol_id_name AS (
			SELECT resource_label, resource_id
			FROM permissions
			WHERE kind = 'volume' AND user_id = :user_id
		)
		INSERT INTO volume_mounts
		(container_id, volume_id, mount_path, sub_path)
		VALUES (
			:container_id, 
			(SELECT resource_id FROM vol_id_name WHERE resource_label = :vol_name), 
			:mount_path, 
			:sub_path
		)`)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	defer stmt.Close()

	for id, container := range contMap {
		params["container_id"] = id
		if container.Volume == nil {
			continue
		}
		for _, v := range *container.Volume {
			params["vol_name"] = v.Name
			params["mount_path"] = v.MountPath
			params["sub_path"] = v.SubPath
			_, err = stmt.ExecContext(ctx, params)
			if err != nil {
				err = models.WrapDBError(err)
				return
			}
		}
	}

	return
}

func (db *pgDB) CreateDeployment(ctx context.Context, userID, nsLabel string,
	deployment kubtypes.Deployment) (firstInNamespace bool, err error) {
	params := map[string]interface{}{
		"user_id":  userID,
		"ns_label": nsLabel,
	}
	db.log.WithFields(params).Debugf("create deployment %#v", deployment)

	nsID, err := db.getNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}
	if nsID == "" {
		err = models.ErrLabeledResourceNotExists
		return
	}

	deplID, err := db.getDeployID(ctx, nsID, deployment.Name)
	if err != nil {
		return
	}
	if deplID != "" {
		err = models.ErrLabeledResourceExists
		return
	}

	deplID, firstInNamespace, err = db.createRawDeployment(ctx, nsID, deployment)
	if err != nil {
		return
	}

	contMap, err := db.createDeploymentContainers(ctx, deplID, deployment.Containers)
	if err != nil {
		return
	}

	if err = db.createContainersEnvs(ctx, contMap); err != nil {
		return
	}

	if err = db.createContainersVolumes(ctx, userID, contMap); err != nil {
		return
	}

	return
}

func (db *pgDB) DeleteDeployment(ctx context.Context, userID, nsLabel, deplLabel string) (lastInNamespace bool, err error) {
	params := map[string]interface{}{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"deploy_label": deplLabel,
	}
	db.log.WithFields(params).Debug("delete deployment")

	nsID, err := db.getNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}
	if nsID == "" {
		err = models.ErrLabeledResourceNotExists
		return
	}

	result, err := sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
		`UPDATE deployments
		SET deleted = TRUE, delete_time = now() AT TIME ZONE 'UTC'
		WHERE ns_id = :ns_id AND name = :name`,
		rstypes.Deployment{NamespaceID: nsID, Name: deplLabel})
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	if count, _ := result.RowsAffected(); count == 0 {
		err = models.ErrLabeledResourceNotExists
		return
	}

	var activeDeployCount int
	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT count(*) FROM deployments WHERE ns_id = :ns_id AND NOT deleted`,
		rstypes.Deployment{NamespaceID: nsID})
	err = sqlx.GetContext(ctx, db.extLog, &activeDeployCount, db.extLog.Rebind(query), args...)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}

	lastInNamespace = activeDeployCount <= 0
	return
}

func (db *pgDB) ReplaceDeployment(ctx context.Context, userID, nsLabel, deplLabel string, deploy kubtypes.Deployment) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"deploy_label": deplLabel,
	}).Debugf("replacing deployment with %#v", deploy)

	nsID, err := db.getNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}
	if nsID == "" {
		err = models.ErrLabeledResourceNotExists
		return
	}

	result, err := sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
		`DELETE FROM deployments
		WHERE ns_id = :ns_id AND name = :name`,
		rstypes.Deployment{NamespaceID: nsID, Name: deplLabel})
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	if count, _ := result.RowsAffected(); count == 0 {
		err = models.ErrLabeledResourceNotExists
		return
	}

	_, err = db.CreateDeployment(ctx, userID, nsLabel, deploy)
	return
}

func (db *pgDB) SetDeploymentReplicas(ctx context.Context, userID, nsLabel, deplLabel string, replicas int) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"deploy_label": deplLabel,
		"replicas":     replicas,
	}).Debug("set deployment replicas")

	nsID, err := db.getNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}
	if nsID == "" {
		err = models.ErrLabeledResourceNotExists
		return
	}

	result, err := sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
		`UPDATE deployments
		SET replicas = :replicas
		WHERE ns_id = :ns_id AND name = :name`,
		rstypes.Deployment{NamespaceID: nsID, Replicas: replicas, Name: deplLabel})
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	if count, _ := result.RowsAffected(); count == 0 {
		err = models.ErrLabeledResourceNotExists
		return
	}

	return
}

func (db *pgDB) SetContainerImage(ctx context.Context, userID, nsLabel, deplLabel string,
	req rstypes.SetContainerImageRequest) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"deploy_label": deplLabel,
	}).Debugf("set container image %#v", req)

	nsID, err := db.getNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}
	if nsID == "" {
		err = models.ErrLabeledResourceNotExists
		return
	}

	deplID, err := db.getDeployID(ctx, nsID, deplLabel)
	if err != nil {
		return
	}
	if deplID == "" {
		err = models.ErrLabeledResourceNotExists
		return
	}

	result, err := sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
		`UPDATE containers
		SET image = :image
		WHERE depl_id = :depl_id AND name = :name`,
		rstypes.Container{DeployID: deplID, Name: req.ContainerName, Image: req.Image})
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	if count, _ := result.RowsAffected(); count == 0 {
		err = models.ErrLabeledResourceNotExists
		return
	}

	return
}
