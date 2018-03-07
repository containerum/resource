package postgres

import (
	"context"

	"database/sql"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
	"github.com/jmoiron/sqlx"
)

func (db *pgDB) CreateStorage(ctx context.Context, req rstypes.CreateStorageRequest) (err error) {
	db.log.Debugf("creating storage %#v", req)

	_, err = sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
		`INSERT INTO storages
		(name, size, replicas, ips)
		VALUES (:name, :size, :replicas, :ips)
		RETURNING *`,
		rstypes.Storage{Name: req.Name, Size: req.Size, Replicas: req.Replicas, IPs: req.IPs})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}

func (db *pgDB) GetStorages(ctx context.Context) (ret []rstypes.Storage, err error) {
	db.log.Debug("get storages")

	ret = make([]rstypes.Storage, 0)
	err = sqlx.SelectContext(ctx, db.extLog, &ret /* language=sql */, `SELECT * FROM storages`)
	switch err {
	case nil, sql.ErrNoRows:
		err = nil
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}

func (db *pgDB) UpdateStorage(ctx context.Context, name string, req rstypes.UpdateStorageRequest) (err error) {
	db.log.WithField("name", name).Debug("update storage with %#v", req)

	result, err := sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
		`UPDATE storages
		SET
			name = COALESCE(:name, name),
			replicas = COALESCE(:replicas, replicas),
			size = COALESCE(:size, size),
			ips = COALESCE(:ips, ips)
		WHERE name = :oldname`,
		map[string]interface{}{"oldname": name, "name": req.Name, "replicas": req.Replicas, "size": req.Size, "ips": req.IPs})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return err
	}

	if count, _ := result.RowsAffected(); count <= 0 {
		err = rserrors.ErrResourceNotExists().Log(err, db.log)
	}

	return
}

func (db *pgDB) DeleteStorage(ctx context.Context, name string) (err error) {
	db.log.WithField("name", name).Debug("delete storage")

	result, err := sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
		`DELETE FROM storages WHERE name = :name`,
		map[string]interface{}{"name": name})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	if count, _ := result.RowsAffected(); count <= 0 {
		err = rserrors.ErrResourceNotExists().Log(err, db.log)
	}

	return
}

func (db *pgDB) ChooseAvailableStorage(ctx context.Context, minFree int) (storage rstypes.Storage, err error) {
	db.log.WithField("min_free", minFree).Debug("choose appropriate storage")

	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT * 
		FROM storages
		WHERE size - used >= :min_free AND name != 'DUMMY'
		LIMIT 1`,
		map[string]interface{}{"min_free": minFree})
	err = sqlx.GetContext(ctx, db.extLog, &storage, db.extLog.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}
