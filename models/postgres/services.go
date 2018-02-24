package postgres

import (
	"context"

	"database/sql"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/models"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

func (db *pgDB) createServicePorts(ctx context.Context, serviceID string, ports []kubtypes.ServicePort) (err error) {
	db.log.WithField("service_id", serviceID).Debugf("add service ports %#v", ports)

	stmt, err := db.preparer.PrepareNamed( /* language=sql */
		`INSERT INTO service_ports
		(service_id, name, port, target_port)
		VALUES (:service_id, :name, :port, :target_port)`)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	defer stmt.Close()

	for _, port := range ports {
		_, err = stmt.ExecContext(ctx, rstypes.Port{
			ServiceID:  serviceID,
			Name:       port.Name,
			Port:       port.Port,
			TargetPort: port.TargetPort,
		})
		if err != nil {
			err = models.WrapDBError(err)
			return
		}
	}

	return
}

func (db *pgDB) CreateService(ctx context.Context, userID, nsLabel, serviceType string, req kubtypes.Service) (err error) {
	db.log.WithFields(logrus.Fields{
		"type":     serviceType,
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Debugf("create service %#v", req)

	nsID, err := db.getNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}
	if nsID == "" {
		err = models.ErrLabeledResourceNotExists
		return
	}

	var deplID string
	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT id FROM deployments WHERE ns_id = :ns_id AND name = :name`,
		map[string]interface{}{"ns_id": nsID, "name": req.Deploy})
	err = sqlx.GetContext(ctx, db.extLog, &deplID, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = models.ErrLabeledResourceNotExists
		return
	default:
		err = models.WrapDBError(err)
		return
	}

	var serviceID string
	query, args, _ = sqlx.Named( /* language=sql */
		`INSERT INTO services
		(deploy_id, name, type)
		VALUES (:deploy_id, :name, :type)
		RETURNING id`,
		map[string]interface{}{"deploy_id": deplID, "name": req.Name, "type": serviceType})
	err = sqlx.GetContext(ctx, db.extLog, &serviceID, db.extLog.Rebind(query), args...)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}

	err = db.createServicePorts(ctx, serviceID, req.Ports)
	return
}
