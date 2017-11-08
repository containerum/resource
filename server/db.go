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
	"github.com/sirupsen/logrus"
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
			" VALUES ($1,$2,$3)",
		action,
		objType,
		objID,
	)
}

func (db resourceSvcDB) namespaceCreate(resourceUUID uuid.UUID, nsLabel string, cpuQuota, memQuota int) (ns Namespace, err error) {
	_, err = db.con.Exec(
		"INSERT INTO namespaces (id, resource_id, namespace_label, cpu, memory)"+
			" VALUES ($1,$2,$3,$4,$5)",
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
		"DELETE FROM namespaces WHERE id = $1",
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
			"SELECT (id,label,user_id,create_time,ram,cpu,max_ext_svc,max_int_svc,max_traffic,deleted,delete_time,tariff_id)" +
				" FROM namespaces WHERE deleted = false",
		)
	} else {
		rows, err = db.con.Query(
			"SELECT (id,label,user_id,create_time,ram,cpu,max_ext_svc,max_int_svc,max_traffic,deleted,delete_time,tariff_id)"+
				" FROM namespaces WHERE user_id = CAST($1 AS uuid) AND deleted = false",
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

func (db resourceSvcDB) namespaceGet(owner uuid.UUID, label string) (ns Namespace, err error) {
	rows, err := db.con.Query(
		"SELECT (id,label,user_id,create_time,ram,cpu,max_ext_svc,max_int_svc,max_traffic,deleted,delete_time,tariff_id)"+
			" FROM namespaces WHERE user_id = $1 AND label = $2 AND deleted = false",
		owner,
		label,
	)
	if err != nil {
		return
	}
	defer rows.Close()
	var count int
	for rows.Next() {
		count++
		if count > 1 {
			err = newError("database inconsistency found: more than 1 value for pair (user_id,label)")
			return
		}
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
	}
	if count != 1 {
		err = NoSuchResource
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

func (db resourceSvcDB) permFetch(resourceUUID, userUUID uuid.UUID) (perm string, err error) {
	var permLimited string
	var limited bool

	err = db.con.QueryRow(
		"SELECT (status_main, limited, status_limited) FROM permissions"+
			" WHERE resource_id=$1 AND user_id=$2",
		resourceUUID,
		userUUID,
	).Scan(&perm, &limited, &permLimited)
	if err != nil {
		return
	}
	if permLimited != "" && limited == false {
		perm = permLimited
	} else if permLimited != "" && limited == true {
		perm = "read"
	}
	return
}

func (db resourceSvcDB) permSetLimited(resourceUUID uuid.UUID, limited bool) error {
	_, err := db.con.Exec(
		"UPDATE permissions SET limited=$1 WHERE resource_id=$2",
		limited,
		resourceUUID,
	)
	return err
}

func (db resourceSvcDB) permSetOtherUser(resUUID, otherUserUUID uuid.UUID, perm string) error {
	var resKind string
	err := db.con.QueryRow("SELECT (kind) FROM permissions WHERE resource_id=$1 LIMIT 1", resUUID).Scan(&resKind)
	if err != nil {
		if err == sql.ErrNoRows {
			return NoSuchResource
		}
		return err
	}

	permUUID := uuid.NewV4()
	_, err = db.con.Exec(
		"INSERT INTO permissions(id, kind, resource_id, user_id, status_main, status_limited, status_change_time)"+
		" VALUES($1, $2, $3, $4, $5, $6, now())",
		permUUID,
		resKind,
		resUUID,
		otherUserUUID,
		"none",
		perm,
	)
	return err
}

func permCheck(perm, action string) bool {
	switch perm {
	case "read":
		if action == "delete" {
			return false
		}
		fallthrough
	case "readdelete":
		if action == "write" {
			return false
		}
		fallthrough
	case "write":
		if action == "owner" {
			return false
		}
		fallthrough
	case "owner":
		return true
	}
	logrus.Errorf("unreachable in db.go:/permCheck")
	return false
}
