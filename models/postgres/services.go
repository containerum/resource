package postgres

import (
	"context"

	"database/sql"

	"strings"

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
		(service_id, name, port, target_port, protocol)
		VALUES (:service_id, :name, :port, :target_port, :protocol)`)
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
			Protocol:   rstypes.PortProtocol(strings.ToLower(string(port.Protocol))),
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

func (db *pgDB) getRawServices(ctx context.Context, nsID string) (serviceMap map[string]kubtypes.Service, serviceIDs []string, err error) {
	db.log.WithField("ns_id", nsID).Debug("get raw services")

	dbEntries := make([]rstypes.Service, 0)
	query, args, _ := sqlx.Named( /* language=sql */
		`WITH depl_id_name AS (
			SELECT id, "name" FROM deployments WHERE ns_id = :ns_id
		)
		SELECT 
			s.id,
			(SELECT depl_id_name.name FROM depl_id_name WHERE s.deploy_id = depl_id_name.id) AS deploy_id,
			s.name,
			s.created_at
		FROM services s`,
		map[string]interface{}{"ns_id": nsID})
	err = sqlx.SelectContext(ctx, db.extLog, &dbEntries, db.extLog.Rebind(query), args...)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}

	serviceMap = make(map[string]kubtypes.Service)
	for _, v := range dbEntries {
		createdAt := v.CreatedAt.Unix()
		serviceMap[v.ID] = kubtypes.Service{
			Name:      v.Name,
			CreatedAt: &createdAt,
			Deploy:    v.DeployID,
		}
		serviceIDs = append(serviceIDs, v.ID)
	}

	return
}

func (db *pgDB) getServicesPorts(ctx context.Context, serviceIDs []string, serviceMap map[string]kubtypes.Service) (err error) {
	db.log.Debugf("get services ports %#v", serviceIDs)

	dbEntries := make([]rstypes.Port, 0)
	query, args, _ := sqlx.In( /* language=sql */
		`SELECT * FROM service_ports WHERE id IN (?)`, serviceIDs)
	err = sqlx.SelectContext(ctx, db.extLog, &dbEntries, db.extLog.Rebind(query), args...)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}

	portMap := make(map[string][]kubtypes.ServicePort)
	for _, v := range dbEntries {
		ports := portMap[v.ServiceID]
		ports = append(ports, kubtypes.ServicePort{
			Name:       v.Name,
			Port:       v.Port,
			TargetPort: v.TargetPort,
			Protocol:   kubtypes.Protocol(strings.ToUpper(string(v.Protocol))),
		})
		portMap[v.ServiceID] = ports
	}

	for _, v := range dbEntries {
		service := serviceMap[v.ServiceID]
		service.Ports = portMap[v.ServiceID]
		serviceMap[v.ServiceID] = service
	}

	return
}

func (db *pgDB) GetServices(ctx context.Context, userID, nsLabel string) (ret []kubtypes.Service, err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Debug("get services")

	nsID, err := db.getNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}
	if nsID == "" {
		err = models.ErrLabeledResourceNotExists
		return
	}

	serviceMap, serviceIDs, err := db.getRawServices(ctx, nsID)
	if err != nil {
		return
	}

	if err = db.getServicesPorts(ctx, serviceIDs, serviceMap); err != nil {
		return
	}

	ret = make([]kubtypes.Service, 0)
	for _, v := range serviceMap {
		ret = append(ret, v)
	}

	return
}
