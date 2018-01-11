package server

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	rstypes "git.containerum.net/ch/json-types/resource-service"

	"git.containerum.net/ch/json-types/errors"
	"git.containerum.net/ch/utils"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/mattes/migrate"
	mig_postgres "github.com/mattes/migrate/database/postgres"
	_ "github.com/mattes/migrate/source/file"
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

// resourceSvcDB is the database interface of the resource service.
//
// Assuming correct usage of returned dbTransaction objects,
// all methods of this type should ideally:
//  - Transition database from one valid state to another.
//  - Do so concurrently.
//
// BUG: the above requirement doesn't hold.
type resourceSvcDB struct {
	conn *sqlx.DB // do not use it in select/exec operations
	log  *logrus.Entry
	qLog *utils.SQLXQueryLogger
	eLog *utils.SQLXExecLogger
}

func dbConnect(dbDSN string) (*resourceSvcDB, error) {
	conn, err := sqlx.Open("postgres", dbDSN)
	if err != nil {
		return nil, err
	}

	log := logrus.WithField("component", "db")

	log.Debugln("pinging")
	if err := conn.Ping(); err != nil {
		return nil, err
	}

	inst, err := mig_postgres.WithInstance(conn.DB, &mig_postgres.Config{})
	if err != nil {
		return nil, newError("what the fuck is this: %v", err)
	}
	mig, err := migrate.NewWithDatabaseInstance(os.Getenv("MIGRATION_URL"), "postgres", inst)
	if err != nil {
		return nil, newError("cannot create migration: %v", err)
	}
	if err = mig.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, newError("cannot run migration: %v", err)
	}
	return &resourceSvcDB{
		conn: conn,
		log:  log,
		qLog: utils.NewSQLXQueryLogger(conn, log),
		eLog: utils.NewSQLXExecLogger(conn, log),
	}, nil
}

var (
	ErrTransactionBegin    = errors.New("transaction begin error")
	ErrTransactionRollback = errors.New("transaction rollback error")
	ErrTransactionCommit   = errors.New("transaction commit error")
)

func (db resourceSvcDB) transactional(f func(tx resourceSvcDB) error) (err error) {
	start := time.Now().Format(time.ANSIC)
	e := db.log.WithField("transaction_at", start)
	e.Debugln("Begin transaction")
	tx, txErr := db.conn.Beginx()
	if txErr != nil {
		e.WithError(txErr).Errorln("Begin transaction error")
		return ErrTransactionBegin
	}

	arg := resourceSvcDB{
		conn: db.conn,
		log:  e,
		eLog: utils.NewSQLXExecLogger(tx, e),
		qLog: utils.NewSQLXQueryLogger(tx, e),
	}

	// needed for recovering panics in transactions.
	defer func(dberr error) {
		// if panic recovered, try to rollback transaction
		if panicErr := recover(); panicErr != nil {
			dberr = fmt.Errorf("panic in transaction: %v", panicErr)
		}

		if dberr != nil {
			e.WithError(dberr).Debugln("Rollback transaction")
			if rerr := tx.Rollback(); rerr != nil {
				e.WithError(rerr).Errorln("Rollback error")
				err = ErrTransactionRollback
			}
			err = dberr // forward error with panic description
			return
		}

		e.Debugln("Commit transaction")
		if cerr := tx.Commit(); cerr != nil {
			e.WithError(cerr).Errorln("Commit error")
			err = ErrTransactionCommit
		}
	}(f(arg))

	return
}

func (db resourceSvcDB) namespaceCreate(tariff rstypes.NamespaceTariff, user string, label string) (nsID string, err error) {
	nsID = utils.NewUUID()
	{
		var count int
		db.qLog.QueryRowx(`SELECT count(*)
									FROM accesses
									WHERE user_id=$1 AND resource_label=$2 AND kind='Namespace'`,
			user, label).Scan(&count)
		if err != nil {
			return
		}
		if count > 0 {
			err = ErrAlreadyExists
			return
		}
	}

	_, err = db.eLog.Exec(
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

	_, err = db.eLog.Exec(
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
		utils.NewUUID(),
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

func (db resourceSvcDB) namespaceList(user string) (nss []Namespace, err error) {
	rows, err := db.qLog.Query(
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

func (db resourceSvcDB) namespaceRename(user string, oldname, newname string) (err error) {
	_, err = db.eLog.Exec(
		`UPDATE accesses SET resource_label=$1
		WHERE resource_label=$2 AND user_id=$3 AND kind='Namespace'`,
		newname,
		oldname,
		user,
	)
	return
}

func (db resourceSvcDB) namespaceSetLimited(owner string, ownerLabel string, limited bool) (err error) {
	_, err = db.eLog.Exec(
		`UPDATE accesses SET limited=$3
		WHERE user_id=$1 AND resource_label=$2 AND kind='Namespace'`,
		owner,
		ownerLabel,
		limited,
	)
	return
}

func (db resourceSvcDB) namespaceSetAccess(owner string, label string, other string, access string) (err error) {
	var resID string

	// get resource id
	err = db.qLog.QueryRowx(
		`SELECT resource_id FROM accesses
		WHERE user_id=$1 AND resource_label=$2 AND owner_user_id=user_id AND kind='Namespace'`,
		owner,
		label,
	).Scan(&resID)
	switch err {
	case nil:
	case sql.ErrNoRows:
		return ErrNoSuchResource
	default:
		return
	}

	if other == owner {
		_, err = db.eLog.Exec(
			`UPDATE accesses SET new_access_level=$1
			WHERE owner_user_id=$2 AND resource_id=$3 AND kind='Namespace'`,
			access,
			owner,
			resID,
		)
	} else {
		_, err = db.eLog.Exec(
			`INSERT INTO accesses (
					id,
					kind,
					resource_id,
					resouce_label,
					user_id,
					owner_user_id,
					access_level,
					new_access_level
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
				ON CONFLICT (resource_id, user_id) DO UPDATE SET new_access_level = $8`,
			utils.NewUUID(),
			"Namespace",
			resID,
			utils.NewUUID(),
			other,
			owner,
			access,
			access,
		)
	}

	return
}

func (db resourceSvcDB) namespaceSetTariff(owner string, label string, t rstypes.NamespaceTariff) (err error) {
	var resID string

	// check if owner & ns_label exists by getting its ID
	err = db.qLog.QueryRowx(
		`SELECT resource_id FROM accesses
		WHERE owner_user_id=user_id AND user_id=$1 AND resource_label=$2
			AND kind='Namespace'`,
		owner,
		label,
	).Scan(&resID)
	switch err {
	case nil:
	case sql.ErrNoRows:
		return ErrNoSuchResource
	default:
		return
	}

	// and UPDATE tariff_id and the rest of the fields
	_, err = db.eLog.Exec(
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
	return
}

func (db resourceSvcDB) namespaceDelete(user string, label string) (err error) {
	var alvl string
	var owner string
	var resID string
	var subVolsCnt int

	defer func() {
		if err != nil {
			err = dbErrorWrap(err)
		}
	}()

	err = db.qLog.QueryRowx(
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
	switch err {
	case nil:
	case sql.ErrNoRows:
		return ErrNoSuchResource
	default:
		return
	}

	if owner == user {
		err = db.qLog.QueryRowx(
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

	if owner == user {
		_, err = db.eLog.Exec(
			`UPDATE namespaces
			SET deleted=true, delete_time=statement_timestamp()
			WHERE id=$1`,
			resID,
		)
		if err != nil {
			err = fmt.Errorf("UPDATE namespaces ... : %[1]v <%[1]T>", err)
			return
		}
		_, err = db.eLog.Exec(`DELETE FROM accesses WHERE resource_id=$1`, resID)
		if err != nil {
			err = fmt.Errorf("DELETE FROM accesses ...: %[1]v <%[1]T>", err)
			return
		}
	} else {
		_, err = db.eLog.Exec(`DELETE FROM accesses WHERE resource_id=$1 AND user_id=$2`, resID, user)
		if err != nil {
			err = fmt.Errorf("DELETE FROM accesses ...: %[1]v <%[1]T>", err)
			return
		}
	}

	return
}

func (db resourceSvcDB) namespaceAccesses(owner string, label string) (ns Namespace, err error) {
	defer func() {
		if err != nil {
			err = dbErrorWrap(err)
		}
	}()

	err = db.qLog.QueryRowx(
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
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = ErrNoSuchResource
		return
	default:
		return
	}

	rows, err := db.qLog.Query(
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

func (db resourceSvcDB) namespaceVolumeAssociate(nsID, vID string) (err error) {
	_, err = db.eLog.Exec(
		`INSERT INTO namespace_volume (ns_id, vol_id)
		VALUES ($1,$2)`,
		nsID,
		vID,
	)
	return
}

func (db resourceSvcDB) namespaceVolumeListAssoc(nsID string) (vl []Volume, err error) {
	rows, err := db.qLog.Query(
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

func (db resourceSvcDB) volumeCreate(tariff rstypes.VolumeTariff, user string, label string) (volID string, err error) {
	volID = utils.NewUUID()
	{
		var count int
		err = db.qLog.QueryRowx(`SELECT count(*) FROM accesses WHERE user_id=$1 AND resource_label=$2 AND kind='Volume'`, user, label).Scan(&count)
		if err != nil {
			return
		}
		if count > 0 {
			err = ErrAlreadyExists
			return
		}
	}

	_, err = db.eLog.Exec(
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

	_, err = db.eLog.Exec(
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
		utils.NewUUID(),
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
	return
}

func (db resourceSvcDB) volumeList(user string) (vols []Volume, err error) {
	rows, err := db.qLog.Query(
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
		user)
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

func (db resourceSvcDB) volumeRename(user string, oldname, newname string) (err error) {
	_, err = db.eLog.Exec(
		`UPDATE accesses SET resource_label=$1
		WHERE resource_label=$2 AND user_id=$3 AND kind='Volume'`,
		newname,
		oldname,
		user,
	)
	return
}

func (db resourceSvcDB) volumeSetLimited(owner string, ownerLabel string, limited bool) (err error) {
	_, err = db.eLog.Exec(
		`UPDATE accesses SET limited=$3
		WHERE user_id=$1 AND resource_label=$2 AND kind='Volume'`,
		owner,
		ownerLabel,
		limited,
	)
	return
}

func (db resourceSvcDB) volumeSetAccess(owner string, label string, other string, access string) (err error) {
	var resID string

	// get resource id
	err = db.qLog.QueryRowx(
		`SELECT resource_id FROM accesses
		WHERE user_id=$1 AND resource_label=$2 AND owner_user_id=user_id AND kind='Volume'`,
		owner,
		label,
	).Scan(&resID)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = ErrNoSuchResource
		return
	default:
		return
	}

	if other == owner {
		_, err = db.eLog.Exec(
			`UPDATE accesses SET new_access_level=$1
			WHERE owner_user_id=$2 AND resource_id=$3 AND kind='Volume'`,
			access,
			owner,
			resID,
		)
	} else {
		_, err = db.eLog.Exec(
			`INSERT INTO accesses (
					id,
					kind,
					resource_id,
					resouce_label,
					user_id,
					owner_user_id,
					access_level,
					new_access_level
					) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
					ON CONFLICT (resource_id, user_id) DO UPDATE SET new_access_level = $8`,
			utils.NewUUID(),
			"Volume",
			resID,
			utils.NewUUID(),
			other,
			owner,
			access,
			access,
		)
	}

	return
}

func (db resourceSvcDB) volumeSetTariff(owner string, label string, t rstypes.VolumeTariff) (err error) {
	var resID string

	// check if owner & ns_label exists by getting its ID
	err = db.qLog.QueryRowx(
		`SELECT resource_id FROM accesses
		WHERE owner_user_id=user_id AND user_id=$1 AND resource_label=$2
			AND kind='Volume'`,
		owner,
		label,
	).Scan(&resID)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = ErrNoSuchResource
		return
	default:
		return
	}

	// UPDATE tariff_id and the rest of the fields
	_, err = db.eLog.Exec(
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
	return
}

func (db resourceSvcDB) volumeDelete(user string, label string) (err error) {
	var alvl string
	var owner string
	var resID string

	err = db.qLog.QueryRowx(
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
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = ErrNoSuchResource
		return
	default:
		return
	}

	if owner == user {
		_, err = db.eLog.Exec(
			`UPDATE volumes SET deleted=true, delete_time=statement_timestamp()
			WHERE id=$1`,
			resID,
		)
		if err != nil {
			err = fmt.Errorf("UPDATE volumes ... : %[1]v <%[1]T>", err)
			return
		}

		_, err = db.eLog.Exec(`DELETE FROM accesses WHERE resource_id=$1`, resID)
		if err != nil {
			err = fmt.Errorf("DELETE FROM accesses ...: %[1]v <%[1]T>", err)
			return
		}
		_, err = db.eLog.Exec(`DELETE FROM namespace_volume WHERE vol_id=$1`, resID)
		if err != nil {
			err = fmt.Errorf("DELETE FROM namespace_volume ...: %[1]v <%[1]T>", err)
			return
		}
	} else {
		_, err = db.eLog.Exec(`DELETE FROM accesses WHERE resource_id=$1 AND user_id=$2`, resID, user)
		if err != nil {
			err = fmt.Errorf("DELETE FROM accesses ...: %[1]v <%[1]T>", err)
			return
		}
	}

	return
}

func (db resourceSvcDB) volumeAccesses(owner string, label string) (vol Volume, err error) {
	err = db.qLog.QueryRowx(
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
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = ErrNoSuchResource
		return
	default:
		return
	}

	rows, err := db.qLog.Query(
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
func (db resourceSvcDB) byID(id string) (obj interface{}, err error) {
	return nil, fmt.Errorf("not implemented")
}

func (db resourceSvcDB) namespaceListAllByTime(ctx context.Context, after time.Time, count uint) (nsch <-chan Namespace, err error) {
	direction := ctx.Value("sort-direction").(string) //assuming the actual method function validated this data
	rows, err := db.qLog.Query(
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
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = ErrNoSuchResource
		return
	default:
		return
	}

	nsch1 := make(chan Namespace)
	nsch2 := make(chan Namespace)
	nsch = nsch2
	go db.streamNamespaces(ctx, nsch1, rows)
	go db.streamNSAddVolumes(ctx, nsch2, nsch1)

	return
}

func (db resourceSvcDB) namespaceListAllByOwner(ctx context.Context, after string, count uint) (nsch <-chan Namespace, err error) {
	direction := ctx.Value("sort-direction").(string)
	rows, err := db.qLog.Query(
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
	go db.streamNamespaces(ctx, nsch2, rows)
	go db.streamNSAddVolumes(ctx, nsch2, nsch1)

	return
}

func (db resourceSvcDB) streamNamespaces(ctx context.Context, ch chan<- Namespace, rows *sql.Rows) {
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

func (db resourceSvcDB) streamNSAddVolumes(ctx context.Context, out chan<- Namespace, in <-chan Namespace) {
	log := db.log.WithField("function", "streamNSAddVolumes")
	for ns := range in {
		var err error
		rowsv, err := db.qLog.Query(
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

func (db resourceSvcDB) streamVolumes(ctx context.Context, ch chan<- Volume, rows *sql.Rows) {
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
	direction := ctx.Value("sort-direction").(string) //assuming the actual method function validated this data
	rows, err := db.qLog.Query(
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
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = ErrNoSuchResource
		return
	default:
		return
	}
	vch = make(chan Volume)
	go db.streamVolumes(ctx, vch, rows)
	return
}
