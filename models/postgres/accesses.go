package postgres

import (
	"context"

	"git.containerum.net/ch/grpc-proto-files/auth"
	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

func (db *pgDB) GetUserResourceAccesses(ctx context.Context, userID string) (ret *auth.ResourcesAccess, err error) {
	db.log.WithField("user_id", userID).Debug("get user resource access")

	accessObjects := make([]struct {
		Kind string
		*auth.AccessObject
	}, 0)
	err = sqlx.SelectContext(ctx, db.extLog, &accessObjects, `
		SELECT kind, resource_label AS label, resource_id AS id, new_access_level AS access
		FROM permissions
		WHERE owner_user_id = user_id AND user_id = $1 AND kind in ('namespace', 'volume')`, userID)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}

	ret = &auth.ResourcesAccess{
		Volume:    make([]*auth.AccessObject, 0),
		Namespace: make([]*auth.AccessObject, 0),
	}
	for _, obj := range accessObjects {
		switch obj.Kind {
		case "namespace":
			ret.Namespace = append(ret.Namespace, obj.AccessObject)
		case "volume":
			ret.Volume = append(ret.Volume, obj.AccessObject)
		default:
			db.log.Errorf("unexpected kind %s", obj.Kind)
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

	result, err := sqlx.NamedExecContext(ctx, db.extLog, `
		WITH user_ns AS (
			SELECT resource_id
			FROM permissions
			WHERE owner_user_id = user_id AND 
					user_id = :user_id AND 
					resource_label = :resource_label AND
					resource_kind = :resource_kind
		)
		UPDATE permissions
		SET new_access_level = :access_level
		WHERE resource_id IN (SELECT * FROM user_ns)`,
		map[string]interface{}{
			"user_id":        userID,
			"resource_label": label,
			"resource_kind":  kind,
			"access_level":   access,
		})
	if err != nil {
		err = models.WrapDBError(err)
	}

	if count, _ := result.RowsAffected(); count == 0 {
		err = models.ErrLabeledResourceNotExists
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
