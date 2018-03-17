package postgres

import (
	"context"

	"database/sql"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func (db *PGDB) CreateStorage(ctx context.Context, req rstypes.CreateStorageRequest) (err error) {
	db.log.Debugf("creating storage %#v", req)

	// we can`t recognise constraint violation error so do this check before insert
	var exists bool
	query, args, _ := sqlx.Named( /* language=sql */ `SELECT count(*)>0 FROM storages WHERE "name" = :name`, req)
	err = sqlx.GetContext(ctx, db, &exists, db.Rebind(query), args...)
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	if exists {
		err = rserrors.ErrResourceAlreadyExists().AddDetailF("storage %s already exists", req.Name)
		return
	}

	_, err = sqlx.NamedExecContext(ctx, db, /* language=sql */
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

func (db *PGDB) GetStorages(ctx context.Context) (ret []rstypes.Storage, err error) {
	db.log.Debug("get storages")

	ret = make([]rstypes.Storage, 0)
	err = sqlx.SelectContext(ctx, db, &ret /* language=sql */, `SELECT * FROM storages`)
	switch err {
	case nil, sql.ErrNoRows:
		err = nil
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}

func (db *PGDB) UpdateStorage(ctx context.Context, name string, req rstypes.UpdateStorageRequest) (err error) {
	db.log.WithField("name", name).Debug("update storage with %#v", req)

	result, err := sqlx.NamedExecContext(ctx, db, /* language=sql */
		`UPDATE storages
		SET
			name = COALESCE(:name, name),
			replicas = COALESCE(:replicas, replicas),
			size = COALESCE(:size, size),
			ips = COALESCE(:ips, ips)
		WHERE name = :oldname`,
		map[string]interface{}{
			"oldname":  name,
			"name":     req.Name,
			"replicas": req.Replicas,
			"size":     req.Size,
			"ips":      pq.StringArray(req.IPs),
		})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return err
	}

	if count, _ := result.RowsAffected(); count <= 0 {
		err = rserrors.ErrResourceNotExists().AddDetailF("storage %s not exists", name).Log(err, db.log)
	}

	return
}

func (db *PGDB) DeleteStorage(ctx context.Context, name string) (err error) {
	db.log.WithField("name", name).Debug("delete storage")

	result, err := sqlx.NamedExecContext(ctx, db, /* language=sql */
		`DELETE FROM storages WHERE name = :name`,
		map[string]interface{}{"name": name})
	if err != nil {
		err = rserrors.ErrDatabase().Log(err, db.log)
		return
	}
	if count, _ := result.RowsAffected(); count <= 0 {
		err = rserrors.ErrResourceNotExists().AddDetailF("storage %s not exists", name).Log(err, db.log)
	}

	return
}

func (db *PGDB) ChooseAvailableStorage(ctx context.Context, minFree int) (storage rstypes.Storage, err error) {
	db.log.WithField("min_free", minFree).Debug("choose appropriate storage")

	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT * 
		FROM storages
		WHERE size - used >= :min_free AND name != 'DUMMY'
		LIMIT 1`,
		map[string]interface{}{"min_free": minFree})
	err = sqlx.GetContext(ctx, db, &storage, db.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = rserrors.ErrInsufficientStorage().AddDetailF("can`t find storage to host % Gb volume", minFree)
	default:
		err = rserrors.ErrDatabase().Log(err, db.log)
	}

	return
}
