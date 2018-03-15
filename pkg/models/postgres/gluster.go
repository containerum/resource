package postgres

import (
	"context"

	"git.containerum.net/ch/json-types/kube-api"
	"git.containerum.net/ch/resource-service/pkg/models"
	rserrors "git.containerum.net/ch/resource-service/pkg/resourceServiceErrors"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

func (db *pgDB) CreateGlusterEndpoints(ctx context.Context, userID, nsLabel string) (ret []kube_api.Endpoint, err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Debug("create endpoints for gluster")

	nsID, err := db.getNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	if nsID == "" {
		err = rserrors.ErrResourceNotExists().Log(err, db.log)
		return
	}

	query, args, _ := sqlx.Named( /* language=sql */
		`WITH ns_volumes AS (
			SELECT v.storage_id
			FROM volume_mounts vm
			JOIN volumes v ON vm.volume_id = v.id
			JOIN containers c ON vm.container_id = c.id
			JOIN deployments d ON c.depl_id = d.id
			WHERE d.ns_id = :ns_id
		), volumes_without_endpoints AS (
			SELECT storage_id FROM ns_volumes
			EXCEPT
			SELECT storage_id FROM endpoints
		), inserted_eps AS (
			INSERT INTO endpoints
			(namespace_id, storage_id, service_exists)
			SELECT :ns_id, vwe.storage_id, FALSE FROM volumes_without_endpoints vwe
			RETURNING storage_id
		)
		SELECT s.id, s.ips
		FROM storages s
		WHERE s.id IN (SELECT ie.storage_id FROM inserted_eps ie)`,
		map[string]interface{}{"ns_id": nsID})
	var storages []struct {
		ID  string         `db:"id"`
		IPs pq.StringArray `db:"ips"`
	}
	err = sqlx.SelectContext(ctx, db.extLog, &storages, db.extLog.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	for _, storage := range storages {
		endpointName := models.GlusterEndpointName(storage.ID)
		ret = append(ret, kube_api.Endpoint{
			Name:      endpointName,
			Owner:     &userID,
			Addresses: storage.IPs,
			Ports: []kube_api.Port{{
				Name:     endpointName,
				Port:     1,
				Protocol: kube_api.TCP,
			}},
		})
	}

	return
}

func (db *pgDB) ConfirmGlusterEndpoints(ctx context.Context, userID, nsLabel string) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Info("confirm gluster services created")

	nsID, err := db.getNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	if nsID == "" {
		err = rserrors.ErrResourceNotExists().Log(err, db.log)
		return
	}

	_, err = sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
		`UPDATE endpoints SET service_exists = TRUE WHERE namespace_id = :ns_id`,
		map[string]interface{}{"ns_id": nsID})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}
