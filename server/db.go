package server

import (
	"database/sql"
	"os"
	"fmt"
	
	"github.com/mattes/migrate"
	mig_postgres "github.com/mattes/migrate/database/postgres"
)

/*
type Resource struct {
	ID string
	Type string
}

type Namespace struct {
	ID string
	Label string
	Resource *Resource

	CpuLim uint64
	MemLim uint64
}

type Volume struct {
	ID string
	Label string
	Resource *Resource
}

type Access struct {
	Resource *Resource
	UserID string
	AccessLevel string
}
*/
/*
CREATE TABLE resources (
	resource varchar UNIQUE PRIMARY KEY,
	resource_type varchar NOT NULL,
	tariff_id varchar NOT NULL
);

CREATE TABLE namespaces (
	namespace_id varchar UNIQUE PRIMARY KEY,
	resource_id varchar NOT NULL REFERENCES resources,
	namespace_label varchar NOT NULL,
	cpu int NOT NULL,
	memory int NOT NULL
);

CREATE TABLE volumes (
	volume_id varchar UNIQUE PRIMARY KEY,
	resource_id varchar NOT NULL REFERENCES resources,
	volume_label varchar NOT NULL,
	size int NOT NULL
);

CREATE TABLE accesses (
	user_id varchar NOT NULL,
	resource_id varchar REFERENCES resources,
	access varchar NOT NULL
);

CREATE TABLE log (
	t timestamp NOT NULL DEFAULT statement_timestamp(),
	action varchar NOT NULL,
	obj_type varchar NOT NULL,
	obj_id varchar NOT NULL
);
*/

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
			return fmt.Errorf("cannot run migration: %v", err)
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
		"DELETE FROM namespaces WHERE namespace_id = ?",
		nsID,
	)
	if err != nil {
		return err
	}
	db.log("delete", "namespace", nsID)
	return nil
}

func (db resourceManagerDB) roleAdd(resourceID, userID, role string) error {
	db.log("create", "resource", resourceID)
	return nil
}

func (db resourceManagerDB) roleDelete(resourceID, userID, role string) error {
	db.log("create", "resource", resourceID)
	return nil
}
