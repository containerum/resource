package postgres

import (
	"context"
	"fmt"

	"git.containerum.net/ch/resource-service/pkg/models"
	sqlxutil "github.com/containerum/utils/sqlxutil"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgresql database driver
	"github.com/mattes/migrate"
	migdrv "github.com/mattes/migrate/database/postgres"
	_ "github.com/mattes/migrate/source/file" // needed to load migrations scripts from files
	"github.com/sirupsen/logrus"

	"time"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/containerum/kube-client/pkg/cherry/resource-service"
)

type PG struct {
	sqlx.ExtContext
	sqlxutil.SQLXPreparer

	conn *sqlx.DB // do not use for operations
	log  *cherrylog.LogrusAdapter

	// for information
	pgConnStr         string
	migrations        string
	migrationsVersion string
}

// DBConnect initializes connection to postgresql database.
// github.com/jmoiron/sqlx used to to get work with database.
// Function tries to ping database and apply migrations using github.com/mattes/migrate.
// If migrations applying failed database goes to dirty state and requires manual conflict resolution.
func DBConnect(pgConnStr string, migrations string) (*PG, error) {
	log := logrus.WithField("component", "postgres_db")
	log.Infoln("Connecting to ", pgConnStr)
	conn, err := sqlx.Connect("postgres", pgConnStr)
	if err != nil {
		log.WithError(err).Errorln("postgres connection failed")
		return nil, err
	}

	ret := &PG{
		conn:         conn,
		log:          cherrylog.NewLogrusAdapter(log),
		ExtContext:   sqlxutil.NewSQLXExtContextLogger(conn, log),
		SQLXPreparer: sqlxutil.NewSQLXPreparerLogger(conn, log),
	}

	m, err := ret.migrateUp(migrations)
	if err != nil {
		return nil, err
	}
	version, dirty, err := m.Version()
	log.WithError(err).WithFields(logrus.Fields{
		"dirty":   dirty,
		"version": version,
	}).Infoln("Migrate up")

	ret.pgConnStr = pgConnStr
	ret.migrations = migrations
	ret.migrationsVersion = fmt.Sprintf("%v; dirty = %v", version, dirty)

	return ret, nil
}

func (db *PG) migrateUp(path string) (*migrate.Migrate, error) {
	db.log.Infof("Running migrations")
	instance, err := migdrv.WithInstance(db.conn.DB, &migdrv.Config{})
	if err != nil {
		return nil, err
	}
	m, err := migrate.NewWithDatabaseInstance(path, "clickhouse", instance)
	if err != nil {
		return nil, err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, err
	}
	return m, nil
}

func (db *PG) Transactional(ctx context.Context, f func(ctx context.Context, tx models.RelationalDB) error) (err error) {
	e := db.log.WithField("transaction_id", time.Now().UTC().Unix())
	e.Debugln("Begin transaction")
	log := cherrylog.NewLogrusAdapter(e)
	tx, txErr := db.conn.Beginx()
	if txErr != nil {
		return rserrors.ErrDatabase().Log(txErr, log)
	}

	arg := &PG{
		conn:         db.conn,
		log:          log,
		ExtContext:   sqlxutil.NewSQLXExtContextLogger(tx, e),
		SQLXPreparer: sqlxutil.NewSQLXPreparerLogger(tx, e),
	}

	// needed for recovering panics in transactions.
	var dberr error

	defer func() {
		// if panic recovered, try to rollback transaction
		if panicErr := recover(); panicErr != nil {
			dberr = rserrors.ErrDatabase().AddDetailF("caused by %v", panicErr)
		}

		if dberr != nil {
			e.WithError(dberr).Debugln("Rollback transaction")
			if rerr := tx.Rollback(); rerr != nil {
				err = rserrors.ErrDatabase().AddDetailF("caused by %v", dberr).Log(rerr, log)
				return
			}
			err = dberr // forward error
			return
		}

		e.Debugln("Commit transaction")
		if cerr := tx.Commit(); cerr != nil {
			err = rserrors.ErrDatabase().Log(cerr, log)
		}
	}()

	dberr = f(ctx, arg)

	return
}

func (db *PG) String() string {
	return fmt.Sprintf("address: %s, migrations path: %s (version: %s)",
		db.pgConnStr, db.migrations, db.migrationsVersion)
}

func (db *PG) Close() error {
	return db.conn.Close()
}

type ResourceCountPG struct {
	models.RelationalDB
	log *cherrylog.LogrusAdapter
}

func NewResourceCountPG(db models.RelationalDB) models.ResourceCountDB {
	return &ResourceCountPG{
		RelationalDB: db,
		log:          cherrylog.NewLogrusAdapter(logrus.WithField("component", "resource_count_pg")),
	}
}

func (db *ResourceCountPG) GetResourcesCount(ctx context.Context, userID string) (ret rstypes.GetResourcesCountResponse, err error) {
	db.log.WithField("user_id", userID).Debug("get resources count")

	var nsIDs []string
	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT DISTINCT resource_id
		FROM permissions
		WHERE (user_id = :user_id OR owner_user_id = :user_id) AND kind = 'namespace'`,
		map[string]interface{}{"user_id": userID})
	err = sqlx.SelectContext(ctx, db, &nsIDs, db.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	query, args, _ = sqlx.Named( /* language=sql */
		`SELECT count(DISTINCT resource_id)
			FROM permissions
			WHERE (owner_user_id = :user_id OR user_id = :user_id) AND kind = 'volume'`,
		map[string]interface{}{"user_id": userID})
	err = sqlx.GetContext(ctx, db, &ret.Volumes, db.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	ret.Namespaces = len(nsIDs)
	if ret.Namespaces <= 0 {
		return ret, nil
	}

	var services struct {
		ExtServices int `db:"extcnt"`
		IntServices int `db:"intcnt"`
	}
	query, args, _ = sqlx.In( /* language=sql */
		`SELECT
			count(s.*) FILTER (WHERE s.type = 'external') AS extcnt,
			count(s.*) FILTER (WHERE s.type = 'internal') AS intcnt
		FROM services s
		JOIN deployments d ON s.deploy_id = d.id AND NOT d.deleted
		WHERE d.ns_id IN (?) AND NOT s.deleted`, nsIDs)
	err = sqlx.GetContext(ctx, db, &services, db.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}

	ret.ExtServices = services.ExtServices
	ret.IntServices = services.IntServices

	var deplIDs []string
	if len(nsIDs) > 0 {
		query, args, _ = sqlx.In( /* language=sql */ `SELECT id FROM deployments WHERE ns_id IN (?) AND NOT deleted`, nsIDs)
		err = sqlx.SelectContext(ctx, db, &deplIDs, db.Rebind(query), args...)
		if err != nil {
			err = rserrors.ErrDatabase().Log(err, db.log)
			return
		}
	}

	ret.Deployments = len(deplIDs)

	if ret.Deployments > 0 {
		query, args, _ = sqlx.In( /* language=sql */
			`SELECT count(*) 
		FROM ingresses i
		JOIN services s ON i.service_id = s.id AND NOT s.deleted
		WHERE s.deploy_id IN (?)`,
			deplIDs)
		err = sqlx.GetContext(ctx, db, &ret.Ingresses, db.Rebind(query), args...)
		if err != nil {
			err = rserrors.ErrDatabase().Log(err, db.log)
			return
		}

		query, args, _ = sqlx.In( /* language=sql */ `SELECT count(*) FROM containers WHERE depl_id IN (?)`, deplIDs)
		err = sqlx.GetContext(ctx, db, &ret.Containers, db.Rebind(query), args...)
		if err != nil {
			err = rserrors.ErrDatabase().Log(err, db.log)
			return
		}

		query, args, _ = sqlx.In( /* language=sql */ `SELECT sum(replicas) FROM deployments WHERE id IN (?) AND NOT deleted`, deplIDs)
		err = sqlx.GetContext(ctx, db, &ret.Pods, db.Rebind(query), args...)
		if err != nil {
			err = rserrors.ErrDatabase().Log(err, db.log)
			return
		}
	}
	return
}
