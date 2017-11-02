package server

import (
	"database/sql"
	"os"

	_ "github.com/lib/pq"
	"github.com/mattes/migrate"
	mig_postgres "github.com/mattes/migrate/database/postgres"
	_ "github.com/mattes/migrate/source/file"
	uuid "github.com/satori/go.uuid"
)

type resourceManagerDB struct {
	con *sql.DB
}

func (db resourceManagerDB) initialize() error {
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

func (db resourceManagerDB) log(action, objType, objID string) {
	db.con.Exec(
		"INSERT INTO log (action, obj_type, obj_id)"+
			" VALUES (?,?,?)",
		action,
		objType,
		objID,
	)
}

func (db resourceManagerDB) resourceCreate(resID string, resType string, tariffID string) error {
	_, err := db.con.Exec(
		"INSERT INTO resources (?, ?, ?)",
		resID,
		resType,
		tariffID,
	)
	if err != nil {
		return err
	}
	db.log("create", "resource", resID)
	return nil
}

func (db resourceManagerDB) namespaceCreate(resourceID, nsID string, nsLabel *string, cpuQuota, memQuota *int) error {
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

func (db resourceManagerDB) namespaceDelete(nsID string) error {
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

func (db resourceManagerDB) namespaceList(userID *uuid.UUID) (nss []Namespace, err error) {
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

func (db resourceManagerDB) roleAdd(resourceID, userID, role string) error {
	db.log("create", "resource", resourceID)
	return nil
}

func (db resourceManagerDB) roleDelete(resourceID, userID, role string) error {
	db.log("create", "resource", resourceID)
	return nil
}
