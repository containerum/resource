package postgres

import (
	"context"

	"database/sql"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

func (db *pgDB) isNamespaceExists(ctx context.Context, userID, label string) (exists bool, err error) {
	queryFields := map[string]interface{}{
		"user_id": userID,
		"label":   label,
	}
	entry := db.log.WithFields(queryFields)
	entry.Debug("check if namespace exists")

	var count int
	query, args, _ := sqlx.Named(`
		SELECT count(ns.*)
		FROM namespaces ns
		JOIN permissions p ON p.resource_id = ns.id AND p.resource_kind = 'namespace'
		WHERE p.user_id = :user_id AND p.resource_label = :label`, queryFields)
	err = sqlx.GetContext(ctx, db.extLog, &count, db.extLog.Rebind(query), args...)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}

	entry.Debugf("found %d namespaces", count)
	exists = count > 0
	return
}

func (db *pgDB) CreateNamespace(ctx context.Context, userID, label string, namespace *rstypes.Namespace) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debugf("creating namespace %#v", namespace)

	var exists bool
	if exists, err = db.isNamespaceExists(ctx, userID, label); err != nil {
		return
	}
	if exists {
		err = models.ErrLabeledResourceExists
		return
	}

	query, args, _ := sqlx.Named(`
		INSERT INTO namespaces
		(
			tariff_id,
			ram,
			cpu,
			max_ext_services,
			max_int_services,
			max_traffic,
		)
		VALUES (:tariff_id, :ram, :cpu, :max_ext_services, :max_int_services, :max_traffic)
		RETURNING *`, namespace)
	err = sqlx.GetContext(ctx, db.extLog, namespace, db.extLog.Rebind(query), args...)
	if err != nil {
		err = models.WrapDBError(err)
		return err
	}

	_, err = sqlx.NamedExecContext(ctx, db.extLog, `
		INSERT INTO permissions
		(
			kind,
			resource_id,
			resource_label
			owner_user_id,
			user_id
		)
		VALUES ('namespace', :resource_id, :resource_label, :user_id, :user_id)`,
		map[string]interface{}{
			"resource_id":    namespace.ID,
			"resource_label": label,
			"user_id":        userID,
		})
	if err != nil {
		err = models.WrapDBError(err)
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
		Limit:                 page,
		Offset:                (perPage - 1) * page,
		NamespaceFilterParams: filters,
	}

	namespaces := make([]rstypes.NamespaceWithPermission, 0)
	query, args, _ := sqlx.Named(`
		SELECT ns.*, p.*
		FROM namespaces ns
		JOIN permissions p ON p.resource_id = ns.id AND p.kind = 'Namespace'
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
	err = sqlx.SelectContext(ctx, db.extLog, namespaces, db.extLog.Rebind(query), args...)
	if err != nil {
		return
	}

	nsMap = make(map[string]rstypes.NamespaceWithVolumes)
	for _, v := range namespaces {
		nsIDs = append(nsIDs, v.ID)
		nsMap[v.ID] = rstypes.NamespaceWithVolumes{
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

	query, args, _ := sqlx.Named(`
		SELECT ns.*, p.*
		FROM namespaces ns
		JOIN permissions p ON p.resource_id = ns.id AND p.kind = 'namespace'
		WHERE
			p.user_id = :user_id AND
			(NOT ns.deleted OR NOT :not_deleted) AND
			(ns.deleted OR NOT :deleted) AND
			(p.limited OR NOT :limited) AND
			(NOT p.limited OR NOT :not_limited) AND
			(p.user_id = p.owner_user_id OR NOT :owned)
		ORDER BY ns.create_time DESC`,
		params)

	namespaces := make([]rstypes.NamespaceWithPermission, 0)
	err = sqlx.SelectContext(ctx, db.extLog, namespaces, db.extLog.Rebind(query), args...)
	if err != nil {
		return
	}

	nsMap = make(map[string]rstypes.NamespaceWithVolumes)
	for _, v := range namespaces {
		nsIDs = append(nsIDs, v.ID)
		nsMap[v.ID] = rstypes.NamespaceWithVolumes{
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
		err = models.WrapDBError(err)
		return
	}
	if len(nsIDs) == 0 {
		err = models.ErrResourceNotExists
		return
	}
	if err = db.addVolumesToNamespaces(ctx, nsIDs, nsMap); err != nil {
		err = models.WrapDBError(err)
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
		err = models.WrapDBError(err)
		return
	}
	if len(nsIDs) == 0 {
		err = models.ErrResourceNotExists
		return
	}
	if err = db.addVolumesToNamespaces(ctx, nsIDs, nsMap); err != nil {
		err = models.WrapDBError(err)
		return
	}

	for _, v := range nsMap {
		ret = append(ret, v)
	}

	return
}

func (db *pgDB) GetUserNamespaceByLabel(ctx context.Context, userID, label string) (ret rstypes.NamespaceWithVolumes, err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debug("get namespace by label")

	query, args, _ := sqlx.Named(`
		SELECT ns.*, p.*
		FROM namespaces ns
		JOIN permissions p ON p.resource_id = ns.id AND p.kind = 'namespace'
		WHERE p.user_id = :user_id AND p.resource_label = :resource_label`,
		map[string]interface{}{
			"user_id":        userID,
			"resource_label": label,
		})
	err = sqlx.GetContext(ctx, db.extLog, &ret.NamespaceWithPermission, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = models.ErrLabeledResourceNotExists
		return
	default:
		err = models.WrapDBError(err)
		return
	}

	query, args, _ = sqlx.Named(`
		SELECT v.*, p.*
		FROM namespace_volume nv
		JOIN volumes v ON v.id = nv.vol_id
		JOIN permissions p ON p.resource_id = nv.vol_id AND p.kind = 'volume'
		WHERE nv.ns_id = :ns_id`,
		map[string]interface{}{
			"ns_id": ret.ID,
		})
	err = sqlx.SelectContext(ctx, db.extLog, ret.Volume, db.extLog.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
	default:
		err = models.WrapDBError(err)
		return
	}

	return
}

func (db *pgDB) GetNamespaceWithUserPermissions(ctx context.Context,
	userID, label string) (ret rstypes.NamespaceWithUserPermissions, err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debug("get user namespace with user permissions")

	query, args, _ := sqlx.Named(`
		SELECT ns.*, p.*
		FROM namespaces ns
		JOIN permissions p ON p.resource_id = ns.id AND p.kind = 'namespace'
		WHERE p.user_id = :user_id AND p.resource_label = :resource_label`,
		map[string]interface{}{
			"user_id":        userID,
			"resource_label": label,
		})
	err = sqlx.GetContext(ctx, db.extLog, &ret.NamespaceWithPermission, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = models.ErrLabeledResourceNotExists
		return
	default:
		err = models.WrapDBError(err)
		return
	}

	query, args, _ = sqlx.Named(`
		SELECT * 
		FROM permissions 
		WHERE user_id != owner_user_id AND 
				resource_id = :resource_id AND 
				resource_kind = 'namespace'`,
		map[string]interface{}{
			"resource_id": ret.ID,
		})
	err = sqlx.SelectContext(ctx, db.extLog, ret.Users, db.extLog.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
	default:
		err = models.WrapDBError(err)
		return
	}

	return
}

func (db *pgDB) DeleteUserNamespaceByLabel(ctx context.Context, userID, label string) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debug("delete user namespace by label")

	result, err := sqlx.NamedExecContext(ctx, db.extLog, `
		WITH user_ns AS (
			SELECT resource_id
			FROM permissions
			WHERE owner_user_id = user_id AND 
					user_id = :user_id AND 
					resource_label = :resource_label AND
					resource_kind = 'namespace'
		)
		UPDATE namespaces
		SET deleted = TRUE
		WHERE id IN (SELECT * FROM user_ns)`,
		map[string]interface{}{
			"user_id":        userID,
			"resource_label": label,
		})
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrLabeledResourceNotExists
	}

	return
}

func (db *pgDB) DeleteAllUserNamespaces(ctx context.Context, userID string) (err error) {
	db.log.WithField("user_id", userID).Debug("delete user namespace by label")

	result, err := sqlx.NamedExecContext(ctx, db.extLog, `
		WITH user_ns AS (
			SELECT resource_id
			FROM permissions
			WHERE owner_user_id = user_id AND 
					user_id = :user_id AND 
					resource_kind = 'namespace'
		)
		UPDATE namespaces
		SET deleted = TRUE
		WHERE id IN (SELECT * FROM user_ns)`,
		map[string]interface{}{
			"user_id": userID,
		})
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrLabeledResourceNotExists
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

	var exists bool
	if exists, err = db.isNamespaceExists(ctx, userID, newLabel); err != nil {
		return
	}
	if exists {
		err = models.ErrLabeledResourceExists
		return
	}

	result, err := sqlx.NamedExecContext(ctx, db.extLog, `
		UPDATE permissions
		SET resource_label = :new_resource_label
		WHERE owner_user_id = :user_id AND
				resource_kind = 'namespace' AND
				resource_label = :old_resource_label`,
		params)
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrLabeledResourceNotExists
	}

	return
}

func (db *pgDB) ResizeNamespace(ctx context.Context, userID, label string, namespace *rstypes.Namespace) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debugf("update namespace to %#v", namespace)

	params := struct {
		UserID string `db:"user_id"`
		Label  string `db:"resource_label"`
		*rstypes.Namespace
	}{
		UserID:    userID,
		Label:     label,
		Namespace: namespace,
	}

	result, err := sqlx.NamedExecContext(ctx, db.extLog, `
		WITH user_ns AS (
			SELECT resource_id
			FROM permissions
			WHERE owner_user_id = user_id AND 
				user_id = $1 AND 
				resource_kind = 'namespace' AND
				resource_label = $2
		)
		UPDATE namespaces
		SET
			tariff_id = $3,
			ram = $4,
			cpu = $5,
			max_ext_services = $6,
			max_int_services = $7,
			max_traffic = $8
		WHERE id IN (SELECT * FROM user_ns)`,
		params)
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrLabeledResourceNotExists
	}

	return
}
