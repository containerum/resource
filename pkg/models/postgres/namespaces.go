package postgres

import (
	"context"

	"database/sql"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/pkg/models"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/containerum/kube-client/pkg/cherry/resource-service"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type NamespacePG struct {
	models.RelationalDB
	log *cherrylog.LogrusAdapter
}

func NewNamespacePG(db models.RelationalDB) models.NamespaceDB {
	return &NamespacePG{
		RelationalDB: db,
		log:          cherrylog.NewLogrusAdapter(logrus.WithField("component", "namespace_pg")),
	}
}

func (db *NamespacePG) CreateNamespace(ctx context.Context, userID, label string, namespace *rstypes.Namespace) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debugf("creating namespace %#v", namespace)

	_, err = NewNamespacePG(db.RelationalDB).GetNamespaceID(ctx, userID, label)
	if err == nil {
		err = rserrors.ErrResourceAlreadyExists().AddDetailF("namespace %s already exists", label)
		return
	}
	if err != nil && !cherry.Equals(err, rserrors.ErrResourceNotExists()) {
		return
	}

	namespace.OwnerUserID = userID
	query, args, _ := sqlx.Named( /* language=sql */
		`INSERT INTO namespaces
		(
			tariff_id,
			ram,
			cpu,
			max_ext_services,
			max_int_services,
			max_traffic,
			owner_user_id
		)
		VALUES (:tariff_id, :ram, :cpu, :max_ext_services, :max_int_services, :max_traffic, :owner_user_id)
		RETURNING *`, namespace)
	err = sqlx.GetContext(ctx, db, namespace, db.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return err
	}

	_, err = sqlx.NamedExecContext(ctx, db, /* language=sql */
		`INSERT INTO permissions
		(
			kind,
			resource_id,
			resource_label,
			owner_user_id,
			user_id
		)
		VALUES ('namespace', :resource_id, :resource_label, :user_id, :user_id)`,
		rstypes.PermissionRecord{
			ResourceID:    &namespace.ID,
			ResourceLabel: label,
			UserID:        userID})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	return
}

func (db *NamespacePG) getNamespacesRaw(ctx context.Context,
	page, perPage int, filters *models.NamespaceFilterParams) (nsIDs []string, nsMap map[string]rstypes.NamespaceWithVolumes, err error) {
	db.log.WithFields(logrus.Fields{
		"page":     page,
		"per_page": perPage,
	}).Debugf("get raw namespaces (filters %#v)", filters)

	params := struct {
		Limit  int `db:"limit"`
		Offset int `db:"offset"`
		*models.NamespaceFilterParams
	}{
		Limit:                 perPage,
		Offset:                (page - 1) * perPage,
		NamespaceFilterParams: filters,
	}

	namespaces := make([]rstypes.NamespaceWithPermission, 0)
	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT ns.*, 
			p.id AS perm_id,
			p.kind,
			p.resource_id,
			p.resource_label,
			p.owner_user_id,
			p.create_time,
			p.user_id,
			p.access_level,
			p.limited,
			p.access_level_change_time,
			p.new_access_level
		FROM namespaces ns
		JOIN permissions p ON p.resource_id = ns.id AND p.kind = 'namespace'
		WHERE
			(NOT ns.deleted OR NOT :not_deleted) AND
			(ns.deleted OR NOT :deleted) AND
			(p.limited OR NOT :limited) AND
			(NOT p.limited OR NOT :not_limited) AND
			(p.user_id = p.owner_user_id OR NOT :owned)
		ORDER BY ns.create_time DESC
		LIMIT :limit
		OFFSET :offset`,
		params)
	err = sqlx.SelectContext(ctx, db, &namespaces, db.Rebind(query), args...)
	if err != nil {
		return
	}

	nsMap = make(map[string]rstypes.NamespaceWithVolumes)
	for _, v := range namespaces {
		nsIDs = append(nsIDs, v.Resource.ID)
		nsMap[v.Resource.ID] = rstypes.NamespaceWithVolumes{
			NamespaceWithPermission: v,
			Volume:                  []rstypes.VolumeWithPermission{},
		}
	}

	return
}

func (db *NamespacePG) getUserNamespacesRaw(ctx context.Context, userID string,
	filters *models.NamespaceFilterParams) (nsIDs []string, nsMap map[string]rstypes.NamespaceWithVolumes, err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
	}).Debugf("get raw user namespaces (filters %#v)", filters)

	params := struct {
		UserID string `db:"user_id"`
		*models.NamespaceFilterParams
	}{
		UserID:                userID,
		NamespaceFilterParams: filters,
	}

	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT ns.*, 
			p.id AS perm_id,
			p.kind,
			p.resource_id,
			p.resource_label,
			p.owner_user_id,
			p.create_time,
			p.user_id,
			p.access_level,
			p.limited,
			p.access_level_change_time,
			p.new_access_level
		FROM namespaces ns
		JOIN permissions p ON p.resource_id = ns.id AND p.kind = 'namespace'
		WHERE
			(p.user_id = :user_id OR -- return borrowed by default
			p.owner_user_id = :user_id) AND -- return owned by default
			(NOT ns.deleted OR NOT :not_deleted) AND
			(ns.deleted OR NOT :deleted) AND
			(p.limited OR NOT :limited) AND
			(NOT p.limited OR NOT :not_limited) AND
			(p.user_id = p.owner_user_id OR NOT :owned)
		ORDER BY ns.create_time DESC`,
		params)

	namespaces := make([]rstypes.NamespaceWithPermission, 0)
	err = sqlx.SelectContext(ctx, db, &namespaces, db.Rebind(query), args...)
	if err != nil {
		return
	}

	nsMap = make(map[string]rstypes.NamespaceWithVolumes)
	for _, v := range namespaces {
		nsIDs = append(nsIDs, v.Resource.ID)
		nsMap[v.Resource.ID] = rstypes.NamespaceWithVolumes{
			NamespaceWithPermission: v,
			Volume:                  []rstypes.VolumeWithPermission{},
		}
	}

	return
}

func (db *NamespacePG) addVolumesToNamespaces(ctx context.Context,
	nsIDs []string, nsMap map[string]rstypes.NamespaceWithVolumes) (err error) {
	db.log.Debugf("add volumes to namespaces %v", nsIDs)
	type volWithNsID struct {
		rstypes.VolumeWithPermission
		NsID string `db:"ns_id"`
	}
	if len(nsIDs) == 0 {
		return nil
	}
	volsWithNsID := make([]volWithNsID, 0)
	query, args, _ := sqlx.In( /* language=sql */
		`SELECT v.*, 
			p.id AS perm_id,
			p.kind,
			p.resource_id,
			p.resource_label,
			p.owner_user_id,
			p.create_time,
			p.user_id,
			p.access_level,
			p.limited,
			p.access_level_change_time,
			p.new_access_level,
			d.ns_id
		FROM volumes v
		JOIN volume_mounts vm ON v.id = vm.volume_id
		JOIN containers c ON vm.container_id = c.id
		JOIN deployments d ON c.depl_id = d.id
		JOIN permissions p ON p.resource_id = v.id
		WHERE d.ns_id IN (?)`, nsIDs)
	err = sqlx.SelectContext(ctx, db, &volsWithNsID, db.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
		err = nil
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	// fetch non-persistent volumes
	query, args, _ = sqlx.In( /* language=sql */
		`SELECT v.*, 
			p.id AS perm_id,
			p.kind,
			p.resource_id,
			p.resource_label,
			p.owner_user_id,
			p.create_time,
			p.user_id,
			p.access_level,
			p.limited,
			p.access_level_change_time,
			p.new_access_level,
			v.ns_id
		FROM volumes v
		JOIN permissions p ON p.resource_id = v.id
		WHERE v.ns_id IN (?)`, nsIDs)
	npvs := make([]volWithNsID, 0)
	err = sqlx.SelectContext(ctx, db, &volsWithNsID, db.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
		err = nil
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	volsWithNsID = append(volsWithNsID, npvs...)

	for _, v := range volsWithNsID {
		ns := nsMap[v.NsID]
		ns.Volume = append(ns.Volume, v.VolumeWithPermission)
		nsMap[v.NsID] = ns
	}

	return
}

func (db *NamespacePG) GetAllNamespaces(ctx context.Context,
	page, perPage int, filters *models.NamespaceFilterParams) (ret []rstypes.NamespaceWithVolumes, err error) {
	ret = make([]rstypes.NamespaceWithVolumes, 0)

	db.log.WithFields(logrus.Fields{
		"page":     page,
		"per_page": perPage,
	}).Debugf("get all namespaces (filters %#v)", filters)

	nsIDs, nsMap, err := db.getNamespacesRaw(ctx, page, perPage, filters)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	if len(nsIDs) == 0 {
		return
	}
	if err = db.addVolumesToNamespaces(ctx, nsIDs, nsMap); err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	for _, v := range nsMap {
		ret = append(ret, v)
	}

	return
}

func (db *NamespacePG) GetUserNamespaces(ctx context.Context, userID string,
	filters *models.NamespaceFilterParams) (ret []rstypes.NamespaceWithVolumes, err error) {
	ret = make([]rstypes.NamespaceWithVolumes, 0)

	db.log.WithField("user_id", userID).Debugf("get user namespaces")
	nsIDs, nsMap, err := db.getUserNamespacesRaw(ctx, userID, filters)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	if len(nsIDs) == 0 {
		return
	}
	if err = db.addVolumesToNamespaces(ctx, nsIDs, nsMap); err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	for _, v := range nsMap {
		ret = append(ret, v)
	}

	return
}

func (db *NamespacePG) GetUserNamespaceByLabel(ctx context.Context, userID, label string) (ret rstypes.NamespaceWithPermission, err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debug("get namespace by label")

	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT ns.*,
			p.id AS perm_id,
			p.kind,
			p.resource_id,
			p.resource_label,
			p.owner_user_id,
			p.create_time,
			p.user_id,
			p.access_level,
			p.limited,
			p.access_level_change_time,
			p.new_access_level
		FROM namespaces ns
		JOIN permissions p ON p.resource_id = ns.id AND p.kind = 'namespace'
		WHERE (p.user_id = :user_id OR p.owner_user_id = :user_id) AND p.resource_label = :resource_label`,
		rstypes.PermissionRecord{UserID: userID, ResourceLabel: label})
	err = sqlx.GetContext(ctx, db, &ret, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("namespace %s not exists", label).Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	ret.PermissionRecord.OwnerUserID = ret.Namespace.OwnerUserID

	return
}

func (db *NamespacePG) GetUserNamespaceWithVolumesByLabel(ctx context.Context, userID, label string) (ret rstypes.NamespaceWithVolumes, err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debug("get namespace with volumes by label")

	ret.Volume = make([]rstypes.VolumeWithPermission, 0)

	ret.NamespaceWithPermission, err = db.GetUserNamespaceByLabel(ctx, userID, label)
	if err != nil {
		return
	}

	// fetches persistent mounted volumes only
	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT v.*,
			p.id AS perm_id,
			p.kind,
			p.resource_id,
			p.resource_label,
			p.owner_user_id,
			p.create_time,
			p.user_id,
			p.access_level,
			p.limited,
			p.access_level_change_time,
			p.new_access_level
		FROM volumes v
		JOIN volume_mounts vm ON v.id = vm.volume_id
		JOIN permissions p ON p.resource_id = vm.volume_id AND p.kind = 'volume'
		JOIN containers c ON vm.container_id = c.id
		JOIN deployments d ON c.depl_id = d.id
		WHERE d.ns_id = :id`,
		ret.Resource)
	err = sqlx.SelectContext(ctx, db, &ret.Volume, db.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
		err = nil
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	var npv rstypes.VolumeWithPermission
	query, args, _ = sqlx.Named( /* language=sql */
		`SELECT v.*,
			p.id AS perm_id,
			p.kind,
			p.resource_id,
			p.resource_label,
			p.owner_user_id,
			p.create_time,
			p.user_id,
			p.access_level,
			p.limited,
			p.access_level_change_time,
			p.new_access_level
		FROM volumes v
		JOIN permissions p ON p.resource_id = v.id AND p.kind = 'volume'
		WHERE v.ns_id = :id`,
		ret.Resource)
	err = sqlx.GetContext(ctx, db, &npv, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = nil
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	ret.Volume = append(ret.Volume, npv)

	return
}

func (db *NamespacePG) GetNamespaceWithUserPermissions(ctx context.Context,
	userID, label string) (ret rstypes.NamespaceWithUserPermissions, err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debug("get user namespace with user permissions")

	ret.Users = make([]rstypes.PermissionRecord, 0)

	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT ns.*,
			p.id AS perm_id,
			p.kind,
			p.resource_id,
			p.resource_label,
			p.owner_user_id,
			p.create_time,
			p.user_id,
			p.access_level,
			p.limited,
			p.access_level_change_time,
			p.new_access_level
		FROM namespaces ns
		JOIN permissions p ON p.resource_id = ns.id AND p.kind = 'namespace'
		WHERE (p.user_id = :user_id) AND p.resource_label = :resource_label`,
		rstypes.PermissionRecord{UserID: userID, ResourceLabel: label})
	err = sqlx.GetContext(ctx, db, &ret.NamespaceWithPermission, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("namespace %s not exists", label).Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	//TODO: Fix unmarshalling
	ret.PermissionRecord.CreateTime = ret.Namespace.CreateTime
	ret.PermissionRecord.OwnerUserID = ret.Namespace.OwnerUserID

	query, args, _ = sqlx.Named( /* language=sql */
		`SELECT 
			p.id AS perm_id,
			p.kind,
			p.resource_id,
			p.resource_label,
			p.owner_user_id,
			p.create_time,
			p.user_id,
			p.access_level,
			p.limited,
			p.access_level_change_time,
			p.new_access_level
		FROM permissions p
		WHERE user_id != owner_user_id AND 
				resource_id = :id AND 
				kind = 'namespace'`,
		ret.Resource)
	err = sqlx.SelectContext(ctx, db, &ret.Users, db.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
		err = nil
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	return
}

func (db *NamespacePG) DeleteUserNamespaceByLabel(ctx context.Context, userID, label string) (namespace rstypes.Namespace, err error) {
	params := map[string]interface{}{
		"user_id":        userID,
		"resource_label": label,
	}
	db.log.WithFields(params).Debug("delete user namespace by label")

	query, args, _ := sqlx.Named( /* language=sql */
		`WITH user_ns AS (
			SELECT resource_id
			FROM permissions
			WHERE owner_user_id = user_id AND 
					user_id = :user_id AND 
					resource_label = :resource_label AND
					kind = 'namespace'
		)
		UPDATE namespaces
		SET deleted = TRUE, delete_time = now()
		WHERE id IN (SELECT resource_id FROM user_ns) AND NOT deleted
		RETURNING *`,
		params)
	err = sqlx.GetContext(ctx, db, &namespace, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("namespace %s not exists", label).Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	return
}

func (db *NamespacePG) DeleteAllUserNamespaces(ctx context.Context, userID string) (err error) {
	db.log.WithField("user_id", userID).Debug("delete user namespace by label")

	result, err := sqlx.NamedExecContext(ctx, db, /* language=sql */
		`WITH user_ns AS (
			SELECT resource_id
			FROM permissions
			WHERE owner_user_id = user_id AND 
					user_id = :user_id AND 
					kind = 'namespace'
		)
		UPDATE namespaces
		SET deleted = TRUE, delete_time = now()
		WHERE id IN (SELECT resource_id FROM user_ns) AND NOT deleted`,
		rstypes.PermissionRecord{UserID: userID})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = rserrors.ErrResourceNotExists().AddDetails("user don`t have namespaces").Log(err, db.log)
	}

	return
}

func (db *NamespacePG) RenameNamespace(ctx context.Context, userID, oldLabel, newLabel string) (err error) {
	params := map[string]interface{}{
		"user_id":            userID,
		"old_resource_label": oldLabel,
		"new_resource_label": newLabel,
	}
	db.log.WithFields(params).Debug("rename namespace")

	_, err = NewNamespacePG(db.RelationalDB).GetNamespaceID(ctx, userID, newLabel)
	if err == nil {
		err = rserrors.ErrResourceAlreadyExists().AddDetailF("namespace %s already exists", newLabel)
		return
	}
	if err != nil && !cherry.Equals(err, rserrors.ErrResourceNotExists()) {
		return
	}

	result, err := sqlx.NamedExecContext(ctx, db, /* language=sql */
		`UPDATE permissions
		SET resource_label = :new_resource_label
		WHERE owner_user_id = :user_id AND
				kind = 'namespace' AND
				resource_label = :old_resource_label`,
		params)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = rserrors.ErrResourceNotExists().AddDetailF("namespace %s not exists", oldLabel).Log(err, db.log)
	}

	return
}

func (db *NamespacePG) ResizeNamespace(ctx context.Context, namespace *rstypes.Namespace) (err error) {
	db.log.WithField("namespace_id", namespace.ID).Debugf("update namespace to %#v", namespace)

	query, args, _ := sqlx.Named( /* language=sql */
		`UPDATE namespaces
		SET
			tariff_id = :tariff_id,
			ram = :ram,
			cpu = :cpu,
			max_ext_services = :max_ext_services,
			max_int_services = :max_int_services,
			max_traffic = :max_traffic
		WHERE id = :id
		RETURNING *`,
		namespace)
	err = sqlx.GetContext(ctx, db, namespace, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("namespace %s not exists", namespace.ID).Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	return
}

func (db *NamespacePG) GetNamespaceID(ctx context.Context, userID, nsLabel string) (nsID string, err error) {
	queryFields := map[string]interface{}{
		"user_id": userID,
		"label":   nsLabel,
	}
	entry := db.log.WithFields(queryFields)
	entry.Debug("check if namespace exists")

	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT resource_id
		FROM permissions
		WHERE kind = 'namespace' AND 
			(owner_user_id = :user_id OR user_id = :user_id) AND
			resource_label = :label`,
		queryFields)
	err = sqlx.GetContext(ctx, db, &nsID, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("namespace %s not exists")
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	return
}

func (db *NamespacePG) GetNamespaceUsage(ctx context.Context, ns rstypes.Namespace) (usage models.NamespaceUsage, err error) {
	db.log.WithField("ns_id", ns.ID).Debugf("get namespace usage")
	query, args, _ := sqlx.Named( /* language=sql */
		`WITH cpu_ram AS (
			SELECT
				sum(c.ram)*d.replicas AS cpus,
				sum(c.cpu)*d.replicas AS rams
			FROM deployments d
			JOIN containers c on d.id = c.depl_id
			WHERE d.ns_id = :id
			GROUP BY d.replicas
		), ext_int AS (
			SELECT
				count(s.id) FILTER (WHERE s.type = 'external') AS extsvc,
				count(s.id) FILTER (WHERE s.type = 'internal') AS intsvc
			FROM deployments d
			JOIN services s ON s.deploy_id = d.id
			WHERE d.ns_id = :id
		)
		SELECT 
			sum(cpu_ram.cpus) AS cpu,
			sum(cpu_ram.rams) AS ram,
			ext_int.extsvc AS extservices,
			ext_int.intsvc AS intservices
		FROM cpu_ram, ext_int
		GROUP BY ext_int.intsvc, ext_int.extsvc
		UNION ALL 
        SELECT 0,0,0,0
        LIMIT 1;`, ns)
	err = sqlx.GetContext(ctx, db, &usage, db.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
	}
	return
}
