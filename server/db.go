package server

import (
	"database/sql"
	"fmt"
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

//type resourceSvcDBOld struct {
//	con *sql.DB
//}
//
//func (db resourceSvcDBOld) initialize() error {
//	err := db.con.Ping()
//	if err != nil {
//		return err
//	}
//
//	inst, err := mig_postgres.WithInstance(db.con, &mig_postgres.Config{})
//	if err != nil {
//		return newError("what the fuck is this: %v", err)
//	}
//	mig, err := migrate.NewWithDatabaseInstance(os.Getenv("MIGRATION_URL"), "postgres", inst)
//	if err != nil {
//		return newError("cannot create migration: %v", err)
//	}
//	if err = mig.Up(); err != nil {
//		if err != migrate.ErrNoChange {
//			return newError("cannot run migration: %v", err)
//		}
//	}
//	return nil
//}
//
//func (db resourceSvcDBOld) log(action, objType, objID string) {
//	db.con.Exec(
//		"INSERT INTO log (action, obj_type, obj_id)"+
//			" VALUES ($1,$2,$3)",
//		action,
//		objType,
//		objID,
//	)
//}
//
//func (db resourceSvcDBOld) namespaceCreate(tariff model.NamespaceTariff) (nsUUID uuid.UUID, err error) {
//	nsUUID = uuid.NewV4()
//	_, err = db.con.Exec(
//		"INSERT INTO namespaces (id,ram,cpu,max_ext_svc,max_int_svc,max_traffic,tariff_id)"+
//			" VALUES ($1,$2,$3,$4,$5,$6,$7)",
//		nsUUID,
//		tariff.MemoryLimit,
//		tariff.CpuLimit,
//		tariff.ExternalServices,
//		tariff.InternalServices,
//		tariff.Traffic,
//		tariff.TariffID,
//	)
//	if err != nil {
//		nsUUID = uuid.Nil
//		return
//	}
//	db.log("create", "namespace", nsUUID.String())
//
//	return
//}
//
//func (db resourceSvcDBOld) namespaceDelete(nsID uuid.UUID) (err error) {
//	_, err = db.con.Exec(
//		`UPDATE namespaces SET deleted=true, delete_time=statement_timestamp() WHERE id=$1`,
//		nsID,
//	)
//	if err != nil {
//		return
//	}
//	db.log("delete", "namespace", nsID.String())
//	return
//}
//
//func (db resourceSvcDBOld) namespaceList(userID *uuid.UUID) (nss []Namespace, err error) {
//	var rows *sql.Rows
//	if userID == nil {
//		rows, err = db.con.Query(
//			`SELECT
//				n.id,
//				n.create_time,
//				NULL,
//				n.ram,
//				n.cpu,
//				n.max_ext_svc,
//				n.max_int_svc,
//				n.max_traffic,
//				n.deleted,
//				n.delete_time,
//				n.tariff_id
//			FROM namespaces n WHERE deleted = false`,
//		)
//	} else {
//		rows, err = db.con.Query(
//			`SELECT
//				n.id,
//				n.create_time,
//				a.resource_label,
//				n.ram,
//				n.cpu,
//				n.max_ext_svc,
//				n.max_int_svc,
//				n.max_traffic,
//				n.deleted,
//				n.delete_time,
//				n.tariff_id
//			FROM namespaces n INNER JOIN accesses a ON a.resource_id=n.id
//			WHERE a.user_id=$1 AND n.deleted=false`,
//			*userID,
//		)
//	}
//	if err != nil {
//		return
//	}
//	defer rows.Close()
//	nss = make([]Namespace, 0)
//	for rows.Next() {
//		var ns Namespace
//		err = rows.Scan(
//			&ns.ID,
//			&ns.CreateTime,
//			&ns.Label,
//			&ns.RAM,
//			&ns.CPU,
//			&ns.MaxExtService,
//			&ns.MaxIntService,
//			&ns.MaxTraffic,
//			&ns.Deleted,
//			&ns.DeleteTime,
//			&ns.TariffID,
//		)
//		if err != nil {
//			return
//		}
//		nss = append(nss, ns)
//	}
//	return
//}
//
//func (db resourceSvcDBOld) namespaceGetByID(nsID uuid.UUID) (ns Namespace, err error) {
//	err = db.con.QueryRow(
//		`SELECT
//			id,
//			create_time,
//			ram,
//			cpu,
//			max_ext_svc,
//			max_int_svc,
//			max_traffic,
//			deleted,
//			delete_time,
//			tariff_id
//		FROM namespaces
//		WHERE id=$1 AND deleted=false`,
//		nsID,
//	).Scan(
//		&ns.ID,
//		&ns.CreateTime,
//		&ns.RAM,
//		&ns.CPU,
//		&ns.MaxExtService,
//		&ns.MaxIntService,
//		&ns.MaxTraffic,
//		&ns.Deleted,
//		&ns.DeleteTime,
//		&ns.TariffID,
//	)
//	if err == sql.ErrNoRows {
//		err = ErrNoSuchResource
//	}
//	return
//}
//
//func (db resourceSvcDBOld) namespaceGet(userID uuid.UUID, label string) (ns Namespace, err error) {
//	err = db.con.QueryRow(
//		`SELECT
//			n.id,
//			n.create_time,
//			n.ram,
//			n.cpu,
//			n.max_ext_svc,
//			n.max_int_svc,
//			n.max_traffic,
//			n.deleted,
//			n.delete_time,
//			n.tariff_id
//		FROM namespaces n INNER JOIN accesses a ON a.resource_id=n.id
//		WHERE a.user_id=$1 AND a.resource_label=$2 AND a.kind='Namespace'
//			AND n.deleted=false`,
//		userID,
//		label,
//	).Scan(
//		&ns.ID,
//		&ns.CreateTime,
//		&ns.RAM,
//		&ns.CPU,
//		&ns.MaxExtService,
//		&ns.MaxIntService,
//		&ns.MaxTraffic,
//		&ns.Deleted,
//		&ns.DeleteTime,
//		&ns.TariffID,
//	)
//	if err == sql.ErrNoRows {
//		err = ErrNoSuchResource
//	}
//	ns.Label = new(string)
//	*ns.Label = label
//	return
//}
//
//func (db resourceSvcDBOld) permCreateOwner(resKind string, resUUID uuid.UUID, resLabel string, ownerUUID uuid.UUID) (permUUID uuid.UUID, err error) {
//	permUUID = uuid.NewV4()
//	_, err = db.con.Exec(
//		`INSERT INTO accesses(
//			id,
//			kind,
//			resource_id,
//			resource_label,
//			user_id,
//			owner_user_id,
//			access_level,
//			access_level_change_time,
//			limited)
//		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
//		permUUID,
//		resKind,
//		resUUID,
//		resLabel,
//		ownerUUID,
//		ownerUUID,
//		"owner",
//		time.Now(),
//		false,
//	)
//	if err != nil {
//		return
//	}
//	db.log("create", "access", permUUID.String())
//	return
//}
//
//func (db resourceSvcDBOld) permGet(userID uuid.UUID, resKind, resLabel string) (resID, permID uuid.UUID, lvl string, err error) {
//	var limited bool
//	var ownerID uuid.UUID
//
//	defer func() {
//		if err == sql.ErrNoRows {
//			err = ErrNoSuchResource
//		}
//		if err != nil {
//			resID = uuid.Nil
//			lvl = ""
//		}
//	}()
//
//	err = db.con.QueryRow(
//		"SELECT id, resource_id, access_level, owner_user_id FROM accesses"+
//			" WHERE kind=$1 AND resource_label=$2 AND user_id=$3",
//		resKind,
//		resLabel,
//		userID,
//	).Scan(&permID, &resID, &lvl, &ownerID)
//	if err != nil {
//		return
//	}
//
//	err = db.con.QueryRow(
//		"SELECT limited FROM accesses WHERE user_id=$1 AND resource_id=$2 AND limited IS NOT NULL",
//		ownerID,
//		resID,
//	).Scan(&limited)
//	if err != nil {
//		return
//	}
//
//	if limited && ownerID != userID {
//		lvl = "none"
//	}
//	return
//}
//
//func (db resourceSvcDBOld) permGetByResourceID(resID, userID uuid.UUID) (resKind, resLabel string, permID uuid.UUID, lvl string, err error) {
//	var ownerID uuid.UUID
//
//	err = db.con.QueryRow(
//		`SELECT id, kind, resource_label, access_level, owner_user_id
//		FROM accesses
//		WHERE resource_id=$1 AND user_id=$2`,
//		resID,
//		userID,
//	).Scan(&permID, &resKind, &resLabel, &lvl, &ownerID)
//	if err != nil {
//		if err == sql.ErrNoRows {
//			err = newError("no such access level record")
//		}
//		return
//	}
//
//	var limited bool
//	err = db.con.QueryRow(
//		`SELECT limited FROM accesses WHERE resource_id=$1 AND user_id=$2`,
//		resID,
//		ownerID,
//	).Scan(&limited)
//	if err != nil {
//		return
//	}
//	if limited && ownerID != userID {
//		lvl = "none"
//	}
//	return
//}
//
//func (db resourceSvcDBOld) permGrant(resID uuid.UUID, resLabel string, ownerID, otherUserID uuid.UUID, perm string) (err error) {
//	var resKind string
//
//	defer func() {
//		if err == sql.ErrNoRows {
//			err = ErrNoSuchResource
//		}
//	}()
//
//	// Obtain resource kind.
//	err = db.con.QueryRow(
//		"SELECT kind FROM permissions WHERE resource_id=$1 AND user_id=$2",
//		resID,
//		ownerID,
//	).Scan(&resKind)
//	if err != nil {
//		return err
//	}
//
//	// Check that ownerID owns resID.
//	var ownerAccLevel string
//	err = db.con.QueryRow(
//		"SELECT access_level FROM accesses"+
//			" WHERE resource_id=$1 AND user_id=$2 AND owner_user_id=$2",
//		resID,
//		ownerID,
//	).Scan(&ownerAccLevel)
//	if err != nil && err != sql.ErrNoRows {
//		return
//	}
//	if ownerAccLevel != "owner" {
//		if err == sql.ErrNoRows {
//			err = fmt.Errorf("user %s does not own resource %s")
//		}
//		return
//	}
//
//	permID := uuid.NewV4()
//	_, err = db.con.Exec(
//		"INSERT INTO accesses(id,kind,resource_id,resource_label,user_id,owner_user_id,"+
//			"access_level,access_level_change_time)"+
//			" VALUES($1,$2,$3,$4,$5,$6,$7,$8)",
//		permID,
//		resKind,
//		resID,
//		resLabel,
//		otherUserID,
//		ownerID,
//		perm,
//		time.Now(),
//	)
//	if err != nil {
//		return err
//	}
//
//	db.log("grant", "access", permID.String())
//	return nil
//}
//
//func (db resourceSvcDBOld) permSetLevel(permID uuid.UUID, lvl string) (err error) {
//	_, err = db.con.Exec(
//		`UPDATE accesses SET access_level=$1 WHERE id=$2`,
//		lvl,
//		permID,
//	)
//	if err != nil {
//		return
//	}
//
//	db.log("setlevel", "access", permID.String())
//	return nil
//}
//
//func (db resourceSvcDBOld) permSetLimited(permID uuid.UUID, limited bool) (err error) {
//	_, err = db.con.Exec(
//		"UPDATE accesses SET limited=$1 WHERE id=$2",
//		limited,
//		permID,
//	)
//	if err != nil {
//		return
//	}
//
//	db.log("setlimited", "access", permID.String())
//	return
//}
//
//func (db resourceSvcDBOld) permDelete(permID uuid.UUID) (err error) {
//	_, err = db.con.Exec(
//		`DELETE FROM accesses WHERE id=$1`,
//		permID,
//	)
//	if err != nil {
//		return
//	}
//
//	db.log("delete", "access", permID.String())
//	return
//}

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
	logrus.Errorf("unreachable code in db.go:/^func permCheck")
	return false
}

//func (db resourceSvcDBOld) volumeCreate(tariff model.VolumeTariff) (volumeID uuid.UUID, err error) {
//	volumeID = uuid.NewV4()
//	_, err = db.con.Exec(
//		`INSERT INTO volumes(id,tariff_id,capacity,replicas)
//		VALUES ($1,$2,$3,$4)`,
//		volumeID,
//		tariff.TariffID,
//		tariff.StorageLimit,
//		tariff.ReplicasLimit,
//	)
//	if err != nil {
//		return
//	}
//
//	db.log("create", "volume", volumeID.String())
//	return
//}
//
//func (db resourceSvcDBOld) volumeDelete(volumeID uuid.UUID) (err error) {
//	_, err = db.con.Exec(
//		`UPDATE volumes SET deleted=true, delete_time=statement_timestamp() WHERE id=$1`,
//		volumeID,
//	)
//	if err != nil {
//		return
//	}
//	db.log("delete", "volume", volumeID.String())
//	return
//}
//
//func (db resourceSvcDBOld) volumeList(userID *uuid.UUID) (vols []Volume, err error) {
//	var rows *sql.Rows
//	if userID == nil {
//		rows, err = db.con.Query(
//			`SELECT
//				id,
//				create_time,
//				tariff_id,
//				delete_time,
//				capacity,
//				replicas,
//				NULL
//			FROM volumes`,
//		)
//	} else {
//		rows, err = db.con.Query(
//			`SELECT
//				v.id,
//				v.create_time,
//				v.tariff_id,
//				v.delete_time,
//				v.capacity,
//				v.replicas,
//				a.resource_label
//			FROM volumes v INNER JOIN accesses a ON a.resource_id=v.id
//			WHERE a.user_id=$1 AND a.kind='Volume'`,
//			*userID,
//		)
//	}
//	if err != nil {
//		return
//	}
//	defer rows.Close()
//	vols = make([]Volume, 0)
//	for rows.Next() {
//		var v Volume
//		err = rows.Scan(
//			&v.ID,
//			&v.CreateTime,
//			&v.TariffID,
//			&v.DeleteTime,
//			&v.Storage,
//			&v.Replicas,
//			&v.Label,
//		)
//		if err != nil {
//			return
//		}
//		vols = append(vols, v)
//	}
//	return
//}
//
//func (db resourceSvcDBOld) volumeGet(userID uuid.UUID, label string) (v Volume, err error) {
//	err = db.con.QueryRow(
//		`SELECT
//			v.id,
//			v.create_time,
//			v.tariff_id,
//			v.capacity,
//			v.replicas,
//			a.resource_label
//		FROM volumes v INNER JOIN accesses a ON a.resource_id=v.id
//		WHERE a.user_id=$1 AND v.deleted=false AND a.kind='Volume'`,
//		userID,
//		label,
//	).Scan(
//		&v.ID,
//		&v.CreateTime,
//		&v.TariffID,
//		&v.Storage,
//		&v.Replicas,
//		&v.Label,
//	)
//	if err == sql.ErrNoRows {
//		err = ErrNoSuchResource
//	}
//	return
//}
//
//func (db resourceSvcDBOld) volumeGetByID(volID uuid.UUID) (v Volume, err error) {
//	err = db.con.QueryRow(
//		`SELECT
//			id,
//			create_time,
//			tariff_id,
//			capacity,
//			replicas
//		WHERE id=$1`,
//		volID,
//	).Scan(
//		&v.ID,
//		&v.CreateTime,
//		&v.TariffID,
//		&v.Storage,
//		&v.Replicas,
//	)
//	if err == sql.ErrNoRows {
//		err = ErrNoSuchResource
//	}
//	return
//}

type dbTransaction struct {
	tx *sql.Tx
}

func (t *dbTransaction) Commit() error {
	if t.tx != nil {
		err := t.tx.Commit()
		t.tx = nil
		return err
	}
	return nil
}

func (t *dbTransaction) Rollback() error {
	if t.tx != nil {
		err := t.tx.Rollback()
		t.tx = nil
		return err
	}
	return nil
}

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
	mig, err := migrate.NewWithDatabaseInstance(os.Getenv("MIGRATION_URL"), "postgres", inst)
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

func (db resourceSvcDB) namespaceCreate(tariff model.NamespaceTariff, user uuid.UUID, label string) (tr *dbTransaction, nsID uuid.UUID, err error) {
	defer func() {
		if err != nil {
			if tr.tx != nil {
				tr.tx.Rollback()
				tr.tx = nil
			}
		}
	}()

	nsID = uuid.NewV4()
	{
		var count int
		err = db.con.QueryRow(`SELECT count(*) FROM accesses WHERE user_id=$1 AND resource_label=$2 AND kind='Namespace'`, user, label).Scan(&count)
		if err != nil {
			return
		}
		if count > 0 {
			return ErrResourceExists
		}
	}

	tr = new(dbTransaction)
	tr.tx, err = db.con.Begin()
	if err != nil {
		return
	}

	_, err = tr.tx.Exec(
		`INSERT INTO namespaces (
			id,
			ram,
			cpu,
			max_ext_svc,
			max_int_svc,
			max_traffic,
			tariff_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		nsID,
		tariff.MemoryLimit,
		tariff.CpuLimit,
		tariff.ExternalServices,
		tariff.InternalServices,
		tariff.Traffic,
		tariff.TariffID,
	)
	if err != nil {
		return
	}

	_, err = tr.tx.Exec(
		`INSERT INTO accesses(
			id,
			kind,
			resource_id,
			resource_label,
			user_id,
			owner_user_id,
			access_level,
			access_level_change_time,
			limited)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		uuid.NewV4(),
		"Namespace",
		nsID,
		label,
		user,
		user,
		"owner",
		time.Now(),
		false,
	)
	if err != nil {
		return
	}

	return
}

func (db resourceSvcDB) namespaceList(user uuid.UUID) (nss []Namespace, err error) {
	var rows *sql.Rows
	rows, err = db.con.Query(
		`SELECT (
			n.id,
			n.create_time,
			a.resource_label,
			n.ram,
			n.cpu,
			n.max_ext_svc,
			n.max_int_svc,
			n.max_traffic,
			n.deleted,
			n.delete_time,
			n.tariff_id
		) FROM namespaces n INNER JOIN accesses a ON a.resource_id=n.id
		WHERE a.user_id=$1`,
		user,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var ns Namespace
		err = rows.Scan(
			&ns.ID,
			&ns.CreateTime,
			&ns.Label,
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
}

func (db resourceSvcDB) namespaceRename(user uuid.UUID, oldname, newname string) (tr *dbTransaction, err error) {
	tr = new(dbTransaction)
	tr.tx, err = db.con.Begin()
	_, err = tr.tx.Exec(
		`UPDATE accesses SET resource_label=$1
		WHERE resource_label=$2 AND user_id=$3 AND kind='Namespace'`,
		newname,
		oldname,
		user,
	)
	if err != nil {
		tr.Rollback()
	}
	return
}

func (db resourceSvcDB) namespaceSetLimited(owner uuid.UUID, ownerLabel string, limited bool) (tr *dbTransaction, err error) {
	tr = new(dbTransaction)
	tr.tx, err = db.con.Begin()
	_, err = tr.tx.Exec(
		`UPDATE accesses SET limited=$3
		WHERE user_id=$1 AND resource_label=$2 AND kind='Namespace'`,
		owner,
		ownerLabel,
		limited,
	)
	if err != nil {
		tr.Rollback()
	}
	return
}

func (db resourceSvcDB) namespaceSetAccess(owner uuid.UUID, ownerLabel string, other uuid.UUID, access string) (tr *dbTransaction, err error) {
	var resID uuid.UUID

	// check if the owner is really owner and get the resource_id
	err = db.con.QueryRow(
		`SELECT resource_id FROM accesses
		WHERE user_id=owner_user_id AND user_id=$1 AND resource_label=$2 AND kind='Namespace'`,
		owner,
		ownerLabel,
	).Scan(&resID)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrNoSuchResource
		}
		return
	}

	tr = new(dbTransaction)
	tr.tx, err = db.con.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			tr.Rollback()
		}
	}()

	_, err = tr.tx.Exec(
		`UPDATE accesses SET access_level=$1, access_level_change_time=statement_timestamp()
		WHERE resource_id=$2 AND user_id=$3`,
		access,
		resID,
		other,
	)
	if err != nil {
		return
	}
}

func (db resourceSvcDB) namespaceDelete(user uuid.UUID, label string) (tr *dbTransaction, err error) {
	var alvl string
	var limited bool
	var owner uuid.UUID
	var resID uuid.UUID

	err = db.con.QueryRow(`SELECT access_level, owner_user_id, resource_id FROM accesses WHERE user_id=$1 AND resource_label=$2 AND kind='Namespace'`, user, resource_label).Scan(&alvl, &owner, &resID)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrNoSuchResource
		}
		return
	}
	err = db.con.QueryRow(`SELECT limited FROM accesses WHERE resource_id=$1 AND user_id=$2 AND kind='Namespace'`, resID, owner).Scan(&limited)
	if err != nil {
		if err == sql.ErrNoRows {
			err = newError("database consistency error (namespaceDelete, SELECT limited FROM accesses ...)")
		}
		return
	}
	if limited {
		alvl = "readdelete"
	}
	if !permCheck(alvl, "delete") {
		err = ErrDenied
		return
	}

	tr = new(dbTransaction)
	tr.tx, err = db.con.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			tr.Rollback()
		}
	}()
	_, err = tr.tx.Exec(
		`UPDATE namespaces SET deleted=true, delete_time=statement_timestamp() WHERE id=$1`,
		resID,
	)
	if err != nil {
		err = fmt.Errorf("UPDATE namespaces ... : <%T> %[1]v", err)
		return
	}
	if owner == user {
		_, err = tr.tx.Exec(`DELETE FROM accesses WHERE resource_id=$1`, resID)
		if err != nil {
			return
		}
	} else {
		_, err = tr.tx.Exec(`DELETE FROM accesses WHERE resource_id=$1 AND user_id=$2`, resID, user)
		if err != nil {
			return
		}
	}
}

func (db resourceSvcDB) volumeCreate(tariff model.VolumeTariff, user uuid.UUID, label string) (tr *dbTransaction, volID uuid.UUID, err error) {
	defer func() {
		if err != nil {
			if tr.tx != nil {
				tr.tx.Rollback()
				tr.tx = nil
			}
		}
	}()

	volID = uuid.NewV4()
	{
		var count int
		err = db.con.QueryRow(`SELECT count(*) FROM accesses WHERE user_id=$1 AND resource_label=$2 AND kind='Volume'`, user, label).Scan(&count)
		if err != nil {
			return
		}
		if count > 0 {
			return ErrResourceExists
		}
	}

	tr = new(dbTransaction)
	tr.tx, err = db.con.Begin()
	if err != nil {
		return
	}

	_, err = tr.tx.Exec(
		`INSERT INTO volumes (
			id,
			capacity,
			replicas,
			tariff_id
		) VALUES ($1,$2,$3,$4)`,
		volID,
		tariff.StorageLimit,
		tariff.ReplicasLimit,
		tariff.TariffID,
	)
	if err != nil {
		return
	}

	_, err = tr.tx.Exec(
		`INSERT INTO accesses(
			id,
			kind,
			resource_id,
			resource_label,
			user_id,
			owner_user_id,
			access_level,
			access_level_change_time,
			limited)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		uuid.NewV4(),
		"Volume",
		volID,
		label,
		user,
		user,
		"owner",
		time.Now(),
		false,
	)
	if err != nil {
		return
	}

	return
}

func (db resourceSvcDB) volumeList(user uuid.UUID) (vols []Volume, err error) {
	var rows *sql.Rows
	rows, err = db.con.Query(
		`SELECT (
			v.id,
			v.create_time,
			a.resource_label,
			v.deleted,
			v.delete_time,
			v.tariff_id
		) FROM volumes v INNER JOIN accesses a ON a.resource_id=n.id
		WHERE a.user_id=$1`,
		user,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var vol Volume
		err = rows.Scan(
			&vol.ID,
			&vol.CreateTime,
			&vol.Label,
			&vol.Storage,
			&vol.Replicas,
			&vol.Deleted,
			&vol.DeleteTime,
			&vol.TariffID,
		)
		if err != nil {
			return
		}
		vols = append(vols, vol)
	}
}

func (db resourceSvcDB) volumeRename(user uuid.UUID, oldname, newname string) (tr *dbTransaction, err error) {
	tr = new(dbTransaction)
	tr.tx, err = db.con.Begin()
	_, err = tr.tx.Exec(
		`UPDATE accesses SET resource_label=$1
		WHERE resource_label=$2 AND user_id=$3 AND kind='Volume'`,
		newname,
		oldname,
		user,
	)
	if err != nil {
		tr.Rollback()
	}
	return
}

func (db resourceSvcDB) volumeSetAccess(owner uuid.UUID, ownerLabel string, other uuid.UUID, access string) (tr *dbTransaction, err error) {
	var resID uuid.UUID

	// check if the owner is really owner and get the resource_id
	err = db.con.QueryRow(
		`SELECT resource_id FROM accesses
		WHERE user_id=owner_user_id AND user_id=$1 AND resource_label=$2 AND kind='Volume'`,
		owner,
		ownerLabel,
	).Scan(&resID)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrNoSuchResource
		}
		return
	}

	tr = new(dbTransaction)
	tr.tx, err = db.con.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			tr.Rollback()
		}
	}()

	_, err = tr.tx.Exec(
		`UPDATE accesses SET access_level=$1, access_level_change_time=statement_timestamp()
		WHERE resource_id=$2 AND user_id=$3`,
		access,
		resID,
		other,
	)
	if err != nil {
		return
	}
}

func (db resourceSvcDB) volumeSetLimited(owner uuid.UUID, ownerLabel string, limited bool) (tr *dbTransaction, err error) {
	tr = new(dbTransaction)
	tr.tx, err = db.con.Begin()
	_, err = tr.tx.Exec(
		`UPDATE accesses SET limited=$3
		WHERE user_id=$1 AND resource_label=$2 AND kind='Volume'`,
		owner,
		ownerLabel,
		limited,
	)
	if err != nil {
		tr.Rollback()
	}
	return
}

func (db resourceSvcDB) volumeDelete(user uuid.UUID, label string) (tr *dbTransaction, err error) {
	var alvl string
	var limited bool
	var owner uuid.UUID
	var resID uuid.UUID

	err = db.con.QueryRow(
		`SELECT access_level, owner_user_id, resource_id
		FROM accesses
		WHERE user_id=$1 AND resource_label=$2 AND kind='Volume'`,
		user,
		resource_label,
	).Scan(
		&alvl,
		&owner,
		&resID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrNoSuchResource
		}
		return
	}
	err = db.con.QueryRow(`SELECT limited FROM accesses WHERE resource_id=$1 AND user_id=$2 AND kind='Volume'`, resID, owner).Scan(&limited)
	if err != nil {
		if err == sql.ErrNoRows {
			err = newError("database consistency error (volumeDelete, SELECT limited FROM accesses ...)")
		}
		return
	}
	if limited {
		alvl = "readdelete"
	}
	if !permCheck(alvl, "delete") {
		err = ErrDenied
		return
	}

	tr = new(dbTransaction)
	tr.tx, err = db.con.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			tr.Rollback()
		}
	}()
	_, err = tr.tx.Exec(
		`UPDATE volumes SET deleted=true, delete_time=statement_timestamp() WHERE id=$1`,
		resID,
	)
	if err != nil {
		err = fmt.Errorf("UPDATE volumes ... : <%T> %[1]v", err)
		return
	}
	if owner == user {
		_, err = tr.tx.Exec(`DELETE FROM volumes WHERE resource_id=$1`, resID)
		if err != nil {
			return
		}
	} else {
		_, err = tr.tx.Exec(`DELETE FROM accesses WHERE resource_id=$1 AND user_id=$2`, resID, user)
		if err != nil {
			return
		}
	}
}

func (db resourceSvcDB) byID(id uuid.UUID) (obj interface{}, err error) {
	return nil, fmt.Errorf("not implemented")
}
