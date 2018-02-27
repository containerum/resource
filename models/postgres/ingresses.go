package postgres

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
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
		`SELECT count(i.*)>0 
		FROM ingresses i 
		JOIN services s ON i.service_id = s.id
		JOIN deployments d ON s.deploy_id = d.id
		WHERE d.ns_id = :ns_id`,
		params)
	err = sqlx.GetContext(ctx, db.extLog, &exist, db.extLog.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
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
		err = rserrors.ErrResourceNotExists().Log(err, db.log)
		return
	}

	exists, err := db.isIngressExists(ctx, nsID, req.Domain)
	if err != nil {
		return
	}
	if exists {
		err = rserrors.ErrResourceAlreadyExists().Log(err, db.log)
		return
	}

	_, err = sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
		`WITH service_id_name AS (
			SELECT DISTINCT id, name FROM services WHERE deploy_id IN (SELECT id FROM deployments WHERE ns_id = :ns_id)
		)
		INSERT INTO ingresses
		(custom_domain, type, service_id)
		VALUES (:custom_domain, 
			:type,
			(SELECT id FROM service_id_name WHERE name = :service)
		)`,
		map[string]interface{}{"ns_id": nsID, "custom_domain": req.Domain, "type": req.Type, "service": req.Service})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}

func (db *pgDB) GetUserIngresses(ctx context.Context, userID, nsLabel string, params rstypes.GetIngressesQueryParams) (ret []rstypes.Ingress, err error) {
	db.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Debug("get all ingresses")

	nsID, err := db.getNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}
	if nsID == "" {
		err = rserrors.ErrResourceNotExists().Log(err, db.log)
		return
	}

	ret = make([]rstypes.Ingress, 0)
	entries := make([]rstypes.IngressEntry, 0)
	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT 
			i.id,
			i.custom_domain,
			i.type,
			s.name AS service_id, --hack to inject service name instead of id
			i.created_at
		FROM ingresses i 
		JOIN services s ON i.service_id = s.id
		JOIN deployments d ON s.deploy_id = d.id
		WHERE d.ns_id = :ns_id
		LIMIT :limit 
		OFFSET :offset`,
		map[string]interface{}{
			"ns_id":  nsID,
			"limit":  params.PerPage,
			"offset": params.PerPage * (params.Page - 1),
		})
	err = sqlx.SelectContext(ctx, db.extLog, &entries, db.extLog.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	for _, v := range entries {
		ret = append(ret, rstypes.Ingress{
			Domain:    v.Domain,
			Type:      v.Type,
			Service:   v.ServiceID,
			CreatedAt: &v.CreatedAt,
		})
	}

	return
}

func (db *pgDB) GetAllIngresses(ctx context.Context, params rstypes.GetIngressesQueryParams) (ret []rstypes.Ingress, err error) {
	db.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
	}).Debug("get all ingresses")

	ret = make([]rstypes.Ingress, 0)
	entries := make([]rstypes.IngressEntry, 0)
	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT
			i.id,
			i.custom_domain,
			i.type,
			s.name AS service_id,
			i.created_at
		FROM ingresses i
		JOIN services s on i.service_id = s.id
		LIMIT :limit OFFSET :offset`,
		map[string]interface{}{"limit": params.PerPage, "offset": params.PerPage * (params.Page - 1)})
	err = sqlx.SelectContext(ctx, db.extLog, &entries, db.extLog.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	for _, v := range entries {
		ret = append(ret, rstypes.Ingress{
			Domain:    v.Domain,
			Type:      v.Type,
			Service:   v.ServiceID, // name here
			CreatedAt: &v.CreatedAt,
		})
	}

	return
}

func (db *pgDB) DeleteIngress(ctx context.Context, userID, nsLabel, domain string) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
		"domain":   domain,
	}).Info("delete ingress")

	nsID, err := db.getNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}
	if nsID == "" {
		err = rserrors.ErrResourceNotExists().Log(err, db.log)
		return
	}

	result, err := sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
		`WITH ns_services AS (
			SELECT s.id 
			FROM services s
			JOIN deployments d ON s.deploy_id = d.id
			WHERE d.ns_id = :ns_id
		)
		DELETE FROM ingresses
		WHERE service_id IN (SELECT id FROM ns_services) AND custom_domain = :domain`,
		map[string]interface{}{"ns_id": nsID, "domain": domain})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	if count, _ := result.RowsAffected(); count <= 0 {
		err = rserrors.ErrResourceNotExists().Log(err, db.log)
	}

	return
}
