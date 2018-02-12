package postgres

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

func (db *pgDB) isIngressExists(ctx context.Context, nsID, domain string) (exist bool, err error) {
	params := map[string]interface{}{
		"ns_id":  nsID,
		"domain": domain,
	}
	db.log.WithFields(params).Debug("check if ingress for domain exist")

	query, args, _ := sqlx.Named( /* language=sql */
		`WITH ns_services AS (
			SELECT id FROM services WHERE deploy_id IN (SELECT id FROM deployments WHERE ns_id = :ns_id)
		)
		SELECT count(*)>0 FROM ingresses WHERE service_id IN (SELECT id FROM ns_services)`,
		params)
	err = sqlx.GetContext(ctx, db.extLog, &exist, db.extLog.Rebind(query), args...)
	if err != nil {
		err = models.WrapDBError(err)
	}

	return
}

func (db *pgDB) CreateIngress(ctx context.Context, userID, nsLabel string, req rstypes.CreateIngressRequest) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Debugf("create ingress %#v", req)

	nsID, err := db.getNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}
	if nsID == "" {
		err = models.ErrLabeledResourceNotExists
		return
	}

	exists, err := db.isIngressExists(ctx, nsID, req.Domain)
	if err != nil {
		return
	}
	if exists {
		err = models.ErrIngressExists
		return
	}

	params := struct {
		NsID string `db:"ns_id"`
		rstypes.CreateIngressRequest
	}{
		NsID:                 nsID,
		CreateIngressRequest: req,
	}

	_, err = sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
		`WITH service_id_name AS (
			SELECT DISTINCT id, name FROM services WHERE deploy_id IN (SELECT id FROM deployments WHERE ns_id = :ns_id)
		)
		INSERT INTO ingresses
		(custom_domain, type, service_id)
		VALUES (:custom_domain, 
			:type,
			(SELECT id FROM service_id_name WHERE name = :service_name)
		)`,
		params)
	if err != nil {
		err = models.WrapDBError(err)
	}

	return
}
