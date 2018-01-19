package models

import (
	"database/sql"
	"fmt"
	"time"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	rserrors "git.containerum.net/ch/resource-service/server/errors"

	"context"

	"git.containerum.net/ch/utils"
)

type Volume = rstypes.Volume

func (db ResourceSvcDB) VolumeCreate(ctx context.Context, tariff rstypes.VolumeTariff, user string, label string) (volID string, err error) {
	volID = utils.NewUUID()
	{
		var count int
		err = db.qLog.QueryRowxContext(ctx, `SELECT count(*) FROM accesses WHERE user_id=$1 AND resource_label=$2 AND kind='Volume'`, user, label).Scan(&count)
		if err != nil {
			return
		}
		if count > 0 {
			err = rserrors.ErrAlreadyExists
			return
		}
	}

	_, err = db.eLog.ExecContext(ctx,
		`INSERT INTO volumes (
			id,
			capacity,
			replicas,
			tariff_id,
			is_persistent
		) VALUES ($1,$2,$3,$4,$5)`,
		volID,
		tariff.StorageLimit,
		tariff.ReplicasLimit,
		tariff.TariffID,
		tariff.IsPersistent,
	)
	if err != nil {
		return
	}

	_, err = db.eLog.ExecContext(ctx,
		`INSERT INTO accesses(
			id,
			kind,
			resource_id,
			resource_label,
			user_id,
			owner_user_id,
			access_level,
			access_level_change_time,
			limited,
			new_access_level
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		utils.NewUUID(),
		"Volume",
		volID,
		label,
		user,
		user,
		"owner",
		time.Now(),
		false,
		"owner",
	)
	return
}

func (db ResourceSvcDB) VolumeList(ctx context.Context, user string) (vols []Volume, err error) {
	rows, err := db.qLog.QueryContext(ctx,
		`SELECT
			v.id,
			v.create_time,
			v.deleted,
			v.delete_time,
			v.tariff_id,
			a.resource_label,
			a.access_level,
			a.access_level_change_time,
			a.limited,
			a.new_access_level,
			v.capacity,
			v.replicas,
			v.is_persistent
		FROM volumes v INNER JOIN accesses a ON a.resource_id=v.id
		WHERE a.user_id=$1 AND a.kind='Volume'`,
		user)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var vol Volume
		err = rows.Scan(
			&vol.ID,
			&vol.CreateTime,
			&vol.Deleted,
			&vol.DeleteTime,
			&vol.TariffID,
			&vol.Label,
			&vol.Access,
			&vol.AccessChangeTime,
			&vol.Limited,
			&vol.NewAccess,
			&vol.Storage,
			&vol.Replicas,
			&vol.Persistent,
		)
		if err != nil {
			return
		}
		vols = append(vols, vol)
	}
	return
}

func (db ResourceSvcDB) VolumeRename(ctx context.Context, user string, oldname, newname string) (err error) {
	_, err = db.eLog.ExecContext(ctx,
		`UPDATE accesses SET resource_label=$1
		WHERE resource_label=$2 AND user_id=$3 AND kind='Volume'`,
		newname,
		oldname,
		user,
	)
	return
}

func (db ResourceSvcDB) VolumeSetLimited(ctx context.Context, owner string, ownerLabel string, limited bool) (err error) {
	_, err = db.eLog.ExecContext(ctx,
		`UPDATE accesses SET limited=$3
		WHERE user_id=$1 AND resource_label=$2 AND kind='Volume'`,
		owner,
		ownerLabel,
		limited,
	)
	return
}

func (db ResourceSvcDB) VolumeSetAccess(ctx context.Context, owner string, label string, other string, access string) (err error) {
	var resID string

	// get resource id
	err = db.qLog.QueryRowxContext(ctx,
		`SELECT resource_id FROM accesses
		WHERE user_id=$1 AND resource_label=$2 AND owner_user_id=user_id AND kind='Volume'`,
		owner,
		label,
	).Scan(&resID)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrNoSuchResource
		return
	default:
		return
	}

	if other == owner {
		_, err = db.eLog.ExecContext(ctx,
			`UPDATE accesses SET new_access_level=$1
			WHERE owner_user_id=$2 AND resource_id=$3 AND kind='Volume'`,
			access,
			owner,
			resID,
		)
	} else {
		_, err = db.eLog.ExecContext(ctx,
			`INSERT INTO accesses (
					id,
					kind,
					resource_id,
					resouce_label,
					user_id,
					owner_user_id,
					access_level,
					new_access_level
					) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
					ON CONFLICT (resource_id, user_id) DO UPDATE SET new_access_level = $8`,
			utils.NewUUID(),
			"Volume",
			resID,
			utils.NewUUID(),
			other,
			owner,
			access,
			access,
		)
	}

	return
}

func (db ResourceSvcDB) VolumeSetTariff(ctx context.Context, owner string, label string, t rstypes.VolumeTariff) (err error) {
	var resID string

	// check if owner & ns_label exists by getting its ID
	err = db.qLog.QueryRowxContext(ctx,
		`SELECT resource_id FROM accesses
		WHERE owner_user_id=user_id AND user_id=$1 AND resource_label=$2
			AND kind='Volume'`,
		owner,
		label,
	).Scan(&resID)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrNoSuchResource
		return
	default:
		return
	}

	// UPDATE tariff_id and the rest of the fields
	_, err = db.eLog.ExecContext(ctx,
		`UPDATE volumes SET
			tariff_id=$2,
			capacity=$3,
			replicas=$4,
			is_persistent=$5
		WHERE id=$1`,
		resID,
		t.TariffID,
		t.StorageLimit,
		t.ReplicasLimit,
		t.IsPersistent,
	)
	return
}

func (db ResourceSvcDB) VolumeDelete(ctx context.Context, user string, label string) (err error) {
	var alvl string
	var owner string
	var resID string

	err = db.qLog.QueryRowxContext(ctx,
		`SELECT access_level, owner_user_id, resource_id
		FROM accesses
		WHERE user_id=$1 AND resource_label=$2 AND kind='Volume'`,
		user,
		label,
	).Scan(
		&alvl,
		&owner,
		&resID,
	)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrNoSuchResource
		return
	default:
		return
	}

	if owner == user {
		_, err = db.eLog.ExecContext(ctx,
			`UPDATE volumes SET deleted=true, delete_time=statement_timestamp()
			WHERE id=$1`,
			resID,
		)
		if err != nil {
			err = fmt.Errorf("UPDATE volumes ... : %[1]v <%[1]T>", err)
			return
		}

		_, err = db.eLog.ExecContext(ctx, `DELETE FROM accesses WHERE resource_id=$1`, resID)
		if err != nil {
			err = fmt.Errorf("DELETE FROM accesses ...: %[1]v <%[1]T>", err)
			return
		}
		_, err = db.eLog.ExecContext(ctx, `DELETE FROM namespace_volume WHERE vol_id=$1`, resID)
		if err != nil {
			err = fmt.Errorf("DELETE FROM namespace_volume ...: %[1]v <%[1]T>", err)
			return
		}
	} else {
		_, err = db.eLog.ExecContext(ctx, `DELETE FROM accesses WHERE resource_id=$1 AND user_id=$2`, resID, user)
		if err != nil {
			err = fmt.Errorf("DELETE FROM accesses ...: %[1]v <%[1]T>", err)
			return
		}
	}

	return
}

func (db ResourceSvcDB) VolumeAccesses(ctx context.Context, owner string, label string) (vol Volume, err error) {
	err = db.qLog.QueryRowxContext(ctx,
		`SELECT
			v.id,
			v.create_time,
			v.deleted,
			v.delete_time,
			a.user_id,
			v.tariff_id,
			a.resource_label,
			a.access_level,
			a.access_level_change_time,
			v.capacity,
			v.replicas,
			v.is_persistent
		FROM accesses a INNER JOIN volumes v ON v.id=a.resource_id
		WHERE a.user_id=$1 AND a.resource_label=$2 AND a.owner_user_id=a.user_id AND a.kind='Volume'`,
		owner,
		label,
	).Scan(
		&vol.ID,
		&vol.CreateTime,
		&vol.Deleted,
		&vol.DeleteTime,
		&vol.UserID,
		&vol.TariffID,
		&vol.Label,
		&vol.Access,
		&vol.AccessChangeTime,
		&vol.Storage,
		&vol.Replicas,
		&vol.Persistent,
	)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrNoSuchResource
		return
	default:
		return
	}

	rows, err := db.qLog.QueryContext(ctx,
		`SELECT
			user_id,
			access_level,
			limited,
			new_access_level,
			access_level_change_time
		FROM accesses
		WHERE kind='Volume' AND resource_id=$1`,
		vol.ID,
	)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var ar rstypes.AccessRecord
		err = rows.Scan(
			&ar.UserID,
			&ar.Access,
			&ar.Limited,
			&ar.NewAccess,
			&ar.AccessChangeTime,
		)
		if err != nil {
			return
		}
		vol.Users = append(vol.Users, ar)
	}
	return
}
