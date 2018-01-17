package models

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"git.containerum.net/ch/grpc-proto-files/auth"
	"git.containerum.net/ch/json-types/errors"
	rserrors "git.containerum.net/ch/resource-service/server/errors"
	"git.containerum.net/ch/utils"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/mattes/migrate"
	mig_postgres "github.com/mattes/migrate/database/postgres"
	_ "github.com/mattes/migrate/source/file"
	"github.com/sirupsen/logrus"
)

func PermCheck(perm, needed string) bool {
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
	logrus.Errorf("unreachable code in db.go:/^func PermCheck")
	return false
}

type ResourceSvcDB struct {
	conn *sqlx.DB // do not use it in select/exec operations
	log  *logrus.Entry
	qLog sqlx.QueryerContext
	eLog sqlx.ExecerContext
}

func DBConnect(dbDSN string) (*ResourceSvcDB, error) {
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
		return nil, errors.Format("what the fuck is this: %v", err)
	}
	mig, err := migrate.NewWithDatabaseInstance(os.Getenv("MIGRATION_URL"), "postgres", inst)
	if err != nil {
		return nil, errors.Format("cannot create migration: %v", err)
	}
	if err = mig.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, errors.Format("cannot run migration: %v", err)
	}
	return &ResourceSvcDB{
		conn: conn,
		log:  log,
		qLog: utils.NewSQLXContextQueryLogger(conn, log),
		eLog: utils.NewSQLXContextExecLogger(conn, log),
	}, nil
}

var (
	ErrTransactionBegin    = errors.New("transaction begin error")
	ErrTransactionRollback = errors.New("transaction rollback error")
	ErrTransactionCommit   = errors.New("transaction commit error")
)

func (db ResourceSvcDB) Transactional(f func(tx ResourceSvcDB) error) (err error) {
	start := time.Now().Format(time.ANSIC)
	e := db.log.WithField("transaction_at", start)
	e.Debugln("Begin transaction")
	tx, txErr := db.conn.Beginx()
	if txErr != nil {
		e.WithError(txErr).Errorln("Begin transaction error")
		return ErrTransactionBegin
	}

	arg := ResourceSvcDB{
		conn: db.conn,
		log:  e,
		eLog: utils.NewSQLXContextExecLogger(tx, e),
		qLog: utils.NewSQLXContextQueryLogger(tx, e),
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

// ByID is supposed to fetch any kind of model by searching all models for the id.
func (db ResourceSvcDB) ByID(id string) (obj interface{}, err error) {
	return nil, fmt.Errorf("not implemented")
}

func (db ResourceSvcDB) NamespaceListAllByTime(ctx context.Context, after time.Time, count uint) (nsch <-chan Namespace, err error) {
	direction := ctx.Value("sort-direction").(string) //assuming the actual method function validated this data
	rows, err := db.qLog.QueryContext(ctx,
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
		err = rserrors.ErrNoSuchResource
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

func (db ResourceSvcDB) NamespaceListAllByOwner(ctx context.Context, after string, count uint) (nsch <-chan Namespace, err error) {
	direction := ctx.Value("sort-direction").(string)
	rows, err := db.qLog.QueryContext(ctx,
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
		return
	}

	nsch1 := make(chan Namespace)
	nsch2 := make(chan Namespace)
	nsch = nsch2
	go db.streamNamespaces(ctx, nsch2, rows)
	go db.streamNSAddVolumes(ctx, nsch2, nsch1)

	return
}

func (db ResourceSvcDB) streamNamespaces(ctx context.Context, ch chan<- Namespace, rows *sql.Rows) {
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

func (db ResourceSvcDB) streamNSAddVolumes(ctx context.Context, out chan<- Namespace, in <-chan Namespace) {
	log := db.log.WithField("function", "streamNSAddVolumes")
	for ns := range in {
		var err error
		rowsv, err := db.qLog.QueryContext(ctx,
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

func (db ResourceSvcDB) streamVolumes(ctx context.Context, ch chan<- Volume, rows *sql.Rows) {
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

func (db ResourceSvcDB) VolumeListAllByTime(ctx context.Context, after time.Time, count uint) (vch chan Volume, err error) {
	direction := ctx.Value("sort-direction").(string) //assuming the actual method function validated this data
	rows, err := db.qLog.QueryContext(ctx,
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
		err = rserrors.ErrNoSuchResource
		return
	default:
		return
	}
	vch = make(chan Volume)
	go db.streamVolumes(ctx, vch, rows)
	return
}

func (db ResourceSvcDB) UserResourceAccess(ctx context.Context, owner string) (*auth.ResourcesAccess, error) {
	rows, err := db.qLog.QueryContext(ctx, `SELECT resource_label, id, access_level, kind
													FROM accesses
													WHERE owner_user_id = $1 AND 
															user_id = owner_user_id AND
															kind IN ('Namespace', 'Volume') `,
		owner)
	switch err {
	case nil:
	case sql.ErrNoRows:
		return nil, rserrors.ErrNoSuchResource
	default:
		return nil, err
	}
	defer rows.Close()
	var resp auth.ResourcesAccess
	for rows.Next() {
		var obj auth.AccessObject
		var kind, accessLevelStr string
		err = rows.Scan(
			&obj.Label,
			&obj.Id,
			&accessLevelStr,
		)
		accessLevel, ok := auth.Role_value[accessLevelStr]
		if !ok {
			return nil, errors.Format("access level %s not defined in grpc", accessLevel)
		}
		switch kind {
		case "Namespace":
			resp.Namespace = append(resp.Namespace, &obj)
		case "Volume":
			resp.Volume = append(resp.Volume, &obj)
		default:
			return nil, errors.Format("unexpected kind %s", kind)
		}
	}
	return &resp, rows.Err()
}

func (db ResourceSvcDB) Close() error {
	return db.conn.Close()
}
