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
	params := map[string]interface{}{
		"user_id": userID,
		"label":   label,
	}
	entry := db.log.WithFields(params)
	entry.Debug("check if volume exists")

	var count int
	query, args, _ := sqlx.Named(`
		SELECT count(ns.*)
		FROM volumes ns
		JOIN permissions p ON p.resource_id = ns.id AND p.resource_kind = 'volume'
		WHERE p.user_id = $1 AND p.resource_label = $2`, params)
	err = sqlx.GetContext(ctx, db.extLog, &count, db.extLog.Rebind(query), args...)
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
	var volsWithNsID []struct {
		rstypes.VolumeWithPermission
		NsID string `db:"ns_id"`
	}
	query, args, _ := sqlx.In(`
		SELECT v.*, perm.*, nv.ns_id
		FROM namespace_volume nv
		JOIN volumes v ON nv.vol_id = v.id
		JOIN permissions perm ON perm.resource_id = v.id
		WHERE nv.ns_id IN (?)`, nsIDs)
	err = sqlx.SelectContext(ctx, db.extLog, volsWithNsID, db.extLog.Rebind(query), args...)
	if err != nil {
		return
	}
	for _, v := range volsWithNsID {
		ns := nsMap[v.NsID]
		ns.Volume = append(ns.Volume, v.VolumeWithPermission)
		nsMap[v.NsID] = ns
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

	query, args, _ := sqlx.Named(`
		INSERT INTO volumes
		(
			tariff_id,
			capacity,
			replicas,
			is_persistent,
		)
		VALUES (:tariff_id, :capacity, :replicas, :is_persistent)
		RETURNING *`,
		volume)
	err = sqlx.GetContext(ctx, db.extLog, volume, db.extLog.Rebind(query), args...)
	if err != nil {
		err = models.WrapDBError(err)
		return
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
		VALUES ('volume', :resource_id, :resource_label, :user_id, :user_id)`,
		map[string]interface{}{
			"resource_id": volume.ID,
			"label":       label,
			"user_id":     userID,
		})
	if err != nil {
		err = models.WrapDBError(err)
	}

	return
}

func (db *pgDB) GetUserVolumes(ctx context.Context,
	userID string, filters *models.VolumeFilterParams) (ret []rstypes.VolumeWithPermission, err error) {
	ret = make([]rstypes.VolumeWithPermission, 0)
	db.log.WithField("user_id", userID).Debugf("get user volumes (filters %#v)", filters)

	params := struct {
		UserID string `db:"user_id"`
		*models.VolumeFilterParams
	}{
		UserID:             userID,
		VolumeFilterParams: filters,
	}
	query, args, _ := sqlx.Named(`
		SELECT v.*, p.*
		FROM volumes v
		JOIN permissions p ON p.resource_id = v.id AND p.kind = 'volume'
		WHERE 
			p.user_id = :user_id AND
			(NOT v.deleted OR NOT :not_deleted) AND
			(v.deleted OR NOT :deleted) AND
			(p.limited OR NOT :limited) AND
			(NOT p.limited OR NOT :not_limited) AND
			(p.owner_user_id = p_user_id OR NOT :owned) AND
			(p.is_persistent OR NOT :persistent) AND
			(NOT p.is_persistent OR NOT :not_persistent)
		ORDER BY v.create_time DESC`,
		params)

	err = sqlx.SelectContext(ctx, db.extLog, ret, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = models.ErrResourceNotExists
	default:
		err = models.WrapDBError(err)
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

	params := struct {
		Limit  int `db:"limit"`
		Offset int `db:"offset"`
		*models.VolumeFilterParams
	}{
		Limit:              perPage,
		Offset:             (page - 1) * perPage,
		VolumeFilterParams: filters,
	}
	query, args, _ := sqlx.Named(`
			SELECT v.*, p.*
			FROM volumes v
			JOIN permissions p ON p.resource_id = v.id AND p.kind = 'volume'
			WHERE 
				(NOT v.deleted OR NOT :not_deleted) AND
				(v.deleted OR NOT :deleted) AND
				(p.limited OR NOT :limited) AND
				(NOT p.limited OR NOT :not_limited) AND
				(p.owner_user_id = p_user_id OR NOT :owned) AND
				(p.is_persistent OR NOT :persistent) AND
				(NOT p.is_persistent OR NOT :not_persistent)
			ORDER BY v.create_time DESC
			LIMIT :limit
			OFFSET :offset`,
		params)

	err = sqlx.SelectContext(ctx, db.extLog, ret, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = models.ErrResourceNotExists
	default:
		err = models.WrapDBError(err)
	}

	return
}

func (db *pgDB) GetUserVolumeByLabel(ctx context.Context,
	userID, label string) (ret rstypes.VolumeWithPermission, err error) {
	params := map[string]interface{}{
		"user_id":        userID,
		"resource_label": label,
	}
	db.log.WithFields(params).Debug("get user volume by label")

	query, args, _ := sqlx.Named(`
		SELECT v.*, p.*
		FROM volumes v
		JOIN permissions p ON p.resource_id = v.id AND p.kind = 'volume'
		WHERE p.user_id = p.owner_user_id AND p.user_id = :user_id AND p.resource_label = :resource_label`,
		params)
	err = sqlx.SelectContext(ctx, db.extLog, &ret, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = models.ErrResourceNotExists
	default:
		err = models.WrapDBError(err)
	}

	return
}

func (db *pgDB) GetVolumeWithUserPermissions(ctx context.Context,
	userID, label string) (ret rstypes.VolumeWithUserPermissions, err error) {
	params := map[string]interface{}{
		"user_id":        userID,
		"resource_label": label,
	}
	db.log.WithFields(params).Debug("get volume with user permissions")

	query, args, _ := sqlx.Named(`
		SELECT v.*, p.*
		FROM volumes v
		JOIN permissions p ON p.resource_id = v.id AND p.kind = 'volume'
		WHERE p.user_id = p.owner_user_id AND p.user_id = $1 AND p.resource_label = $2`,
		params)
	err = sqlx.GetContext(ctx, db.extLog, &ret.VolumeWithPermission, db.extLog.Rebind(query), args...)
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
		WHERE owner_user_id != user_id AND
				resource_kind = 'volume' AND
				resource_id = :resource_id`,
		map[string]interface{}{
			"resource_id": ret.ID,
		})
	err = sqlx.SelectContext(ctx, db.extLog, ret.Users, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = models.ErrResourceNotExists
	default:
		err = models.WrapDBError(err)
	}

	return
}

func (db *pgDB) DeleteUserVolumeByLabel(ctx context.Context, userID, label string) (err error) {
	params := map[string]interface{}{
		"user_id":        userID,
		"resource_label": label,
	}
	db.log.WithFields(params).Debug("delete user volume by label")

	result, err := sqlx.NamedExecContext(ctx, db.extLog, `
		WITH user_vol AS (
			SELECT resource_id
			FROM permissions
			WHERE user_id = owner_user_id AND
					kind = 'volume' AND
					user_id = :user_id AND 
					resource_label = :resource_label
		)
		UPDATE volumes
		SET deleted = TRUE
		WHERE id IN (SELECT * FROM user_vol)`,
		params)
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrLabeledResourceNotExists
	}

	return
}

func (db *pgDB) DeleteAllUserVolumes(ctx context.Context, userID string) (err error) {
	params := map[string]interface{}{
		"user_id": userID,
	}
	db.log.WithFields(params).Debug("delete all user volumes")

	result, err := sqlx.NamedExecContext(ctx, db.extLog, `
		WITH user_vol AS (
			SELECT resource_id
			FROM permissions
			WHERE user_id = owner_user_id AND 
					kind = 'volume' AND 
					user_id = :user_id
		)
		UPDATE volumes
		SET deleted = TRUE
		WHERE id IN (SELECT * FROM user_vol)`,
		params)
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrLabeledResourceNotExists
	}

	return
}

func (db *pgDB) RenameVolume(ctx context.Context, userID, oldLabel, newLabel string) (err error) {
	params := map[string]interface{}{
		"user_id":   userID,
		"old_label": oldLabel,
		"new_label": newLabel,
	}
	db.log.WithFields(params).Debug("rename user volume")

	result, err := sqlx.NamedExecContext(ctx, db.extLog, `
		UPDATE permissions
		SET resource_label = :old_label
		WHERE owner_user_id = :user_id AND
				resource_kind = 'volume' AND
				resource_label = :new_label`,
		params)
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

	params := struct {
		UserID string `db:"user_id"`
		Label  string `db:"label"`
		*rstypes.Volume
	}{
		UserID: userID,
		Label:  label,
		Volume: volume,
	}
	result, err := sqlx.NamedExecContext(ctx, db.extLog, `
		WITH user_vol AS (
			SELECT resource_id
			FROM permissions
			WHERE owner_user_id = user_id AND 
				user_id = :user_id AND 
				resource_kind = 'volume' AND
				resource_label = :label
		)
		UPDATE volumes
		SET
			tariff_id = :tariff_id,
			capacity = :capacity,
			replicas = :replicas
		WHERE id IN (SELECT * FROM user_vol)`,
		params)
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrLabeledResourceNotExists
	}

	return
}

func (db *pgDB) SetVolumeActiveByID(ctx context.Context, id string, active bool) (err error) {
	params := map[string]interface{}{
		"id":     id,
		"active": active,
	}
	db.log.WithFields(params).Debug("activating volume by id")

	result, err := sqlx.NamedExecContext(ctx, db.extLog, `UPDATE volumes SET active = :id WHERE id = :active`, params)
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrResourceNotExists
	}

	return
}

func (db *pgDB) SetUserVolumeActive(ctx context.Context, userID, label string, active bool) (err error) {
	params := map[string]interface{}{
		"user_id": userID,
		"label":   label,
		"active":  active,
	}
	db.log.WithFields(params).Debug("activating user volume")

	result, err := sqlx.NamedExecContext(ctx, db.extLog, `
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
		WHERE id IN (SELECT * FROM user_vol)`,
		params)
	if err != nil {
		err = models.WrapDBError(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = models.ErrLabeledResourceNotExists
	}

	return
}
