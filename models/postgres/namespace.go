package postgres

import (
	"context"

	"database/sql"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"github.com/sirupsen/logrus"
)

func (db *pgDB) isNamespaceExists(ctx context.Context, userID, label string) (exists bool, err error) {
	entry := db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	})
	entry.Debug("check if namespace exists")

	var count int
	err = db.qLog.QueryRowxContext(ctx, `
		SELECT count(ns.*)
		FROM namespaces ns
		JOIN permissions p ON p.resource_id = ns.id AND p.resource_kind = 'namespace'
		WHERE p.user_id = $1 AND p.resource_label = $2`,
		userID, label).Scan(&count)
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
	}).Infof("creating namespace %#v", namespace)

	var exists bool
	if exists, err = db.isNamespaceExists(ctx, userID, label); err != nil {
		return
	}
	if exists {
		err = models.ErrLabeledResourceExists
		return
	}

	err = db.qLog.QueryRowxContext(ctx, `
		INSERT INTO namespaces
		(
			tariff_id,
			ram,
			cpu,
			max_ext_services,
			max_int_services,
			max_traffic,
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING *`,
		namespace.TariffID,
		namespace.RAM,
		namespace.CPU,
		namespace.MaxExternalServices,
		namespace.MaxIntServices,
		namespace.MaxTraffic).StructScan(namespace)
	if err != nil {
		err = models.WrapDBError(err)
		return err
	}

	_, err = db.eLog.ExecContext(ctx, `
			INSERT INTO permissions
			(
				kind,
				resource_id,
				resource_label
				owner_user_id,
				user_id
			)
			VALUES ('namespace', $1, $2, $3, $3)`, namespace.ID, label, userID)
	if err != nil {
		err = models.WrapDBError(err)
		return err
	}

	return err
}

func (db *pgDB) getNamespacesRaw(ctx context.Context,
	page, perPage int, filters *models.NamespaceFilterParams) (nsIDs []string, nsMap map[string]rstypes.NamespaceWithVolumes, err error) {
	db.log.WithFields(logrus.Fields{
		"page":     page,
		"per_page": perPage,
	}).Debugf("get raw namespaces (filters %#v)", filters)
	rows, err := db.qLog.QueryxContext(ctx, `
			SELECT ns.*, p.*
			FROM namespaces ns
			JOIN permissions p ON p.resource_id = ns.id AND p.kind = 'Namespace'
			WHERE
				(NOT ns.deleted OR NOT $3) AND
				(ns.deleted OR NOT $4) AND
				(p.limited OR NOT $5) AND
				(NOT p.limited OR NOT $6) AND
				(p.user_id = p.owner_user_id OR NOT $7)
			ORDER BY ns.create_time DESC
			LIMIT $1
			OFFSET $2`,
		perPage,
		(page-1)*perPage,
		filters.NotDeleted,
		filters.Deleted,
		filters.Limited,
		filters.NotLimited,
		filters.Owners)
	if err != nil {
		return
	}
	defer rows.Close()

	nsMap = make(map[string]rstypes.NamespaceWithVolumes)
	for rows.Next() {
		var ns rstypes.NamespaceWithPermission
		if err = rows.StructScan(&ns); err != nil {
			return
		}
		nsIDs = append(nsIDs, ns.ID)
		nsMap[ns.ID] = rstypes.NamespaceWithVolumes{
			NamespaceWithPermission: ns,
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
	rows, err := db.qLog.QueryxContext(ctx, `
			SELECT ns.*, p.*
			FROM namespaces ns
			JOIN permissions p ON p.resource_id = ns.id AND p.kind = 'namespace'
			WHERE
				p.user_id = $1 AND
				(NOT ns.deleted OR NOT $2) AND
				(ns.deleted OR NOT $3) AND
				(p.limited OR NOT $4) AND
				(NOT p.limited OR NOT $5) AND
				(p.user_id = p.owner_user_id OR NOT $6)
			ORDER BY ns.create_time DESC`,
		userID,
		filters.NotDeleted,
		filters.Deleted,
		filters.Limited,
		filters.NotLimited,
		filters.Owners)
	if err != nil {
		return
	}
	defer rows.Close()

	nsMap = make(map[string]rstypes.NamespaceWithVolumes)
	for rows.Next() {
		var ns rstypes.NamespaceWithPermission
		if err = rows.StructScan(&ns); err != nil {
			return
		}
		nsIDs = append(nsIDs, ns.ID)
		nsMap[ns.ID] = rstypes.NamespaceWithVolumes{
			NamespaceWithPermission: ns,
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

	err = db.qLog.QueryRowxContext(ctx, `
		SELECT ns.*, p.*
		FROM namespaces ns
		JOIN permissions p ON p.resource_id = ns.id AND p.kind = 'namespace'
		WHERE p.user_id = $1 AND p.resource_label = $2`, userID, label).
		StructScan(&ret.NamespaceWithPermission)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = models.ErrLabeledResourceNotExists
		return
	default:
		err = models.WrapDBError(err)
		return
	}

	rows, err := db.qLog.QueryxContext(ctx, `
		SELECT v.*, p.*
		FROM namespace_volume nv
		JOIN volumes v ON v.id = nv.vol_id
		JOIN permissions p ON p.resource_id = nv.vol_id AND p.kind = 'volume'
		WHERE nv.ns_id = $1`, ret.ID)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	defer rows.Close()

	ret.Volume = make([]rstypes.VolumeWithPermission, 0)
	for rows.Next() {
		var volume rstypes.VolumeWithPermission
		if err = rows.StructScan(&volume); err != nil {
			err = models.WrapDBError(err)
			return
		}
		ret.Volume = append(ret.Volume, volume)
	}

	return
}

func (db *pgDB) GetNamespaceWithUserPermissions(ctx context.Context,
	userID, label string) (ret rstypes.NamespaceWithUserPermissions, err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debug("get user namespace with user permissions")

	err = db.qLog.QueryRowxContext(ctx, `
		SELECT ns.*, p.*
		FROM namespaces ns
		JOIN permissions p ON p.resource_id = ns.id AND p.kind = 'namespace'
		WHERE p.user_id = $1 AND p.resource_label = $2`, userID, label).
		StructScan(&ret.NamespaceWithPermission)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = models.ErrLabeledResourceNotExists
		return
	default:
		err = models.WrapDBError(err)
		return
	}

	rows, err := db.qLog.QueryxContext(ctx, `
		SELECT * 
		FROM permissions 
		WHERE user_id != owner_user_id AND 
				resource_id = $1 AND 
				resource_kind = 'namespace'`, ret.ID)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}

	for rows.Next() {
		var pr rstypes.PermissionRecord
		if err = rows.StructScan(&pr); err != nil {
			err = models.WrapDBError(err)
			return
		}
		ret.Users = append(ret.Users, pr)
	}

	return
}

func (db *pgDB) DeleteUserNamespaceByLabel(ctx context.Context, userID, label string) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debug("delete user namespace by label")

	result, err := db.eLog.ExecContext(ctx, `
		WITH user_ns AS (
			SELECT resource_id
			FROM permissions
			WHERE owner_user_id = user_id AND 
					user_id = $1 AND 
					resource_label = $2 AND
					resource_kind = 'namespace'
		)
		UPDATE namespaces
		SET deleted = TRUE
		WHERE id IN (SELECT * FROM user_ns)`, userID, label)
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

	result, err := db.eLog.ExecContext(ctx, `
		WITH user_ns AS (
			SELECT resource_id
			FROM permissions
			WHERE owner_user_id = user_id AND 
					user_id = $1 AND 
					resource_kind = 'namespace'
		)
		UPDATE namespaces
		SET deleted = TRUE
		WHERE id IN (SELECT * FROM user_ns)`, userID)
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrLabeledResourceNotExists
	}

	return
}

func (db *pgDB) RenameNamespace(ctx context.Context, userID, oldLabel, newLabel string) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"old_label": oldLabel,
		"new_label": newLabel,
	}).Debug("rename namespace")

	var exists bool
	if exists, err = db.isNamespaceExists(ctx, userID, newLabel); err != nil {
		return
	}
	if exists {
		err = models.ErrLabeledResourceExists
		return
	}

	result, err := db.eLog.ExecContext(ctx, `
		UPDATE permissions
		SET resource_label = $2
		WHERE owner_user_id = $1 AND
				resource_kind = 'namespace' AND
				resource_label = $3`, userID, newLabel, oldLabel)
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

	result, err := db.eLog.ExecContext(ctx, `
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
		userID,
		label,
		namespace.TariffID,
		namespace.RAM,
		namespace.CPU,
		namespace.MaxExternalServices,
		namespace.MaxIntServices,
		namespace.MaxTraffic)
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrLabeledResourceNotExists
	}

	return
}
