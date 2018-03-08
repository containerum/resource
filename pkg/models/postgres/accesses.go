package postgres

import (
	"context"

	"git.containerum.net/ch/grpc-proto-files/auth"
	"git.containerum.net/ch/json-types/misc"
	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

func (db *pgDB) GetUserResourceAccesses(ctx context.Context, userID string) (ret *auth.ResourcesAccess, err error) {
	db.log.WithField("user_id", userID).Debug("get user resource access")

	accessObjects := make([]struct {
		Kind rstypes.Kind `db:"kind"`
		*auth.AccessObject
	}, 0)

	err = sqlx.SelectContext(ctx, db.extLog, &accessObjects, /* language=sql */
		`SELECT kind, resource_label AS label, resource_id AS id, new_access_level AS access
		FROM permissions
		WHERE owner_user_id = user_id AND user_id = $1 AND kind IN ('namespace', 'volume')`, userID)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	ret = &auth.ResourcesAccess{
		Volume:    make([]*auth.AccessObject, 0),
		Namespace: make([]*auth.AccessObject, 0),
	}
	for _, obj := range accessObjects {
		switch obj.Kind {
		case rstypes.KindNamespace:
			ret.Namespace = append(ret.Namespace, obj.AccessObject)
		case rstypes.KindVolume:
			ret.Volume = append(ret.Volume, obj.AccessObject)
		default:
			db.log.Errorf("unexpected kind %s", obj.Kind)
		}
	}

	return
}

func (db *pgDB) SetResourceAccess(ctx context.Context, permRec *rstypes.PermissionRecord) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":      permRec.UserID,
		"label":        permRec.ResourceLabel,
		"access_level": permRec.AccessLevel,
	}).Debugf("set %s access", permRec.Kind)

	query, args, _ := sqlx.Named( /* language=sql */
		`INSERT INTO permissions (
			kind,
			owner_user_id,
			user_id,
			resource_id,
			resource_label,
			access_level,
			new_access_level
		)
		VALUES (
			:kind,
			:owner_user_id,
			:user_id,
			:resource_id,
			:resource_label,
			:access_level,
			:access_level
		)
		ON CONFLICT (kind, resource_id, resource_label, user_id) DO UPDATE SET
			access_level = EXCLUDED.access_level,
			new_access_level = EXCLUDED.access_level,
			access_level_change_time = now() AT TIME ZONE 'UTC'
		RETURNING
			id AS perm_id,
			kind,
			resource_id,
			resource_label,
			owner_user_id,
			create_time,
			user_id,
			access_level,
			limited,
			access_level_change_time,
			new_access_level`,
		permRec)
	err = sqlx.GetContext(ctx, db.extLog, permRec, db.extLog.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}

func (db *pgDB) SetAllResourcesAccess(ctx context.Context, userID string, access rstypes.PermissionStatus) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":          userID,
		"new_access_level": access,
	}).Debug("set user resources access")

	_, err = sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
		`WITH current_user_access AS (
			SELECT id
		  	FROM permissions
		  	WHERE user_id = owner_user_id AND user_id = :user_id
		), updated_owner_accesses AS (
			UPDATE permissions
			SET limited = CASE WHEN new_access_level > :new_access_level THEN TRUE
						  		ELSE FALSE END,
				new_access_level = CASE WHEN new_access_level > :new_access_level THEN :new_access_level
										ELSE access_level END,
				access_level_change_time = now() AT TIME ZONE 'UTC'						
			WHERE id IN (SELECT id FROM current_user_access)
			RETURNING *		  
		)
		UPDATE permissions
		SET limited = (new_access_level > :new_access_level OR access_level > :new_access_level),
			new_access_level = CASE WHEN new_access_level > :new_access_level OR access_level > :new_access_level THEN :new_access_level
									ELSE access_level END,
			access_level_change_time = now() AT TIME ZONE 'UTC'
	  	WHERE owner_user_id IN (SELECT owner_user_id FROM updated_owner_accesses)`,
		rstypes.PermissionRecord{UserID: userID, NewAccessLevel: access})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}

func (db *pgDB) DeleteResourceAccess(ctx context.Context, resource rstypes.Resource, userID string) (err error) {
	db.log.WithFields(logrus.Fields{
		"resource_id": resource.ID,
		"user_id":     userID,
	}).Debug("delete resource access")

	result, err := sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
		`DELETE FROM permissions WHERE (user_id, resource_id) = (:user_id, :resource_id) AND owner_user_id != user_id`,
		rstypes.PermissionRecord{UserID: userID, ResourceID: misc.WrapString(resource.ID)})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	if count, _ := result.RowsAffected(); count <= 0 {
		err = rserrors.ErrAccessRecordNotExists().Log(err, db.log)
	}

	return
}
