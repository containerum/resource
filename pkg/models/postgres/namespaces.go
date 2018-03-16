package postgres

import (
	"context"

	"database/sql"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
	"git.containerum.net/ch/resource-service/pkg/models"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

func (db *pgDB) getNamespaceID(ctx context.Context, userID, label string) (id string, err error) {
	queryFields := map[string]interface{}{
		"user_id": userID,
		"label":   label,
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
	err = sqlx.GetContext(ctx, db.extLog, &id, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		id = ""
		err = nil
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	entry.Debugf("found namespace %s", id)
	return
}

func (db *pgDB) CreateNamespace(ctx context.Context, userID, label string, namespace *rstypes.Namespace) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debugf("creating namespace %#v", namespace)

	var nsID string
	if nsID, err = db.getNamespaceID(ctx, userID, label); err != nil {
		return
	}
	if nsID != "" {
		err = rserrors.ErrResourceAlreadyExists().Log(err, db.log)
		return
	}

	query, args, _ := sqlx.Named( /* language=sql */
		`INSERT INTO namespaces
		(
			tariff_id,
			ram,
			cpu,
			max_ext_services,
			max_int_services,
			max_traffic
		)
		VALUES (:tariff_id, :ram, :cpu, :max_ext_services, :max_int_services, :max_traffic)
		RETURNING *`, namespace)
	err = sqlx.GetContext(ctx, db.extLog, namespace, db.extLog.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return err
	}

	_, err = sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
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

func (db *pgDB) getNamespacesRaw(ctx context.Context,
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
	err = sqlx.SelectContext(ctx, db.extLog, &namespaces, db.extLog.Rebind(query), args...)
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

func (db *pgDB) getUserNamespacesRaw(ctx context.Context, userID string,
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
	err = sqlx.SelectContext(ctx, db.extLog, &namespaces, db.extLog.Rebind(query), args...)
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

func (db *pgDB) GetAllNamespaces(ctx context.Context,
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

func (db *pgDB) GetUserNamespaces(ctx context.Context, userID string,
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

func (db *pgDB) GetUserNamespaceByLabel(ctx context.Context, userID, label string) (ret rstypes.NamespaceWithPermission, err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debug("get namespace with volumes by label")

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
	err = sqlx.GetContext(ctx, db.extLog, &ret, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	return
}

func (db *pgDB) GetUserNamespaceWithVolumesByLabel(ctx context.Context, userID, label string) (ret rstypes.NamespaceWithVolumes, err error) {
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
	err = sqlx.SelectContext(ctx, db.extLog, &ret.Volume, db.extLog.Rebind(query), args...)
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
	err = sqlx.GetContext(ctx, db.extLog, &npv, db.extLog.Rebind(query), args...)
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

func (db *pgDB) GetNamespaceWithUserPermissions(ctx context.Context,
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
		WHERE (p.user_id = :user_id OR p.owner_user_id = :user_id) AND p.resource_label = :resource_label`,
		rstypes.PermissionRecord{UserID: userID, ResourceLabel: label})
	err = sqlx.GetContext(ctx, db.extLog, &ret.NamespaceWithPermission, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

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
	err = sqlx.SelectContext(ctx, db.extLog, &ret.Users, db.extLog.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
		err = nil
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	return
}

func (db *pgDB) DeleteUserNamespaceByLabel(ctx context.Context, userID, label string) (namespace rstypes.Namespace, err error) {
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
		SET deleted = TRUE, delete_time = now() AT TIME ZONE 'UTC'
		WHERE id IN (SELECT resource_id FROM user_ns)
		RETURNING *`,
		params)
	err = sqlx.GetContext(ctx, db.extLog, &namespace, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	return
}

func (db *pgDB) DeleteAllUserNamespaces(ctx context.Context, userID string) (err error) {
	db.log.WithField("user_id", userID).Debug("delete user namespace by label")

	result, err := sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
		`WITH user_ns AS (
			SELECT resource_id
			FROM permissions
			WHERE owner_user_id = user_id AND 
					user_id = :user_id AND 
					kind = 'namespace'
		)
		UPDATE namespaces
		SET deleted = TRUE, delete_time = now() AT TIME ZONE 'UTC'
		WHERE id IN (SELECT resource_id FROM user_ns)`,
		rstypes.PermissionRecord{UserID: userID})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = rserrors.ErrResourceNotExists().Log(err, db.log)
	}

	return
}

func (db *pgDB) RenameNamespace(ctx context.Context, userID, oldLabel, newLabel string) (err error) {
	params := map[string]interface{}{
		"user_id":            userID,
		"old_resource_label": oldLabel,
		"new_resource_label": newLabel,
	}
	db.log.WithFields(params).Debug("rename namespace")

	var nsID string
	if nsID, err = db.getNamespaceID(ctx, userID, newLabel); err != nil {
		return
	}
	if nsID != "" {
		err = rserrors.ErrResourceAlreadyExists().Log(err, db.log)
		return
	}

	result, err := sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
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
		err = rserrors.ErrResourceNotExists().Log(err, db.log)
	}

	return
}

func (db *pgDB) ResizeNamespace(ctx context.Context, namespace *rstypes.Namespace) (err error) {
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
	err = sqlx.GetContext(ctx, db.extLog, namespace, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	return
}

func (db *pgDB) GetNamespaceID(ctx context.Context, userID, nsLabel string) (nsID string, err error) {
	nsID, err = db.getNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}
	if nsID == "" {
		err = rserrors.ErrResourceNotExists().AddDetailF("namespace %s not found for user %s", nsLabel, userID)
	}

	return
}
