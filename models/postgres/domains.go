package postgres

import (
	"context"

	"database/sql"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

func (db *pgDB) AddDomain(ctx context.Context, req rstypes.AddDomainRequest) (err error) {
	db.log.Debugf("add domain %#v")

	stmt, err := db.preparer.PrepareNamed( /* language=sql */
		`INSERT INTO domains
		(ip, domain, domain_group)
		VALUES (:ip, :domain, :domain_group)
		ON CONFLICT (domain) DO UPDATE SET
			ip = EXCLUDED.ip,
			domain_group = EXCLUDED.domain_group`)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, req)
	if err != nil {
		err = models.WrapDBError(err)
	}

	return
}

func (db *pgDB) GetAllDomains(ctx context.Context, params rstypes.GetAllDomainsQueryParams) (domains []rstypes.Domain, err error) {
	db.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
	}).Debug("get all domains")

	domains = make([]rstypes.Domain, 0)
	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT * FROM domains LIMIT :limit OFFSET :offset`,
		map[string]interface{}{"limit": params.PerPage, "offset": params.PerPage * (params.Page - 1)})
	err = sqlx.SelectContext(ctx, db.extLog, &domains, db.extLog.Rebind(query), args...)
	if err != nil {
		err = models.WrapDBError(err)
	}

	return
}

func (db *pgDB) GetDomain(ctx context.Context, domain string) (entry rstypes.Domain, err error) {
	db.log.WithField("domain", domain).Debug("get domain")

	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT * FROM domains WHERE domain = :domain`,
		rstypes.Domain{Domain: domain})
	err = sqlx.GetContext(ctx, db.extLog, &entry, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = models.ErrDomainNotExists
	default:
		err = models.WrapDBError(err)
	}

	return
}

func (db *pgDB) DeleteDomain(ctx context.Context, domain string) (err error) {
	db.log.WithField("domain", domain).Debug("delete domain")

	result, err := sqlx.NamedExecContext(ctx, db.extLog, /* language=sql */
		`DELETE FROM domains WHERE domain = :domain`,
		rstypes.Domain{Domain: domain})
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	if count, _ := result.RowsAffected(); count == 0 {
		err = models.ErrLabeledResourceNotExists
	}

	return
}

func (db *pgDB) ChooseRandomDomain(ctx context.Context) (entry rstypes.Domain, err error) {
	db.log.Debugf("choose random domain")

	err = sqlx.SelectContext(ctx, db.extLog, &entry, /* language=sql*/
		`SELECT * FROM domains ORDER BY RANDOM() LIMIT 1`)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = models.ErrDomainNotExists
	default:
		err = models.WrapDBError(err)
	}

	return
}
