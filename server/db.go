package server

import (
	"database/sql"
	"os"
	"time"

	"bitbucket.org/exonch/resource-service/server/model"

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

func (db resourceSvcDB) namespaceCreate(resID uuid.UUID, nsLabel string, tariff model.NamespaceTariff) error {
	_, err := db.con.Exec(
		"INSERT INTO namespaces (id, label, ram, cpu, max_ext_svc, max_int_svc, max_traffic, tariff_id)"+
			" VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
		resID,
		nsLabel,
		tariff.MemoryLimit,
		tariff.CpuLimit,
		tariff.ExternalServices,
		tariff.InternalServices,
		tariff.Traffic,
		tariff.TariffID,
	)
	if err != nil {
		return err
	}
	db.log("create", "namespace", resID.String())
	return nil
}

func (db resourceSvcDB) namespaceDelete(nsID uuid.UUID) error {
	_, err := db.con.Exec(
		"DELETE FROM namespaces WHERE id = $1",
		nsID,
	)
	if err != nil {
		return err
	}
	db.log("delete", "namespace", nsID.String())
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
				" FROM namespaces WHERE user_id=$1 AND deleted = false",
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
	err = db.con.QueryRow(
		"SELECT (id,label,user_id,create_time,ram,cpu,max_ext_svc,max_int_svc,"+
			"max_traffic,deleted,delete_time,tariff_id)"+
			" FROM namespaces WHERE user_id = $1 AND label = $2 AND deleted = false",
		owner,
		label,
	).Scan(
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
		if err == sql.ErrNoRows {
			err = NoSuchResource
		}
	}
	return
}

func (db resourceSvcDB) permCreate(resourceKind string, resourceUUID, ownerUserUUID uuid.UUID) error {
	permUUID := uuid.NewV4()
	_, err := db.con.Exec(
		"INSERT INTO permissions(id, kind, resource_id, user_id, owner_user_id, status, limited, status_change_time)"+
			" VALUES($1, $2, $3, $4, $5, $6, $7, $8)",
		permUUID,
		resourceKind,
		resourceUUID,
		ownerUserUUID,
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
	var limited bool

	err = db.con.QueryRow(
		"SELECT (status, limited) FROM permissions"+
			" WHERE resource_id=$1 AND user_id=$2",
		resourceUUID,
		userUUID,
	).Scan(&perm, &limited)
	if err != nil {
		if err == sql.ErrNoRows {
			err = NoSuchResource
		}
		return
	}
	if limited {
		perm = "none"
	}
	return
}

func (db resourceSvcDB) permGrant(resID, ownerID, otherUserID uuid.UUID, perm string) error {
	var resKind string
	err := db.con.QueryRow(
		"SELECT kind FROM permissions WHERE resource_id=$1 AND user_id=$2",
		resID,
		ownerID,
	).Scan(&resKind)
	if err != nil {
		if err == sql.ErrNoRows {
			err = NoSuchResource
		}
		return err
	}

	permID := uuid.NewV4()
	_, err = db.con.Exec(
		"INSERT INTO permissions(id, kind, resource_id, user_id, owner_user_id, status, status_change_time)"+
			" VALUES($1,$2,$3,$4,$5,$6,$7)",
		permID,
		resKind,
		resID,
		ownerID,
		otherUserID,
		perm,
		time.Now(),
	)
	if err != nil {
		return err
	}

	db.log("grant", "permission", permID.String())
	return nil
}

func (db resourceSvcDB) permRevoke(resID, otherUserID uuid.UUID) error {
	var permID uuid.UUID
	err := db.con.QueryRow(
		"SELECT id FROM permissions WHERE resource_id=$1, user_id=$2",
		resID,
		otherUserID,
	).Scan(&permID)
	if err != nil {
		if err == sql.ErrNoRows {
			err = NoSuchResource
		}
		return err
	}

	_, err = db.con.Exec(
		"DELETE FROM permissions WHERE resource_id=$1, user_id=$2",
		resID,
		otherUserID,
	)
	if err != nil {
		return err
	}
	db.log("revoke", "permission", permID.String())
	return nil
}

func (db resourceSvcDB) permSetLimited(resourceUUID uuid.UUID, limited bool) error {
	_, err := db.con.Exec(
		"UPDATE permissions SET limited=$1 WHERE resource_id=$2 AND user_id=owner_user_id",
		limited,
		resourceUUID,
	)
	return err
}

func permCheck(perm, needed string) bool {
	switch perm {
	case "read":
		if needed == "delete" {
			return false
		}
		fallthrough
	case "readdelete":
		if needed == "write" {
			return false
		}
		fallthrough
	case "write":
		if needed == "owner" {
			return false
		}
		fallthrough
	case "owner":
		return true
	}
	logrus.Errorf("unreachable in db.go:/permCheck")
	return false
}
