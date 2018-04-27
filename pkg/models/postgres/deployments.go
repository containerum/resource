package postgres

import (
	"context"

	"database/sql"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/pkg/models"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/containerum/kube-client/pkg/cherry/resource-service"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type DeployPG struct {
	models.RelationalDB
	log *cherrylog.LogrusAdapter
}

func NewDeployPG(db models.RelationalDB) models.DeployDB {
	return &DeployPG{
		RelationalDB: db,
		log:          cherrylog.NewLogrusAdapter(logrus.WithField("component", "deploy_pg")),
	}
}

type volumeMountWithName struct {
	Name string `db:"resource_label"`
	rstypes.VolumeMount
}

func (db *DeployPG) getContainersVolumes(ctx context.Context,
	containerIDs []string) (volMap map[string][]volumeMountWithName, err error) {
	db.log.Debugf("get containers volumes %v", containerIDs)

	volMap = make(map[string][]volumeMountWithName)

	if len(containerIDs) == 0 {
		return volMap, nil
	}

	vols := make([]volumeMountWithName, 0)
	query, args, _ := sqlx.In( /* language=sql */
		`SELECT p.resource_label, vm.* 
				FROM volume_mounts vm
				JOIN permissions p ON vm.volume_id = p.resource_id AND p.kind = 'volume'
				WHERE vm.container_id IN (?)`,
		containerIDs)
	err = sqlx.SelectContext(ctx, db, &vols, db.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
		err = nil
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	for _, vol := range vols {
		tmp := volMap[vol.ContainerID]
		tmp = append(tmp, vol)
		volMap[vol.ContainerID] = tmp
	}

	return
}

func (db *DeployPG) getContainersEnvironments(ctx context.Context,
	containerIDs []string) (envMap map[string][]rstypes.EnvironmentVariable, err error) {
	db.log.Debugf("get containers envs %v", containerIDs)

	envMap = make(map[string][]rstypes.EnvironmentVariable)

	if len(containerIDs) == 0 {
		return envMap, nil
	}

	envs := make([]rstypes.EnvironmentVariable, 0)
	query, args, _ := sqlx.In( /* language=sql */ `SELECT * FROM env_vars WHERE container_id IN (?)`, containerIDs)
	err = sqlx.SelectContext(ctx, db, &envs, db.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
		err = nil
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	for _, env := range envs {
		containerEnvs := envMap[env.ContainerID]
		containerEnvs = append(containerEnvs, env)
		envMap[env.ContainerID] = containerEnvs
	}

	return
}

func (db *DeployPG) getDeploymentsContainers(ctx context.Context,
	deplIDs []string) (contIDs []string, contMap map[string][]rstypes.Container, err error) {
	db.log.Debugf("get deployments containers %v", deplIDs)

	contIDs = make([]string, 0)
	contMap = make(map[string][]rstypes.Container)

	if len(deplIDs) == 0 {
		return contIDs, contMap, nil
	}

	conts := make([]rstypes.Container, 0)
	query, args, _ := sqlx.In( /* language=sql */ `SELECT * FROM containers WHERE depl_id IN (?)`, deplIDs)
	err = sqlx.SelectContext(ctx, db, &conts, db.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
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

func (db *DeployPG) getRawDeployments(ctx context.Context,
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
		JOIN namespaces ns ON d.ns_id = ns.id AND NOT ns.deleted
		JOIN permissions p ON ns.id = p.resource_id AND p.kind = 'namespace'
		WHERE p.resource_label = :ns_label AND p.user_id = :user_id AND NOT d.deleted`,
		params)
	err = sqlx.SelectContext(ctx, db, &deployments, db.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
		err = nil
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
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

func convertVols(vols []volumeMountWithName) (ret []kubtypes.ContainerVolume) {
	if len(vols) == 0 {
		ret = make([]kubtypes.ContainerVolume, 0)
		return
	}
	for _, volume := range vols {
		var volumeResp kubtypes.ContainerVolume
		volumeResp.Name = volume.Name
		volumeResp.MountPath = volume.MountPath
		if volume.SubPath != nil {
			volumeResp.SubPath = volume.SubPath
		}
		ret = append(ret, volumeResp)
	}
	return
}

func (db *DeployPG) GetDeployments(ctx context.Context, userID, nsLabel string) (ret []kubtypes.Deployment, err error) {
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
			containerResp.Limits.CPU = uint(container.CPU)
			containerResp.Limits.Memory = uint(container.RAM)

			env := convertEnv(containerEnv[container.ID])
			containerResp.Env = env

			vols := convertVols(containerVols[container.ID])
			containerResp.VolumeMounts = vols

			deployResp.Containers = append(deployResp.Containers, containerResp)
		}

		ret = append(ret, deployResp)
	}

	return
}

func (db *DeployPG) getDeploymentContainers(ctx context.Context,
	deploy rstypes.Deployment) (ret []rstypes.Container, ids []string, err error) {
	db.log.WithField("deploy_id", deploy.ID).Debug("get deployment containers")

	query, args, _ := sqlx.Named( /* language=sql */ `SELECT * FROM containers WHERE depl_id = :id`, deploy)
	err = sqlx.SelectContext(ctx, db, &ret, db.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	var containerIDs []string
	for _, v := range ret {
		containerIDs = append(containerIDs, v.ID)
	}

	return
}

func (db *DeployPG) GetDeploymentByLabel(ctx context.Context, userID, nsLabel, deplName string) (ret kubtypes.Deployment, err error) {
	params := map[string]interface{}{
		"user_id":     userID,
		"ns_label":    nsLabel,
		"deploy_name": deplName,
	}
	db.log.WithFields(params).Debug("get deployment by label")

	var rawDeploy rstypes.Deployment
	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT d.*
		FROM deployments d
		JOIN permissions p ON p.resource_id = d.ns_id AND p.kind = 'namespace'
		WHERE NOT d.deleted AND (d.name, p.user_id, p.resource_label) = (:deploy_name, :user_id, :ns_label)`,
		params)
	err = sqlx.GetContext(ctx, db, &rawDeploy, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("deployment %s not found", deplName).Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	ret.Name = rawDeploy.Name
	ret.Replicas = rawDeploy.Replicas

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
		containerResp.Limits.CPU = uint(container.CPU)
		containerResp.Limits.Memory = uint(container.RAM)
		env := convertEnv(containerEnv[container.ID])
		containerResp.Env = env

		vols := convertVols(containerVols[container.ID])
		containerResp.VolumeMounts = vols

		ret.Containers = append(ret.Containers, containerResp)
	}

	return
}

func (db *DeployPG) GetDeployID(ctx context.Context, nsID, deplName string) (id string, err error) {
	params := map[string]interface{}{
		"ns_id":       nsID,
		"deploy_name": deplName,
	}
	db.log.WithFields(params).Debug("get deploy id")

	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT id
		FROM deployments
		WHERE NOT deleted AND (ns_id, name) = (:ns_id, :deploy_name)`,
		params)
	err = sqlx.GetContext(ctx, db, &id, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("deployment %s not exists", deplName)
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}

func (db *DeployPG) createRawDeployment(ctx context.Context, nsID string,
	deployment kubtypes.Deployment) (id string, firstInNamespace bool, err error) {
	db.log.WithField("ns_id", nsID).Debugf("create raw deployment %#v", deployment)

	query, args, _ := sqlx.Named( /* language=sql */
		`WITH ns_deploys AS (
			SELECT * FROM deployments WHERE ns_id = :ns_id AND NOT deleted
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
	err = db.QueryRowxContext(ctx, db.Rebind(query), args...).Scan(&id, &firstInNamespace)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}

func (db *DeployPG) createDeploymentContainers(ctx context.Context, deplID string,
	containers []kubtypes.Container) (contMap map[string]kubtypes.Container, err error) {
	db.log.WithField("deploy_id", deplID).Debugf("create deployment containers %#v", containers)

	stmt, err := db.PrepareNamed( /* language=sql */
		`INSERT INTO containers
		(depl_id, name, image, cpu, ram)
		VALUES (:depl_id, :name, :image, :cpu, :ram)
		RETURNING id`)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
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
			CPU:      int(container.Limits.CPU),    // our limits fits into "int", stored as mCPU
			RAM:      int(container.Limits.Memory), // stored as megabytes
		})
		if err != nil {
			err = rserrors.ErrDatabase().Log(err, db.log)
			return
		}
		contMap[containerID] = container
	}

	return
}

func (db *DeployPG) createContainersEnvs(ctx context.Context, contMap map[string]kubtypes.Container) (err error) {
	db.log.Debugf("create containers environments %#v", contMap)

	stmt, err := db.PrepareNamed( /* language=sql */
		`INSERT INTO env_vars
		(container_id, "name", "value")
		VALUES (:container_id, :name, :value)`)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	defer stmt.Close()

	for id, container := range contMap {
		for _, env := range container.Env {
			_, err = stmt.ExecContext(ctx, rstypes.EnvironmentVariable{
				ContainerID: id,
				Name:        env.Name,
				Value:       env.Value,
			})
			if err != nil {
				err = rserrors.ErrDatabase().Log(err, db.log)
				return
			}
		}
	}

	return
}

func (db *DeployPG) checkVolumesExists(ctx context.Context, userID string, contMap map[string]kubtypes.Container) (err error) {
	db.log.WithField("user_id", userID).Debugf("check volume exists for user, containers %#v", contMap)

	volExistMap := make(map[string]bool)
	for _, c := range contMap {
		for _, v := range c.VolumeMounts {
			volExistMap[v.Name] = false
		}
	}

	var existingVols []string
	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT resource_label FROM permissions WHERE (kind, user_id) = ('volume', :user_id)`,
		map[string]interface{}{"user_id": userID})
	err = sqlx.SelectContext(ctx, db, &existingVols, db.Rebind(query), args...)
	switch err {
	case err, sql.ErrNoRows:
		err = nil
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	for _, v := range existingVols {
		volExistMap[v] = true
	}

	var nonExistingVolumes []string
	for vol, exist := range volExistMap {
		if !exist {
			nonExistingVolumes = append(nonExistingVolumes, vol)
		}
	}

	if len(nonExistingVolumes) > 0 {
		err = rserrors.ErrResourceNotExists().AddDetailF("volumes %#v are not exists", nonExistingVolumes)
	}

	return
}

func (db *DeployPG) createContainersVolumes(ctx context.Context, userID string, contMap map[string]kubtypes.Container) (err error) {
	params := map[string]interface{}{"user_id": userID}
	db.log.WithFields(params).Debugf("create containers volumes %#v", contMap)

	if err = db.checkVolumesExists(ctx, userID, contMap); err != nil {
		return
	}

	stmt, err := db.PrepareNamed( /* language=sql */
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
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	defer stmt.Close()

	for id, container := range contMap {
		params["container_id"] = id
		for _, v := range container.VolumeMounts {
			params["vol_name"] = v.Name
			params["mount_path"] = v.MountPath
			params["sub_path"] = v.SubPath
			_, err = stmt.ExecContext(ctx, params)
			if err != nil {
				err = rserrors.ErrDatabase().Log(err, db.log)
				return
			}
		}
	}

	return
}

func (db *DeployPG) CreateDeployment(ctx context.Context, userID, nsLabel string,
	deployment kubtypes.Deployment) (firstInNamespace bool, err error) {
	params := map[string]interface{}{
		"user_id":  userID,
		"ns_label": nsLabel,
	}
	db.log.WithFields(params).Debugf("create deployment %#v", deployment)

	nsID, err := NewNamespacePG(db.RelationalDB).GetNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}

	deplID, err := db.GetDeployID(ctx, nsID, deployment.Name)
	if err == nil {
		err = rserrors.ErrResourceAlreadyExists().AddDetailF("deployment %s already exists", deployment.Name).Log(err, db.log)
		return
	}
	if err != nil && !cherry.Equals(err, rserrors.ErrResourceNotExists()) {
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

func (db *DeployPG) DeleteDeployment(ctx context.Context, userID, nsLabel, deplName string) (lastInNamespace bool, err error) {
	params := map[string]interface{}{
		"user_id":     userID,
		"ns_label":    nsLabel,
		"deploy_name": deplName,
	}
	db.log.WithFields(params).Debug("delete deployment")

	nsID, err := NewNamespacePG(db.RelationalDB).GetNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}

	result, err := sqlx.NamedExecContext(ctx, db, /* language=sql */
		`UPDATE deployments
		SET deleted = TRUE, delete_time = now()
		WHERE (ns_id, "name") = (:ns_id, :name) AND NOT deleted`,
		rstypes.Deployment{NamespaceID: nsID, Name: deplName})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	if count, _ := result.RowsAffected(); count == 0 {
		err = rserrors.ErrResourceNotExists().AddDetailF("deployment %s not exists", deplName).Log(err, db.log)
		return
	}

	var activeDeployCount int
	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT count(*) FROM deployments WHERE ns_id = :ns_id AND NOT deleted`,
		rstypes.Deployment{NamespaceID: nsID})
	err = sqlx.GetContext(ctx, db, &activeDeployCount, db.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	lastInNamespace = activeDeployCount <= 0
	return
}

func (db *DeployPG) ReplaceDeployment(ctx context.Context, userID, nsLabel string, deploy kubtypes.Deployment) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Debugf("replacing deployment with %#v", deploy)

	nsID, err := NewNamespacePG(db.RelationalDB).GetNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}

	var deplID string

	// assuming cascade removal of containers, etc.
	query, args, _ := sqlx.Named( /* language=sql */
		`UPDATE deployments
		SET replicas = :replicas
		WHERE ns_id = :ns_id AND name = :name AND NOT deleted
		RETURNING id`,
		rstypes.Deployment{NamespaceID: nsID, Name: deploy.Name, Replicas: deploy.Replicas})
	err = sqlx.GetContext(ctx, db, &deplID, db.Rebind(query), args...)
	switch err {
	case nil:
		// pass
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("deployment %s not exists", deploy.Name).Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	_, err = sqlx.NamedExecContext(ctx, db, /* language=sql */
		`DELETE FROM containers WHERE depl_id = :id`, map[string]string{"id": deplID})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	contMap, err := db.createDeploymentContainers(ctx, deplID, deploy.Containers)
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

func (db *DeployPG) SetDeploymentReplicas(ctx context.Context, userID, nsLabel, deplName string, replicas int) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_label":    nsLabel,
		"deploy_name": deplName,
		"replicas":    replicas,
	}).Debug("set deployment replicas")

	nsID, err := NewNamespacePG(db.RelationalDB).GetNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}

	result, err := sqlx.NamedExecContext(ctx, db, /* language=sql */
		`UPDATE deployments
		SET replicas = :replicas
		WHERE ns_id = :ns_id AND name = :name AND NOT deleted`,
		rstypes.Deployment{NamespaceID: nsID, Replicas: replicas, Name: deplName})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	if count, _ := result.RowsAffected(); count == 0 {
		err = rserrors.ErrResourceNotExists().AddDetailF("deployment %s not exists", deplName).Log(err, db.log)
		return
	}

	return
}

func (db *DeployPG) SetContainerImage(ctx context.Context, userID, nsLabel, deplName string, req kubtypes.UpdateImage) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"ns_label":    nsLabel,
		"deploy_name": deplName,
	}).Debugf("set container image %#v", req)

	nsID, err := NewNamespacePG(db.RelationalDB).GetNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}

	deplID, err := db.GetDeployID(ctx, nsID, deplName)
	if err != nil {
		return
	}

	result, err := sqlx.NamedExecContext(ctx, db, /* language=sql */
		`UPDATE containers
		SET image = :image
		WHERE depl_id = :depl_id AND name = :name`,
		rstypes.Container{DeployID: deplID, Name: req.Container, Image: req.Image})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	if count, _ := result.RowsAffected(); count == 0 {
		err = rserrors.ErrResourceNotExists().AddDetailF("container %s not exists", req.Container).Log(err, db.log)
		return
	}

	return
}
