package postgres

import (
	"context"

	"database/sql"

	"strings"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/cherrylog"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/pkg/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type ServicePG struct {
	models.RelationalDB
	log *cherrylog.LogrusAdapter
}

func NewServicePG(db models.RelationalDB) models.ServiceDB {
	return &ServicePG{
		RelationalDB: db,
		log:          cherrylog.NewLogrusAdapter(logrus.WithField("component", "service_pg")),
	}
}

func (db *ServicePG) createServicePorts(ctx context.Context, serviceID, domain string,
	serviceType rstypes.ServiceType, ports []kubtypes.ServicePort) (err error) {
	db.log.WithField("service_id", serviceID).Debugf("add service ports %#v", ports)

	var query string
	switch serviceType {
	case rstypes.ServiceInternal:
		query = /* language=sql */ `INSERT INTO service_ports
			(service_id, name, port, target_port, protocol, domain_id)
			VALUES (:service_id, :name, :port, :target_port, :protocol, NULL)`
	case rstypes.ServiceExternal:
		query = /* language=sql */ `INSERT INTO service_ports
			(service_id, name, port, target_port, protocol, domain_id)
			SELECT :service_id, :name, :port, :target_port, :protocol, d.id
			FROM domains d
			WHERE d.domain = :domain`
	}

	stmt, err := db.PrepareNamed(query)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
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
			Domain:     &domain,
		})
		if err != nil {
			err = rserrors.ErrDatabase().Log(err, db.log)
			return
		}
	}

	return
}

func (db *ServicePG) CreateService(ctx context.Context, userID, nsLabel string, serviceType rstypes.ServiceType, req kubtypes.Service) (err error) {
	db.log.WithFields(logrus.Fields{
		"type":     serviceType,
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Debugf("create service %#v", req)

	nsID, err := NewNamespacePG(db.RelationalDB).GetNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}

	var serviceExists bool
	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT count(s.*)>0
		FROM services s
		JOIN deployments d ON s.deploy_id = d.id AND NOT d.deleted
		WHERE (d.ns_id, s.name) = (:ns_id, :name) AND NOT s.deleted`,
		map[string]interface{}{
			"ns_id": nsID,
			"name":  req.Name,
		})
	err = sqlx.GetContext(ctx, db, &serviceExists, db.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	if serviceExists {
		err = rserrors.ErrResourceAlreadyExists().AddDetailF("service %s already exists in namespace", req.Name)
		return
	}

	var deplID string
	query, args, _ = sqlx.Named( /* language=sql */
		`SELECT id FROM deployments WHERE ns_id = :ns_id AND name = :name AND NOT deleted`,
		map[string]interface{}{"ns_id": nsID, "name": req.Deploy})
	err = sqlx.GetContext(ctx, db, &deplID, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("deployment %s not exists", req.Deploy).Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	var serviceID string
	query, args, _ = sqlx.Named( /* language=sql */
		`INSERT INTO services
		(deploy_id, name, type)
		VALUES (:deploy_id, :name, :type)
		RETURNING id`,
		map[string]interface{}{"deploy_id": deplID, "name": req.Name, "type": serviceType})
	err = sqlx.GetContext(ctx, db, &serviceID, db.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	err = db.createServicePorts(ctx, serviceID, req.Domain, serviceType, req.Ports)
	return
}

func (db *ServicePG) getRawServices(ctx context.Context, nsID string) (serviceMap map[string]kubtypes.Service, serviceIDs []string, err error) {
	db.log.WithField("ns_id", nsID).Debug("get raw services")

	dbEntries := make([]rstypes.Service, 0)
	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT 
			s.id,
			d.name AS depl_id,
			s.name,
			s.type,
			s.created_at,
			s.deleted,
			s.delete_time
		FROM services s
		JOIN deployments d ON s.deploy_id = d.id AND NOT d.deleted
		WHERE NOT s.deleted AND d.ns_id = :ns_id`,
		map[string]interface{}{"ns_id": nsID})
	err = sqlx.SelectContext(ctx, db, &dbEntries, db.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
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

func (db *ServicePG) getServicesPorts(ctx context.Context, serviceIDs []string, serviceMap map[string]kubtypes.Service) (err error) {
	db.log.Debugf("get services ports %#v", serviceIDs)

	if len(serviceIDs) == 0 {
		return nil
	}

	dbEntries := make([]rstypes.Port, 0)
	query, args, _ := sqlx.In( /* language=sql */
		`SELECT
			sp.id,
			sp.service_id,
			sp.name,
			sp.port,
			sp.target_port,
			sp.protocol,
			d.domain	
		FROM service_ports sp
		LEFT JOIN domains d ON sp.domain_id = d.id
		WHERE sp.service_id IN (?)`, serviceIDs)
	err = sqlx.SelectContext(ctx, db, &dbEntries, db.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
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

	for serviceID, v := range portMap {
		service := serviceMap[serviceID]
		service.Ports = v
		serviceMap[serviceID] = service
	}

	return
}

func (db *ServicePG) getServicesDomains(ctx context.Context, serviceIDs []string, serviceMap map[string]kubtypes.Service) (err error) {
	db.log.Debugf("get services domains %#v", serviceIDs)

	if len(serviceIDs) == 0 {
		return nil
	}

	var entries []struct {
		Domain    string         `db:"domain"`
		IPs       pq.StringArray `db:"ips"`
		ServiceID string         `db:"service_id"`
	}
	query, args, _ := sqlx.In( /* language=sql */
		`SELECT
		d.domain,
		d.ip AS ips,
		s.id AS service_id
		FROM domains d
		JOIN service_ports sp ON sp.domain_id = d.id
		JOIN services s ON sp.service_id = s.id AND s.type = 'external'
		WHERE s.id IN (?)`, serviceIDs)
	err = sqlx.SelectContext(ctx, db, &entries, db.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	for _, v := range entries {
		service := serviceMap[v.ServiceID]
		service.Domain = v.Domain
		service.IPs = v.IPs
		serviceMap[v.ServiceID] = service
	}

	return
}

func (db *ServicePG) GetServices(ctx context.Context, userID, nsLabel string) (ret []kubtypes.Service, err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Debug("get services")

	nsID, err := NewNamespacePG(db.RelationalDB).GetNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}

	serviceMap, serviceIDs, err := db.getRawServices(ctx, nsID)
	if err != nil {
		return
	}

	if err = db.getServicesPorts(ctx, serviceIDs, serviceMap); err != nil {
		return
	}

	if err = db.getServicesDomains(ctx, serviceIDs, serviceMap); err != nil {
		return
	}

	ret = make([]kubtypes.Service, 0)
	for _, v := range serviceMap {
		ret = append(ret, v)
	}

	return
}

func (db *ServicePG) GetService(ctx context.Context, userID, nsLabel, serviceName string) (ret kubtypes.Service, stype rstypes.ServiceType, err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"service_name": serviceName,
	}).Debug("get service")

	nsID, err := NewNamespacePG(db.RelationalDB).GetNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}

	var serviceEntry rstypes.Service
	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT 
			s.id,
			d.name AS depl_id,
			s.name,
			s.type,
			s.created_at,
			s.deleted,
			s.delete_time
		FROM services s
		JOIN deployments d ON s.deploy_id = d.id AND NOT d.deleted
		WHERE s.name = :name AND NOT s.deleted`,
		map[string]interface{}{"ns_id": nsID, "name": serviceName})
	err = sqlx.GetContext(ctx, db, &serviceEntry, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("service %s not exists", serviceName).Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	stype = serviceEntry.Type
	serviceIDs := []string{serviceEntry.ID}
	createdAt := serviceEntry.CreatedAt.Unix()
	serviceMap := map[string]kubtypes.Service{
		serviceEntry.ID: {
			Name:      serviceEntry.Name,
			CreatedAt: &createdAt,
			Deploy:    serviceEntry.DeployID,
		},
	}

	if err = db.getServicesPorts(ctx, serviceIDs, serviceMap); err != nil {
		return
	}

	if err = db.getServicesDomains(ctx, serviceIDs, serviceMap); err != nil {
		return
	}

	ret = serviceMap[serviceEntry.ID]
	return
}

func (db *ServicePG) UpdateService(ctx context.Context, userID, nsLabel string, newServiceType rstypes.ServiceType, req kubtypes.Service) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":          userID,
		"ns_label":         nsLabel,
		"service_name":     req.Name,
		"new_service_type": newServiceType,
	}).Debugf("update service to %#v", req)

	nsID, err := NewNamespacePG(db.RelationalDB).GetNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}

	var serviceID string
	query, args, _ := sqlx.Named( /* language=sql */
		`WITH service_to_update AS (
			SELECT s.id
			FROM services s
			JOIN deployments d ON s.deploy_id = d.id AND NOT d.deleted
			WHERE d.ns_id = :ns_id AND s.name = :name AND NOT s.deleted
		)
		UPDATE services
		SET "type" = :new_type
		WHERE id = (SELECT id FROM service_to_update)
		RETURNING id`,
		map[string]interface{}{"ns_id": nsID, "name": req.Name, "new_type": newServiceType})
	err = sqlx.GetContext(ctx, db, &serviceID, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrResourceNotExists().AddDetailF("service %s not exists", req.Name).Log(err, db.log)
		return
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	_, err = sqlx.NamedExecContext(ctx, db, /* language=sql */
		`DELETE FROM service_ports WHERE service_id = :service_id`,
		map[string]interface{}{"service_id": serviceID})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	err = db.createServicePorts(ctx, serviceID, req.Domain, newServiceType, req.Ports)
	return
}

func (db *ServicePG) DeleteService(ctx context.Context, userID, nsLabel, serviceName string) (err error) {
	db.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_label":     nsLabel,
		"service_name": serviceName,
	}).Debug("delete service")

	nsID, err := NewNamespacePG(db.RelationalDB).GetNamespaceID(ctx, userID, nsLabel)
	if err != nil {
		return
	}

	result, err := sqlx.NamedExecContext(ctx, db, /* language=sql */
		`WITH service_to_update AS (
			SELECT s.id
			FROM services s
			JOIN deployments d ON s.deploy_id = d.id
			WHERE d.ns_id = :ns_id AND s.name = :name
		)
		UPDATE services
		SET deleted = TRUE, delete_time = now()
		WHERE id = (SELECT id FROM service_to_update)`,
		map[string]interface{}{"ns_id": nsID, "name": serviceName})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	if count, _ := result.RowsAffected(); count <= 0 {
		err = rserrors.ErrResourceNotExists().AddDetailF("service %s not exists", serviceName).Log(err, db.log)
	}

	return
}
