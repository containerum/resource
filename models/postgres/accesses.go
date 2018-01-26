package postgres

import (
	"context"

	"git.containerum.net/ch/grpc-proto-files/auth"
	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"github.com/sirupsen/logrus"
)

func (db *pgDB) GetUserResourceAccesses(ctx context.Context, userID string) (ret *auth.ResourcesAccess, err error) {
	db.log.WithField("user_id", userID).Debug("get user resource access")

	rows, err := db.extLog.QueryxContext(ctx, `
		SELECT kind, resource_label, resource_id, new_access_level
		FROM permissions
		WHERE owner_user_id = user_id AND user_id = $1 AND kind in ('namespace', 'volume')`, userID)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	defer rows.Close()

	ret = &auth.ResourcesAccess{
		Volume:    make([]*auth.AccessObject, 0),
		Namespace: make([]*auth.AccessObject, 0),
	}
	for rows.Next() {
		var obj auth.AccessObject
		var kind string
		if err = rows.Scan(&kind, &obj.Label, &obj.Id, &obj.Access); err != nil {
			err = models.WrapDBError(err)
			return
		}
		switch kind {
		case "namespace":
			ret.Namespace = append(ret.Namespace, &obj)
		case "volume":
			ret.Volume = append(ret.Volume, &obj)
		default:
			db.log.Errorf("unexpected kind %s", kind)
		}
	}

	return
}

func (db *pgDB) setResourceAccess(ctx context.Context,
	kind rstypes.Kind, userID, label string, access rstypes.PermissionStatus) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":          userID,
		"label":            label,
		"new_access_level": access,
	}).Debugf("set %s access", kind)

	_, err = db.extLog.ExecContext(ctx, `
		WITH user_ns AS (
			SELECT resource_id
			FROM permissions
			WHERE owner_user_id = user_id AND 
					user_id = $1 AND 
					resource_label = $2 AND
					resource_kind = $3
		)
		UPDATE permissions
		SET new_access_level = $4
		WHERE resource_id IN (SELECT * FROM user_ns)`, userID, label, kind, access)
	if err != nil {
		err = models.WrapDBError(err)
	}

	return
}

func (db *pgDB) SetNamespaceAccess(ctx context.Context, userID, label string, access rstypes.PermissionStatus) (err error) {
	err = db.setResourceAccess(ctx, rstypes.KindNamespace, userID, label, access)

	return
}

func (db *pgDB) SetVolumeAccess(ctx context.Context, userID, label string, access rstypes.PermissionStatus) (err error) {
	err = db.setResourceAccess(ctx, rstypes.KindVolume, userID, label, access)

	return
}
