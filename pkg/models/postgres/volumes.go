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

type VolumePG struct {
	models.RelationalDB
	log *cherrylog.LogrusAdapter
}

func NewVolumePG(db models.RelationalDB) models.VolumeDB {
	return &VolumePG{
		RelationalDB: db,
		log:          cherrylog.NewLogrusAdapter(logrus.WithField("component", "volume_pg")),
	}
}

func (db *VolumePG) GetVolumeID(ctx context.Context, userID, label string) (volID string, err error) {
	params := map[string]interface{}{
		"user_id": userID,
		"label":   label,
	}
	entry := db.log.WithFields(params)
	entry.Debug("get volume id")

	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT v.id
		FROM volumes v
		JOIN permissions p ON p.resource_id = v.id AND p.kind = 'volume'
		WHERE (p.user_id = :user_id OR p.owner_user_id = :user_id) AND 
			p.resource_label = :label`,
		params)
	err = sqlx.GetContext(ctx, db, &volID, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("volume %s not exists", label)
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}

func (db *VolumePG) CreateVolume(ctx context.Context, userID, label string, volume *rstypes.Volume) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id": userID,
		"label":   label,
	}).Infof("creating volume %#v", volume)

	_, err = db.GetVolumeID(ctx, userID, label)
	if err == nil {
		err = rserrors.ErrResourceAlreadyExists().AddDetailF("volume %s already exists", label)
		return
	}
	if err != nil && !cherry.Equals(err, rserrors.ErrResourceNotExists()) {
		return
	}

	volume.OwnerUserID = userID
	query, args, _ := sqlx.Named( /* language=sql */
		`INSERT INTO volumes
		(
			tariff_id,
			capacity,
			replicas,
			ns_id,
			storage_id,
			gluster_name,
			owner_user_id
		)
		VALUES (:tariff_id, :capacity, :replicas, :ns_id, :storage_id, :gluster_name, :owner_user_id)
		RETURNING *`,
		volume)
	err = sqlx.GetContext(ctx, db, volume, db.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
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
		VALUES ('volume', :resource_id, :resource_label, :user_id, :user_id)`,
		rstypes.PermissionRecord{
			ResourceID:    &volume.ID,
			ResourceLabel: label,
			UserID:        userID,
		})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}

func (db *VolumePG) GetUserVolumes(ctx context.Context,
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
		JOIN permissions p ON p.resource_id = v.id AND p.kind = 'volume'
		WHERE 
			(p.user_id = :user_id OR -- return borrowed by default
			p.owner_user_id = :user_id) AND -- return owned by defaults
			(NOT v.deleted OR NOT :not_deleted) AND
			(v.deleted OR NOT :deleted) AND
			(p.limited OR NOT :limited) AND
			(NOT p.limited OR NOT :not_limited) AND
			(p.owner_user_id = p.user_id OR NOT :owned) AND
			(v.ns_id IS NULL OR NOT :persistent) AND
			(v.ns_id IS NOT NULL OR NOT :not_persistent)
		ORDER BY v.create_time DESC`,
		params)

	err = sqlx.SelectContext(ctx, db, &ret, db.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}

func (db *VolumePG) GetAllVolumes(ctx context.Context,
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
			JOIN permissions p ON p.resource_id = v.id AND p.kind = 'volume'
			WHERE 
				(NOT v.deleted OR NOT :not_deleted) AND
				(v.deleted OR NOT :deleted) AND
				(p.limited OR NOT :limited) AND
				(NOT p.limited OR NOT :not_limited) AND
				(p.owner_user_id = p.user_id OR NOT :owned) AND
				(v.ns_id IS NULL OR NOT :persistent) AND
				(v.ns_id IS NOT NULL OR NOT :not_persistent)
			ORDER BY v.create_time DESC
			LIMIT :limit
			OFFSET :offset`,
		params)

	err = sqlx.SelectContext(ctx, db, &ret, db.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}

func (db *VolumePG) GetUserVolumeByLabel(ctx context.Context,
	userID, label string) (ret rstypes.VolumeWithPermission, err error) {
	params := map[string]interface{}{
		"user_id": userID,
		"label":   label,
	}
	db.log.WithFields(params).Debug("get user volume by label")

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
		JOIN permissions p ON p.resource_id = v.id AND p.kind = 'volume'
		WHERE (p.user_id = :user_id OR p.owner_user_id = :user_id) AND p.resource_label = :label`,
		params)
	err = sqlx.GetContext(ctx, db, &ret, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("volume %s not exists", label).Log(err, db.log)
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
	}
	ret.PermissionRecord.OwnerUserID = ret.Volume.OwnerUserID

	return
}

func (db *VolumePG) GetVolumeWithUserPermissions(ctx context.Context,
	userID, label string) (ret rstypes.VolumeWithUserPermissions, err error) {
	params := map[string]interface{}{
		"user_id": userID,
		"label":   label,
	}
	db.log.WithFields(params).Debug("get volume with user permissions")

	ret.Users = make([]rstypes.PermissionRecord, 0)

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
		JOIN permissions p ON p.resource_id = v.id AND p.kind = 'volume'
		WHERE (p.user_id = :user_id OR p.owner_user_id = :user_id) AND p.resource_label = :label`,
		params)
	err = sqlx.GetContext(ctx, db, &ret.VolumeWithPermission, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("volume %s not exists", label).Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	ret.PermissionRecord.OwnerUserID = ret.Volume.OwnerUserID

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
		WHERE owner_user_id != user_id AND
				kind = 'volume' AND
				resource_id = :id`,
		ret.Resource)
	err = sqlx.SelectContext(ctx, db, &ret.Users, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrAccessRecordNotExists().Log(err, db.log)
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}

func (db *VolumePG) GetVolumesLinkedWithUserNamespace(ctx context.Context, userID, nsLabel string) (ret []rstypes.VolumeWithPermission, err error) {
	params := map[string]interface{}{
		"user_id":  userID,
		"ns_label": nsLabel,
	}
	db.log.WithFields(params).Debug("get volumes linked with user namespace")

	ret = make([]rstypes.VolumeWithPermission, 0)

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
		JOIN containers c ON vm.container_id = c.id
		JOIN deployments d ON c.depl_id = d.id
		JOIN permissions p ON p.resource_id = d.ns_id AND p.kind = 'namespace'
		WHERE (p.user_id = :user_id OR p.owner_user_id = :user_id) AND p.resource_label = :ns_label`,
		params)
	err = sqlx.SelectContext(ctx, db, &ret, db.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}

func (db *VolumePG) DeleteUserVolumeByLabel(ctx context.Context, userID, label string) (volume rstypes.Volume, err error) {
	params := map[string]interface{}{
		"user_id":        userID,
		"resource_label": label,
	}
	db.log.WithFields(params).Debug("delete user volume by label")

	query, args, _ := sqlx.Named( /* language=sql */
		`WITH user_vol AS (
			SELECT resource_id
			FROM permissions
			WHERE user_id = owner_user_id AND
					kind = 'volume' AND
					user_id = :user_id AND 
					resource_label = :resource_label
		)
		UPDATE volumes
		SET deleted = TRUE, active = FALSE, delete_time = now()
		WHERE id IN (SELECT resource_id FROM user_vol) AND NOT deleted
		RETURNING *`,
		params)
	err = sqlx.GetContext(ctx, db, &volume, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("volume %s not exists", label).Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	return
}

func (db *VolumePG) DeleteAllUserVolumes(ctx context.Context, userID string, nonPersistentOnly bool) (ret []rstypes.Volume, err error) {
	params := map[string]interface{}{
		"user_id":             userID,
		"non_persistent_only": nonPersistentOnly,
	}
	db.log.WithFields(params).Debug("delete all user volumes")

	query, args, _ := sqlx.Named( /* language=sql */
		`WITH user_vol AS (
			SELECT resource_id
			FROM permissions
			WHERE user_id = owner_user_id AND 
					kind = 'volume' AND 
					user_id = :user_id
		)
		UPDATE volumes
		SET deleted = TRUE, active = FALSE, delete_time = now()
		WHERE id IN (SELECT resource_id FROM user_vol) AND 
			(ns_id IS NOT NULL OR NOT :non_persistent_only) AND 
			NOT deleted
		RETURNING *`,
		params)
	ret = make([]rstypes.Volume, 0)
	err = sqlx.SelectContext(ctx, db, &ret, db.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	return
}

func (db *VolumePG) RenameVolume(ctx context.Context, userID, oldLabel, newLabel string) (err error) {
	params := map[string]interface{}{
		"user_id":   userID,
		"old_label": oldLabel,
		"new_label": newLabel,
	}
	db.log.WithFields(params).Debug("rename user volume")

	_, err = db.GetVolumeID(ctx, userID, oldLabel)
	if err == nil {
		err = rserrors.ErrResourceAlreadyExists().AddDetailF("volume %s already exists", oldLabel)
		return
	}
	if err != nil && !cherry.Equals(err, rserrors.ErrResourceNotExists()) {
		return
	}

	result, err := sqlx.NamedExecContext(ctx, db, /* language=sql */
		`UPDATE permissions
		SET resource_label = :old_label
		WHERE owner_user_id = :user_id AND
				kind = 'volume' AND
				resource_label = :new_label`,
		params)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = rserrors.ErrResourceNotExists().AddDetailF("volume %s not exists", oldLabel).Log(err, db.log)
	}

	return
}

func (db *VolumePG) ResizeVolume(ctx context.Context, volume *rstypes.Volume) (err error) {
	db.log.WithField("volume_id", volume.ID).Debugf("update volume to %#v", volume)

	query, args, _ := sqlx.Named( /* language=sql */
		`UPDATE volumes
		SET
			tariff_id = :tariff_id,
			capacity = :capacity,
			replicas = :replicas
		WHERE id = :id`,
		volume)
	err = sqlx.GetContext(ctx, db, volume, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("volume %s not exists", volume.ID).Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	return
}

func (db *VolumePG) SetVolumeActiveByID(ctx context.Context, id string, active bool) (err error) {
	params := map[string]interface{}{
		"id":     id,
		"active": active,
	}
	db.log.WithFields(params).Debug("activating volume by id")

	result, err := sqlx.NamedExecContext(ctx, db, /* language=sql */
		`UPDATE volumes SET active = :id WHERE id = :active`, params)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = rserrors.ErrResourceNotExists().AddDetailF("volume %s not exists", id).Log(err, db.log)
	}

	return
}

func (db *VolumePG) SetUserVolumeActive(ctx context.Context, userID, label string, active bool) (err error) {
	params := map[string]interface{}{
		"user_id": userID,
		"label":   label,
		"active":  active,
	}
	db.log.WithFields(params).Debug("activating user volume")

	result, err := sqlx.NamedExecContext(ctx, db, /* language=sql */
		`WITH user_vol AS (
			SELECT resource_id
			FROM permissions
			WHERE owner_user_id = user_id AND 
				user_id = :user_id AND 
				kind = 'volume' AND
				resource_label = :label
		)
		UPDATE volumes 
		SET active = :active
		WHERE id IN (SELECT resource_id FROM user_vol)`,
		params)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		err = rserrors.ErrResourceNotExists().AddDetailF("volume %s not exists", label).Log(err, db.log)
	}

	return
}
