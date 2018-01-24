package postgres

import (
	"context"

	"database/sql"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

func (db *pgDB) isVolumeExists(ctx context.Context, userID, label string) (exists bool, err error) {
	entry := db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	})
	entry.Debug("check if volume exists")

	var count int
	err = db.qLog.QueryRowxContext(ctx, `
		SELECT count(ns.*)
		FROM volumes ns
		JOIN permissions p ON p.resource_id = ns.id AND p.resource_kind = 'volume'
		WHERE p.user_id = $1 AND p.resource_label = $2`,
		userID, label).Scan(&count)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}

	entry.Debugf("found %d volumes", count)
	exists = count > 0
	return
}

func (db *pgDB) addVolumesToNamespaces(ctx context.Context,
	nsIDs []string, nsMap map[string]rstypes.NamespaceWithVolumes) (err error) {
	db.log.Debugf("add volumes to namespaces %v", nsIDs)
	query, args, err := sqlx.In(`
		SELECT v.*, perm.*, nv.ns_id
		FROM namespace_volume nv
		JOIN volumes v ON nv.vol_id = v.id
		JOIN permissions perm ON perm.resource_id = v.id
		WHERE nv.ns_id IN (?)`, nsIDs)
	if err != nil {
		return
	}
	rows, err := db.qLog.QueryxContext(ctx, query, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var volWithNsID struct {
			rstypes.VolumeWithPermission
			NsID string `db:"ns_id"`
		}
		if err = rows.StructScan(&volWithNsID); err != nil {
			return
		}
		ns := nsMap[volWithNsID.NsID]
		ns.Volume = append(ns.Volume, volWithNsID.VolumeWithPermission)
		nsMap[volWithNsID.NsID] = ns
	}

	return
}

func (db *pgDB) CreateVolume(ctx context.Context, userID, label string, volume *rstypes.Volume) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Infof("creating volume %#v", volume)

	var exists bool
	if exists, err = db.isVolumeExists(ctx, userID, label); err != nil {
		return
	}
	if exists {
		err = models.ErrLabeledResourceExists
		return
	}

	err = db.qLog.QueryRowxContext(ctx, `
		INSERT INTO volumes
		(
			tariff_id,
			capacity,
			replicas,
			is_persistent,
		)
		VALUES ($1, $2, $3, $4)
		RETURNING *`,
		volume.TariffID,
		volume.Capacity,
		volume.Replicas,
		volume.Persistent).
		StructScan(volume)
	if err != nil {
		err = models.WrapDBError(err)
		return
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
		VALUES ('volume', $1, $2, $3, $3)`, volume.ID, label, userID)
	if err != nil {
		err = models.WrapDBError(err)
	}

	return
}

func (db *pgDB) GetUserVolumes(ctx context.Context,
	userID, filters *models.VolumeFilterParams) (ret []rstypes.VolumeWithPermission, err error) {
	ret = make([]rstypes.VolumeWithPermission, 0)
	db.log.WithField("user_id", userID).Debugf("get user volumes (filters %#v)", filters)

	rows, err := db.qLog.QueryxContext(ctx, `
		SELECT v.*, p.*
		FROM volumes v
		JOIN permissions p ON p.resource_id = v.id AND p.kind = 'volume'
		WHERE 
			p.user_id = $1 AND
			(NOT v.deleted OR NOT $2) AND
			(v.deleted OR NOT $3) AND
			(p.limited OR NOT $4) AND
			(NOT p.limited OR NOT $5) AND
			(p.owner_user_id = p_user_id OR NOT $6) AND
			(p.is_persistent OR NOT $7) AND
			(NOT p.is_persistent OR NOT $8)
		ORDER BY v.create_time DESC`,
		userID,
		filters.NotDeleted,
		filters.Deleted,
		filters.Limited,
		filters.NotLimited,
		filters.Owners,
		filters.Persistent,
		filters.NotPersistent)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var vol rstypes.VolumeWithPermission
		if err = rows.StructScan(&vol); err != nil {
			return
		}
		ret = append(ret, vol)
	}

	if len(ret) == 0 {
		err = models.ErrResourceNotExists
	}

	return
}

func (db *pgDB) GetAllVolumes(ctx context.Context,
	page, perPage int, filters *models.VolumeFilterParams) (ret []rstypes.VolumeWithPermission, err error) {
	ret = make([]rstypes.VolumeWithPermission, 0)

	db.log.WithFields(logrus.Fields{
		"page":     page,
		"per_page": perPage,
	}).Debug("get all volumes")
	rows, err := db.qLog.QueryxContext(ctx, `
			SELECT v.*, p.*
			FROM volumes v
			JOIN permissions p ON p.resource_id = v.id AND p.kind = 'volume'
			WHERE 
				(NOT v.deleted OR NOT $3) AND
				(v.deleted OR NOT $4) AND
				(p.limited OR NOT $5) AND
				(NOT p.limited OR NOT $6) AND
				(p.owner_user_id = p_user_id OR NOT $7) AND
				(p.is_persistent OR NOT $8) AND
				(NOT p.is_persistent OR NOT $9)
			ORDER BY v.create_time DESC
			LIMIT $1
			OFFSET $2`,
		page,
		(page-1)*perPage,
		filters.NotDeleted,
		filters.Deleted,
		filters.Limited,
		filters.NotLimited,
		filters.Owners,
		filters.Persistent,
		filters.NotPersistent)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var volume rstypes.VolumeWithPermission
		if err = rows.StructScan(&volume); err != nil {
			err = models.WrapDBError(err)
			return
		}
		ret = append(ret, volume)
	}

	if len(ret) == 0 {
		err = models.ErrResourceNotExists
	}

	return
}

func (db *pgDB) GetUserVolumeByLabel(ctx context.Context,
	userID, label string) (ret rstypes.VolumeWithPermission, err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debug("get user volume by label")

	err = db.qLog.QueryRowxContext(ctx, `
		SELECT v.*, p.*
		FROM volumes v
		JOIN permissions p ON p.resource_id = v.id AND p.kind = 'volume'
		WHERE p.user_id = p.owner_user_id AND p.user_id = $1 AND p.resource_label = $2`,
		userID, label).StructScan(&ret)
	if err != nil {
		err = models.WrapDBError(err)
	}

	return
}

func (db *pgDB) GetVolumeWithUserPermissions(ctx context.Context,
	userID, label string) (ret rstypes.VolumeWithUserPermissions, err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debug("get volume with user permissions")

	err = db.qLog.QueryRowxContext(ctx, `
		SELECT v.*, p.*
		FROM volumes v
		JOIN permissions p ON p.resource_id = v.id AND p.kind = 'volume'
		WHERE p.user_id = p.owner_user_id AND p.user_id = $1 AND p.resource_label = $2`,
		userID, label).StructScan(&ret.VolumeWithPermission)
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
		WHERE resource_kind = 'volume' AND resource_id = $1`, ret.ID)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var perm rstypes.PermissionRecord
		if err = rows.StructScan(&perm); err != nil {
			err = models.WrapDBError(err)
			return
		}
		ret.Users = append(ret.Users, perm)
	}

	return
}

func (db *pgDB) DeleteUserVolumeByLabel(ctx context.Context, userID, label string) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debug("delete user volume by label")

	result, err := db.eLog.ExecContext(ctx, `
		WITH user_vol AS (
			SELECT resource_id
			FROM permissions
			WHERE user_id = owner_user_id AND kind = 'volume' AND user_id = $1 AND resource_label = $2
		)
		UPDATE volumes
		SET deleted = TRUE
		WHERE id IN (SELECT * FROM user_vol)`, userID, label)
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrLabeledResourceNotExists
	}

	return
}

func (db *pgDB) DeleteAllUserVolumes(ctx context.Context, userID string) (err error) {
	db.log.WithField("user_id", userID).Debug("delete all user volumes")

	result, err := db.eLog.ExecContext(ctx, `
		WITH user_vol AS (
			SELECT resource_id
			FROM permissions
			WHERE user_id = owner_user_id AND kind = 'volume' AND user_id = $1
		)
		UPDATE volumes
		SET deleted = TRUE
		WHERE id IN (SELECT * FROM user_vol)`, userID)
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrLabeledResourceNotExists
	}

	return
}

func (db *pgDB) RenameVolume(ctx context.Context, userID, oldLabel, newLabel string) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"old_label": oldLabel,
		"new_label": newLabel,
	}).Debug("rename user volume")

	result, err := db.eLog.ExecContext(ctx, `
		UPDATE permissions
		SET resource_label = $2
		WHERE owner_user_id = $1 AND
				resource_kind = 'volume' AND
				resource_label = $3`, userID, newLabel, oldLabel)
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrLabeledResourceNotExists
	}

	return
}

func (db *pgDB) ResizeVolume(ctx context.Context, userID, label string, volume *rstypes.Volume) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Debugf("update volume to %#v", volume)

	result, err := db.eLog.ExecContext(ctx, `
		WITH user_vol AS (
			SELECT resource_id
			FROM permissions
			WHERE owner_user_id = user_id AND 
				user_id = $1 AND 
				resource_kind = 'volume' AND
				resource_label = $2
		)
		UPDATE volumes
		SET
			tariff_id = $3,
			capacity = $4,
			replicas = $5
		WHERE id IN (SELECT * FROM user_vol)`,
		userID,
		label,
		volume.TariffID,
		volume.Capacity,
		volume.Replicas)
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrLabeledResourceNotExists
	}

	return
}

func (db *pgDB) SetVolumeActiveByID(ctx context.Context, id string, active bool) (err error) {
	db.log.WithField("id", id).Debug("activating volume")

	result, err := db.eLog.ExecContext(ctx, `UPDATE volumes SET active = $2 WHERE id = $1`, id, active)
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrResourceNotExists
	}

	return
}

func (db *pgDB) SetUserVolumeActive(ctx context.Context, userID, label string, active bool) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
		"active":  active,
	}).Debug("activating volume")

	result, err := db.eLog.ExecContext(ctx, `
		WITH user_vol AS (
			SELECT resource_id
			FROM permissions
			WHERE owner_user_id = user_id AND 
				user_id = $1 AND 
				resource_kind = 'volume' AND
				resource_label = $2
		)
		UPDATE volumes 
		SET active = $2 
		WHERE id IN (SELECT * FROM user_vol)`, userID, label, active)
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrLabeledResourceNotExists
	}

	return
}
