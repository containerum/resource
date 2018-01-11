package server

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	rstypes "git.containerum.net/ch/json-types/resource-service"

	_ "github.com/lib/pq"
	"github.com/mattes/migrate"
	mig_postgres "github.com/mattes/migrate/database/postgres"
	_ "github.com/mattes/migrate/source/file"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

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

type dbTransaction struct {
	tx *sql.Tx
}

func (t *dbTransaction) Commit() error {
	if t != nil && t.tx != nil {
		err := t.tx.Commit()
		t.tx = nil
		return err
	}
	return nil
}

func (t *dbTransaction) Rollback() error {
	if t != nil && t.tx != nil {
		err := t.tx.Rollback()
		t.tx = nil
		return err
	}
	return nil
}

// resourceSvcDB is the database interface of the resource service.
//
// Assuming correct usage of returned dbTransaction objects,
// all methods of this type should ideally:
//  - Transition database from one valid state to another.
//  - Do so concurrently.
//
// BUG: the above requirement doesn't hold.
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

func (db resourceSvcDB) namespaceCreate(tariff rstypes.NamespaceTariff, user uuid.UUID, label string) (tr *dbTransaction, nsID uuid.UUID, err error) {
	nsID = uuid.NewV4()
	{
		var count int
		err = db.con.QueryRow(`SELECT count(*) FROM accesses WHERE user_id=$1 AND resource_label=$2 AND kind='Namespace'`, user, label).Scan(&count)
		if err != nil {
			return
		}
		if count > 0 {
			err = ErrAlreadyExists
			return
		}
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
			limited,
			new_access_level
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		uuid.NewV4(),
		"Namespace",
		nsID,
		label,
		user,
		user,
		"owner",
		time.Now(),
		false,
		"owner",
	)
	if err != nil {
		return
	}

	return
}

func (db resourceSvcDB) namespaceList(user uuid.UUID) (nss []Namespace, err error) {
	var rows *sql.Rows
	rows, err = db.con.Query(
		`SELECT
			n.id,
			n.create_time,
			n.deleted,
			n.delete_time,
			n.tariff_id,
			a.resource_label,
			a.access_level,
			a.access_level_change_time,
			a.limited,
			a.new_access_level,
			n.ram,
			n.cpu,
			n.max_ext_svc,
			n.max_int_svc,
			n.max_traffic
		FROM namespaces n INNER JOIN accesses a ON a.resource_id=n.id
		WHERE a.user_id=$1 AND a.kind='Namespace'`,
		user,
	)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var ns Namespace
		err = rows.Scan(
			&ns.ID,
			&ns.CreateTime,
			&ns.Deleted,
			&ns.DeleteTime,
			&ns.TariffID,
			&ns.Label,
			&ns.Access,
			&ns.AccessChangeTime,
			&ns.Limited,
			&ns.NewAccess,
			&ns.RAM,
			&ns.CPU,
			&ns.MaxExtService,
			&ns.MaxIntService,
			&ns.MaxTraffic,
		)
		if err != nil {
			return
		}
		nss = append(nss, ns)
	}
	return
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

func (db resourceSvcDB) namespaceSetAccess(owner uuid.UUID, label string, other uuid.UUID, access string) (tr *dbTransaction, err error) {
	var resID uuid.UUID
	var doInsert bool

	defer func() {
		if err != nil {
			err = dbErrorWrap(err)
		}
	}()

	// get resource id
	err = db.con.QueryRow(
		`SELECT resource_id FROM accesses
		WHERE user_id=$1 AND resource_label=$2 AND owner_user_id=user_id AND kind='Namespace'`,
		owner,
		label,
	).Scan(&resID)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrNoSuchResource
		}
		return
	}

	// UPDATE v INSERT
	err = db.con.QueryRow(
		`SELECT 1 FROM accesses
		WHERE user_id=$1 AND resource_id=$2 AND kind='Namespace'`,
		other,
		resID,
	).Scan(new(int))
	if err == sql.ErrNoRows {
		doInsert = true
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

	if other == owner {
		_, err = tr.tx.Exec(
			`UPDATE accesses SET new_access_level=$1
			WHERE owner_user_id=$2 AND resource_id=$3 AND kind='Namespace'`,
			access,
			owner,
			resID,
		)
	} else {
		if !doInsert {
			_, err = tr.tx.Exec(
				`UPDATE accesses SET new_access_level=$1
				WHERE user_id=$2 AND resource_id=$3 AND kind='Namespace'`,
				access,
				other,
				resID,
			)
		} else {
			_, err = tr.tx.Exec(
				`INSERT INTO accesses (
					id,
					kind,
					resource_id,
					resouce_label,
					user_id,
					owner_user_id,
					access_level,
					new_access_level
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
				uuid.NewV4(),
				"Namespace",
				resID,
				uuid.NewV4().String(),
				other,
				owner,
				access,
				access,
			)
		}
	}

	return
}

func (db resourceSvcDB) namespaceSetTariff(owner uuid.UUID, label string, t rstypes.NamespaceTariff) (tr *dbTransaction, err error) {
	var resID uuid.UUID

	// check if owner & ns_label exists by getting its ID
	err = db.con.QueryRow(
		`SELECT resource_id FROM accesses
		WHERE owner_user_id=user_id AND user_id=$1 AND resource_label=$2
			AND kind='Namespace'`,
		owner,
		label,
	).Scan(&resID)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrNoSuchResource
		} else {
			err = newDBError("SELECT resource_id: %v", err.Error())
		}
		return
	}

	// start txn
	tr = new(dbTransaction)
	tr.tx, err = db.con.Begin()
	if err != nil {
		err = newDBError("BEGIN: %v", err.Error())
		return
	}
	defer func() {
		if err != nil {
			tr.Rollback()
		}
	}()

	// and UPDATE tariff_id and the rest of the fields
	_, err = tr.tx.Exec(
		`UPDATE namespaces SET
			tariff_id=$2,
			cpu=$3,
			ram=$4,
			max_traffic=$5,
			max_ext_svc=$6,
			max_int_svc=$7
		WHERE id=$1`,
		resID,
		t.TariffID,
		t.CpuLimit,
		t.MemoryLimit,
		t.Traffic,
		t.ExternalServices,
		t.InternalServices,
	)
	if err != nil {
		err = newDBError("UPDATE namespaces ...: %v", err)
	}
	return
}

func (db resourceSvcDB) namespaceDelete(user uuid.UUID, label string) (tr *dbTransaction, err error) {
	var alvl string
	var owner uuid.UUID
	var resID uuid.UUID
	var subVolsCnt int

	defer func() {
		if err != nil {
			err = dbErrorWrap(err)
		}
	}()

	err = db.con.QueryRow(
		`SELECT access_level, owner_user_id, resource_id
		FROM accesses
		WHERE user_id=$1 AND resource_label=$2 AND kind='Namespace'`,
		user,
		label,
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

	if owner == user {
		err = db.con.QueryRow(
			`SELECT count(nv.*)
			FROM namespace_volume nv
			WHERE nv.ns_id=$1`,
			resID,
		).Scan(&subVolsCnt)
		if err != nil {
			return
		}
		if subVolsCnt > 0 {
			err = newPermissionError("cannot delete, namespace has associated volumes")
			return
		}
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

	if owner == user {
		_, err = tr.tx.Exec(
			`UPDATE namespaces
			SET deleted=true, delete_time=statement_timestamp()
			WHERE id=$1`,
			resID,
		)
		if err != nil {
			err = fmt.Errorf("UPDATE namespaces ... : %[1]v <%[1]T>", err)
			return
		}
		_, err = tr.tx.Exec(`DELETE FROM accesses WHERE resource_id=$1`, resID)
		if err != nil {
			err = fmt.Errorf("DELETE FROM accesses ...: %[1]v <%[1]T>", err)
			return
		}
	} else {
		_, err = tr.tx.Exec(`DELETE FROM accesses WHERE resource_id=$1 AND user_id=$2`, resID, user)
		if err != nil {
			err = fmt.Errorf("DELETE FROM accesses ...: %[1]v <%[1]T>", err)
			return
		}
	}

	return
}

func (db resourceSvcDB) namespaceAccesses(owner uuid.UUID, label string) (ns Namespace, err error) {
	defer func() {
		if err != nil {
			err = dbErrorWrap(err)
		}
	}()

	err = db.con.QueryRow(
		`SELECT
			n.id,
			n.create_time,
			n.deleted,
			n.delete_time,
			a.user_id,
			n.tariff_id,
			a.resource_label,
			a.access_level,
			a.access_level_change_time,
			n.ram,
			n.cpu,
			n.max_ext_svc,
			n.max_int_svc,
			n.max_traffic
		FROM accesses a INNER JOIN namespaces n ON n.id=a.resource_id
		WHERE a.user_id=$1 AND a.resource_label=$2 AND a.owner_user_id=a.user_id AND a.kind='Namespace'`,
		owner,
		label,
	).Scan(
		&ns.ID,
		&ns.CreateTime,
		&ns.Deleted,
		&ns.DeleteTime,
		&ns.UserID,
		&ns.TariffID,
		&ns.Label,
		&ns.Access,
		&ns.AccessChangeTime,
		&ns.RAM,
		&ns.CPU,
		&ns.MaxExtService,
		&ns.MaxIntService,
		&ns.MaxTraffic,
	)
	if err != nil {
		err = ErrNoSuchResource
		return
	}

	var rows *sql.Rows
	rows, err = db.con.Query(
		`SELECT
			user_id,
			access_level,
			limited,
			new_access_level,
			access_level_change_time
		FROM accesses
		WHERE kind='Namespace' AND resource_id=$1`,
		ns.ID,
	)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var ar accessRecord
		err = rows.Scan(
			&ar.UserID,
			&ar.Access,
			&ar.Limited,
			&ar.NewAccess,
			&ar.AccessChangeTime,
		)
		if err != nil {
			return
		}
		ns.Users = append(ns.Users, ar)
	}
	return
}

func (db resourceSvcDB) namespaceVolumeAssociate(nsID, vID uuid.UUID) (tr *dbTransaction, err error) {
	defer func() {
		if err != nil {
			tr.Rollback()
		}
	}()
	tr = new(dbTransaction)
	tr.tx, err = db.con.Begin()
	if err != nil {
		return
	}
	_, err = tr.tx.Exec(
		`INSERT INTO namespace_volume (ns_id, vol_id)
		VALUES ($1,$2)`,
		nsID,
		vID,
	)
	return
}

func (db resourceSvcDB) namespaceVolumeListAssoc(nsID uuid.UUID) (vl []Volume, err error) {
	var rows *sql.Rows
	rows, err = db.con.Query(
		`SELECT nv.vol_id,
			v.create_time,
			v.deleted,
			v.delete_time,
			v.tariff_id,
			a.resource_label,
			a.access_level,
			a.access_level_change_time,
			a.limited,
			a.new_access_level,
			v.capacity,
			v.replicas,
			v.is_persistent
		FROM namespace_volume nv
			INNER JOIN accesses a ON a.resource_id = nv.vol_id
			INNER JOIN volumes v ON v.id = nv.vol_id
		WHERE nv.ns_id=$1`,
		nsID,
	)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var v Volume
		err = rows.Scan(
			&v.ID,
			&v.CreateTime,
			&v.Deleted,
			&v.DeleteTime,
			&v.TariffID,
			&v.Label,
			&v.Access,
			&v.AccessChangeTime,
			&v.Limited,
			&v.NewAccess,
			&v.Storage,
			&v.Replicas,
			&v.Persistent,
		)
		if err != nil {
			return
		}
		vl = append(vl, v)
	}
	return
}

func (db resourceSvcDB) volumeCreate(tariff rstypes.VolumeTariff, user uuid.UUID, label string) (tr *dbTransaction, volID uuid.UUID, err error) {
	volID = uuid.NewV4()
	{
		var count int
		err = db.con.QueryRow(`SELECT count(*) FROM accesses WHERE user_id=$1 AND resource_label=$2 AND kind='Volume'`, user, label).Scan(&count)
		if err != nil {
			return
		}
		if count > 0 {
			err = ErrAlreadyExists
			return
		}
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
		`INSERT INTO volumes (
			id,
			capacity,
			replicas,
			tariff_id,
			is_persistent
		) VALUES ($1,$2,$3,$4,$5)`,
		volID,
		tariff.StorageLimit,
		tariff.ReplicasLimit,
		tariff.TariffID,
		tariff.IsPersistent,
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
			limited,
			new_access_level
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		uuid.NewV4(),
		"Volume",
		volID,
		label,
		user,
		user,
		"owner",
		time.Now(),
		false,
		"owner",
	)
	if err != nil {
		return
	}

	return
}

func (db resourceSvcDB) volumeList(user uuid.UUID) (vols []Volume, err error) {
	var rows *sql.Rows
	rows, err = db.con.Query(
		`SELECT
			v.id,
			v.create_time,
			v.deleted,
			v.delete_time,
			v.tariff_id,
			a.resource_label,
			a.access_level,
			a.access_level_change_time,
			a.limited,
			a.new_access_level,
			v.capacity,
			v.replicas,
			v.is_persistent
		FROM volumes v INNER JOIN accesses a ON a.resource_id=v.id
		WHERE a.user_id=$1 AND a.kind='Volume'`,
		user,
	)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var vol Volume
		err = rows.Scan(
			&vol.ID,
			&vol.CreateTime,
			&vol.Deleted,
			&vol.DeleteTime,
			&vol.TariffID,
			&vol.Label,
			&vol.Access,
			&vol.AccessChangeTime,
			&vol.Limited,
			&vol.NewAccess,
			&vol.Storage,
			&vol.Replicas,
			&vol.Persistent,
		)
		if err != nil {
			return
		}
		vols = append(vols, vol)
	}
	return
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

func (db resourceSvcDB) volumeSetAccess(owner uuid.UUID, label string, other uuid.UUID, access string) (tr *dbTransaction, err error) {
	var resID uuid.UUID
	var doInsert bool

	defer func() {
		if err != nil {
			err = dbErrorWrap(err)
		}
	}()

	// get resource id
	err = db.con.QueryRow(
		`SELECT resource_id FROM accesses
		WHERE user_id=$1 AND resource_label=$2 AND owner_user_id=user_id AND kind='Volume'`,
		owner,
		label,
	).Scan(&resID)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrNoSuchResource
		}
		return
	}

	// UPDATE v INSERT
	err = db.con.QueryRow(
		`SELECT 1 FROM accesses
		WHERE user_id=$1 AND resource_id=$2 AND kind='Volume'`,
		other,
		resID,
	).Scan(new(int))
	if err == sql.ErrNoRows {
		doInsert = true
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

	if other == owner {
		_, err = tr.tx.Exec(
			`UPDATE accesses SET new_access_level=$1
			WHERE owner_user_id=$2 AND resource_id=$3 AND kind='Volume'`,
			access,
			owner,
			resID,
		)
	} else {
		if !doInsert {
			_, err = tr.tx.Exec(
				`UPDATE accesses SET new_access_level=$1
				WHERE user_id=$2 AND resource_id=$3 AND kind='Volume'`,
				access,
				other,
				resID,
			)
		} else {
			_, err = tr.tx.Exec(
				`INSERT INTO accesses (
					id,
					kind,
					resource_id,
					resouce_label,
					user_id,
					owner_user_id,
					access_level,
					new_access_level
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
				uuid.NewV4(),
				"Volume",
				resID,
				uuid.NewV4().String(),
				other,
				owner,
				access,
				access,
			)
		}
	}

	return
}

func (db resourceSvcDB) volumeSetTariff(owner uuid.UUID, label string, t rstypes.VolumeTariff) (tr *dbTransaction, err error) {
	var resID uuid.UUID

	// check if owner & ns_label exists by getting its ID
	err = db.con.QueryRow(
		`SELECT resource_id FROM accesses
		WHERE owner_user_id=user_id AND user_id=$1 AND resource_label=$2
			AND kind='Volume'`,
		owner,
		label,
	).Scan(&resID)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrNoSuchResource
		} else {
			err = newDBError("SELECT resource_id: %v", err)
		}
		return
	}

	// start txn
	tr = new(dbTransaction)
	tr.tx, err = db.con.Begin()
	if err != nil {
		err = newDBError("BEGIN: %v", err)
		return
	}
	defer func() {
		if err != nil {
			tr.Rollback()
		}
	}()

	// and UPDATE tariff_id and the rest of the fields
	_, err = tr.tx.Exec(
		`UPDATE volumes SET
			tariff_id=$2,
			capacity=$3,
			replicas=$4,
			is_persistent=$5
		WHERE id=$1`,
		resID,
		t.TariffID,
		t.StorageLimit,
		t.ReplicasLimit,
		t.IsPersistent,
	)
	if err != nil {
		err = newDBError("UPDATE volumes ...: %v", err)
	}
	return
}

func (db resourceSvcDB) volumeDelete(user uuid.UUID, label string) (tr *dbTransaction, err error) {
	var alvl string
	var owner uuid.UUID
	var resID uuid.UUID

	defer func() {
		if err != nil {
			err = dbErrorWrap(err)
		}
	}()

	err = db.con.QueryRow(
		`SELECT access_level, owner_user_id, resource_id
		FROM accesses
		WHERE user_id=$1 AND resource_label=$2 AND kind='Volume'`,
		user,
		label,
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

	if owner == user {
		_, err = tr.tx.Exec(
			`UPDATE volumes SET deleted=true, delete_time=statement_timestamp()
			WHERE id=$1`,
			resID,
		)
		if err != nil {
			err = fmt.Errorf("UPDATE volumes ... : %[1]v <%[1]T>", err)
			return
		}

		_, err = tr.tx.Exec(`DELETE FROM accesses WHERE resource_id=$1`, resID)
		if err != nil {
			err = fmt.Errorf("DELETE FROM accesses ...: %[1]v <%[1]T>", err)
			return
		}
		_, err = tr.tx.Exec(`DELETE FROM namespace_volume WHERE vol_id=$1`, resID)
		if err != nil {
			err = fmt.Errorf("DELETE FROM namespace_volume ...: %[1]v <%[1]T>", err)
			return
		}
	} else {
		_, err = tr.tx.Exec(`DELETE FROM accesses WHERE resource_id=$1 AND user_id=$2`, resID, user)
		if err != nil {
			err = fmt.Errorf("DELETE FROM accesses ...: %[1]v <%[1]T>", err)
			return
		}
	}

	return
}

func (db resourceSvcDB) volumeAccesses(owner uuid.UUID, label string) (vol Volume, err error) {
	defer func() {
		if err != nil {
			err = dbErrorWrap(err)
		}
	}()

	err = db.con.QueryRow(
		`SELECT
			v.id,
			v.create_time,
			v.deleted,
			v.delete_time,
			a.user_id,
			v.tariff_id,
			a.resource_label,
			a.access_level,
			a.access_level_change_time,
			v.capacity,
			v.replicas,
			v.is_persistent
		FROM accesses a INNER JOIN volumes v ON v.id=a.resource_id
		WHERE a.user_id=$1 AND a.resource_label=$2 AND a.owner_user_id=a.user_id AND a.kind='Volume'`,
		owner,
		label,
	).Scan(
		&vol.ID,
		&vol.CreateTime,
		&vol.Deleted,
		&vol.DeleteTime,
		&vol.UserID,
		&vol.TariffID,
		&vol.Label,
		&vol.Access,
		&vol.AccessChangeTime,
		&vol.Storage,
		&vol.Replicas,
		&vol.Persistent,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrNoSuchResource
		}
		return
	}

	var rows *sql.Rows
	rows, err = db.con.Query(
		`SELECT
			user_id,
			access_level,
			limited,
			new_access_level,
			access_level_change_time
		FROM accesses
		WHERE kind='Volume' AND resource_id=$1`,
		vol.ID,
	)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var ar accessRecord
		err = rows.Scan(
			&ar.UserID,
			&ar.Access,
			&ar.Limited,
			&ar.NewAccess,
			&ar.AccessChangeTime,
		)
		if err != nil {
			return
		}
		vol.Users = append(vol.Users, ar)
	}
	return
}

// byID is supposed to fetch any kind of model by searching all models for the id.
func (db resourceSvcDB) byID(id uuid.UUID) (obj interface{}, err error) {
	return nil, fmt.Errorf("not implemented")
}

func (db resourceSvcDB) namespaceListAllByTime(ctx context.Context, after time.Time, count uint) (nsch <-chan Namespace, err error) {
	var direction string = ctx.Value("sort-direction").(string) //assuming the actual method function validated this data
	var rows *sql.Rows
	rows, err = db.con.QueryContext(
		ctx,
		`SELECT
			n.id,
			n.create_time,
			n.deleted,
			n.delete_time,
			n.tariff_id,
			a.access_level,
			a.access_level_change_time,
			a.resource_label,
			n.ram,
			n.cpu,
			n.max_ext_svc,
			n.max_int_svc,
			n.max_traffic,
			a.user_id
		FROM namespaces n INNER JOIN accesses a ON a.resource_id=n.id
		WHERE a.kind='Namespace' AND n.create_time > $1
		ORDER BY n.create_time `+direction+` LIMIT $2`,
		after,
		count,
	)
	if err != nil {
		// Doesn not matter if context was canceled, it is an error
		// if this method doesn't return at least one result.
		err = newDBError(err.Error())
		return
	}

	nsch1 := make(chan Namespace)
	nsch2 := make(chan Namespace)
	nsch = nsch2
	go streamNamespaces(ctx, nsch1, rows)
	go streamNSAddVolumes(ctx, db.con, nsch2, nsch1)

	return
}

func (db resourceSvcDB) namespaceListAllByOwner(ctx context.Context, after uuid.UUID, count uint) (nsch <-chan Namespace, err error) {
	var direction string = ctx.Value("sort-direction").(string)
	var rows *sql.Rows
	rows, err = db.con.QueryContext(
		ctx,
		`SELECT
			n.id,
			n.create_time,
			n.deleted,
			n.delete_time,
			n.tariff_id,
			a.access_level,
			a.access_level_change_time,
			a.resource_label,
			n.ram,
			n.cpu,
			n.max_ext_svc,
			n.max_int_svc,
			n.max_traffic,
			a.user_id
		FROM namespaces n INNER JOIN accesses a ON a.resource_id=n.id
		WHERE a.kind='Namespace' AND a.owner_user_id=a.user_id AND a.user_id > $1
		ORDER BY a.user_id `+direction+` LIMIT $2`,
		after,
		count,
		direction,
	)
	if err != nil {
		// Doesn not matter if context was canceled, it is an error
		// if this method doesn't return at least one result.
		err = newDBError(err.Error())
		return
	}

	nsch1 := make(chan Namespace)
	nsch2 := make(chan Namespace)
	nsch = nsch2
	go streamNamespaces(ctx, nsch2, rows)
	go streamNSAddVolumes(ctx, db.con, nsch2, nsch1)

	return
}

func streamNamespaces(ctx context.Context, ch chan<- Namespace, rows *sql.Rows) {
	var err error
	defer close(ch)
	defer rows.Close()
loop:
	for rows.Next() {
		var ns Namespace
		err = rows.Scan(
			&ns.ID,
			&ns.CreateTime,
			&ns.Deleted,
			&ns.DeleteTime,
			&ns.TariffID,
			&ns.Access,
			&ns.AccessChangeTime,
			&ns.Label,
			&ns.RAM,
			&ns.CPU,
			&ns.MaxExtService,
			&ns.MaxIntService,
			&ns.MaxTraffic,
			&ns.UserID,
		)
		if err != nil {
			return
		}
		select {
		case <-ctx.Done():
			break loop
		case ch <- ns:
		}
	}
}

func streamNSAddVolumes(ctx context.Context, con *sql.DB, out chan<- Namespace, in <-chan Namespace) {
	log := logrus.StandardLogger().
		WithField("function", "streamNSAddVolumes").
		WithField("module", "git.containerum.net/ch/resource-service/server")
	for ns := range in {
		var rowsv *sql.Rows
		var err error
		rowsv, err = con.QueryContext(
			ctx,
			`SELECT
				v.id,
				v.create_time,
				v.deleted,
				v.delete_time,
				v.tariff_id,
				a.resource_label,
				a.access_level,
				a.access_level_change_time,
				v.capacity,
				v.replicas
			FROM volumes v
				INNER JOIN namespace_volume nv ON v.id = nv.vol_id
				INNER JOIN accesses a ON a.resource_id = v.id
			WHERE nv.ns_id=$1`,
			ns.ID,
		)
		if err != nil {
			log.Errorf("namespace volumes sql failed: %v", err)
			goto sendns
		}
		for rowsv.Next() {
			var v Volume
			err = rowsv.Scan(
				&v.ID,
				&v.CreateTime,
				&v.Deleted,
				&v.DeleteTime,
				&v.TariffID,
				&v.Label,
				&v.Access,
				&v.AccessChangeTime,
				&v.Storage,
				&v.Replicas,
			)
			if err != nil {
				break
			}
			ns.Volumes = append(ns.Volumes, v)
		}
		rowsv.Close()
	sendns:
		out <- ns
	}
	close(out)
}

func streamVolumes(ctx context.Context, ch chan<- Volume, rows *sql.Rows) {
	var err error
	defer close(ch)
	defer rows.Close()
loop:
	for rows.Next() {
		var v Volume
		err = rows.Scan(
			&v.ID,
			&v.CreateTime,
			&v.Deleted,
			&v.DeleteTime,
			&v.TariffID,
			&v.Label,
			&v.Access,
			&v.AccessChangeTime,
			&v.Storage,
			&v.Replicas,
			&v.UserID,
		)
		if err != nil {
			return
		}
		select {
		case <-ctx.Done():
			break loop
		case ch <- v:
		}
	}
}

func (db resourceSvcDB) volumeListAllByTime(ctx context.Context, after time.Time, count uint) (vch chan Volume, err error) {
	var direction string = ctx.Value("sort-direction").(string) //assuming the actual method function validated this data
	var rows *sql.Rows
	rows, err = db.con.QueryContext(
		ctx,
		`SELECT
			v.id,
			v.create_time,
			v.deleted,
			v.delete_time,
			v.tariff_id,
			a.resource_label,
			a.access_level,
			a.access_level_change_time,
			v.capacity,
			v.replicas,
			a.user_id
		FROM volumes v INNER JOIN accesses a ON a.resource_id=v.id
		WHERE a.kind='Volume' AND v.create_time > $1
		ORDER BY v.create_time `+direction+` LIMIT $2`,
		after,
		count,
	)
	if err != nil {
		err = newDBError(err.Error())
		return
	}
	vch = make(chan Volume)
	go streamVolumes(ctx, vch, rows)
	return
}
