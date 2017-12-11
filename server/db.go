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

func (db resourceSvcDB) namespaceCreate(tariff model.NamespaceTariff, user uuid.UUID, label string) (tr *dbTransaction, nsID uuid.UUID, err error) {
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
			&ns.Access,
			&ns.AccessChangeTime,
			&ns.Label,
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
	return
}

func (db resourceSvcDB) namespaceDelete(user uuid.UUID, label string) (tr *dbTransaction, err error) {
	var alvl string
	var limited bool
	var owner uuid.UUID
	var resID uuid.UUID
	var subVolsCnt int

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

	err = db.con.QueryRow(
		`SELECT limited
		FROM accesses
		WHERE resource_id=$1 AND user_id=$2 AND kind='Namespace'`,
		resID,
		owner,
	).Scan(&limited)
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
			return
		}
	} else {
		_, err = tr.tx.Exec(`DELETE FROM accesses WHERE resource_id=$1 AND user_id=$2`, resID, user)
		if err != nil {
			return
		}
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
			v.capacity,
			v.replicas
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
			&v.Storage,
			&v.Replicas,
		)
		if err != nil {
			return
		}
		vl = append(vl, v)
	}
	return
}

func (db resourceSvcDB) volumeCreate(tariff model.VolumeTariff, user uuid.UUID, label string) (tr *dbTransaction, volID uuid.UUID, err error) {
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
			&vol.Storage,
			&vol.Replicas,
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
	err = db.con.QueryRow(
		`SELECT limited
		FROM accesses
		WHERE resource_id=$1 AND user_id=$2 AND kind='Volume'`,
		resID,
		owner,
	).Scan(&limited)
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

		_, err = tr.tx.Exec(`DELETE FROM volumes WHERE id=$1`, resID)
		if err != nil {
			return
		}
		_, err = tr.tx.Exec(`DELETE FROM accesses WHERE resource_id=$1`, resID)
		if err != nil {
			return
		}
		_, err = tr.tx.Exec(`DELETE FROM namespace_volume WHERE vol_id=$1`, resID)
		if err != nil {
			return
		}
	} else {
		_, err = tr.tx.Exec(`DELETE FROM accesses WHERE resource_id=$1 AND user_id=$2`, resID, user)
		if err != nil {
			return
		}
	}

	return
}

func (db resourceSvcDB) byID(id uuid.UUID) (obj interface{}, err error) {
	return nil, fmt.Errorf("not implemented")
}
