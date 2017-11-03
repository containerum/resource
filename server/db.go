package server

import (
	"database/sql"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/mattes/migrate"
	mig_postgres "github.com/mattes/migrate/database/postgres"
	_ "github.com/mattes/migrate/source/file"
	uuid "github.com/satori/go.uuid"
)

type resourceSvcDB struct {
	con *sql.DB
}

func (db resourceSvcDB) initialize() error {
	err := db.con.Ping()
	if err != nil {
		return err
	}

	inst, err := mig_postgres.WithInstance(db.con, &mig_postgres.Config{})
	if err != nil {
		return newError("what the fuck is this: %v", err)
	}
	mig, err := migrate.NewWithDatabaseInstance(os.Getenv("MIGRATION_DIR"), "postgres", inst)
	if err != nil {
		return newError("cannot create migration: %v", err)
	}
	if err = mig.Up(); err != nil {
		if err != migrate.ErrNoChange {
			return newError("cannot run migration: %v", err)
		}
	}
	return nil
}

func (db resourceSvcDB) log(action, objType, objID string) {
	db.con.Exec(
		"INSERT INTO log (action, obj_type, obj_id)"+
			" VALUES (?,?,?)",
		action,
		objType,
		objID,
	)
}

func (db resourceSvcDB) namespaceCreate(resourceID, nsID string, nsLabel *string, cpuQuota, memQuota *int) error {
	_, err := db.con.Exec(
		"INSERT INTO namespaces (namespace_id, resource_id, namespace_label, cpu, memory)"+
			" VALUES (?,?,?,?,?)",
		nsID,
		resourceID,
		nsLabel,
		cpuQuota,
		memQuota,
	)
	if err != nil {
		return err
	}
	db.log("create", "namespace", nsID)
	return nil
}

func (db resourceSvcDB) namespaceDelete(nsID string) error {
	_, err := db.con.Exec(
		"DELETE FROM namespaces WHERE id = ?",
		nsID,
	)
	if err != nil {
		return err
	}
	db.log("delete", "namespace", nsID)
	return nil
}

func (db resourceSvcDB) namespaceList(userID *uuid.UUID) (nss []Namespace, err error) {
	var rows *sql.Rows
	if userID == nil {
		rows, err = db.con.Query(
			"SELECT (id, label, user_id, create_time, ram, cpu, max_ext_svc, max_int_svc, max_traffic, deleted, delete_time, tariff_id)"+
				" FROM namespaces",
		)
	} else {
		rows, err = db.con.Query(
			"SELECT (id, label, user_id, create_time, ram, cpu, max_ext_svc, max_int_svc, max_traffic, deleted, delete_time, tariff_id)"+
				" FROM namespaces WHERE user_id = CAST($1 AS uuid)",
			*userID,
		)
	}
	if err != nil {
		return
	}
	defer rows.Close()
	nss = make([]Namespace, 0)
	for rows.Next() {
		var ns Namespace
		err = rows.Scan(
			&ns.ID,
			&ns.Label,
			&ns.UserID,
			&ns.CreateTime,
			&ns.RAM,
			&ns.CPU,
			&ns.MaxExtService,
			&ns.MaxIntService,
			&ns.MaxTraffic,
			&ns.Deleted,
			&ns.DeleteTime,
			&ns.TariffID,
		)
		if err != nil {
			return
		}
		nss = append(nss, ns)
	}
	return
}

func (db resourceSvcDB) permCreate(resourceKind string, resourceUUID, ownerUserUUID uuid.UUID) error {
	permUUID := uuid.NewV4()
	_, err := db.con.Exec(
		"INSERT INTO permissions(id, kind, resource_id, user_id, status_main, limited, status_change_time)"+
			" VALUES($1, $2, $3, $4, $5, $6, $7)",
		permUUID,
		resourceKind,
		resourceUUID,
		ownerUserUUID,
		"owner",
		false,
		time.Now(),
	)
	if err != nil {
		return err
	}
	db.log("create", "permission", permUUID.String())
	return nil
}

func (db resourceSvcDB) permCheck(resourceUUID, userUUID uuid.UUID, perm string) error {
	return nil
}

func (db resourceSvcDB) permSetLimited(limited bool) error {
	return nil
}
