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
		ON CONFLICT (ip) DO UPDATE SET
			domain = EXCLUDED.domain, 
			domain_group = EXCLUDED.domain_group`)
	if err != nil {
		err = models.WrapDBError(err)
		return
	}
	defer stmt.Close()

	for _, ip := range req.IP {
		_, err = stmt.ExecContext(ctx, rstypes.Domain{
			IP:          ip,
			Domain:      req.Domain,
			DomainGroup: req.DomainGroup,
		})
		if err != nil {
			err = models.WrapDBError(err)
			return
		}
	}

	return
}

func (db *pgDB) GetAllDomains(ctx context.Context, params rstypes.GetAllDomainsQueryParams) (domains []rstypes.DomainEntry, err error) {
	db.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
	}).Debug("get all domains")

	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT * FROM domains LIMIT :limit OFFSET :offset`,
		map[string]interface{}{"limit": params.PerPage, "offset": params.PerPage * (params.Page - 1)})
	dbEntries := make([]rstypes.Domain, 0)
	err = sqlx.SelectContext(ctx, db.extLog, &dbEntries, db.extLog.Rebind(query), args...)
	switch err {
	case nil, sql.ErrNoRows:
		err = nil
		return
	default:
		err = models.WrapDBError(err)
		return
	}

	domainMap := make(map[string]rstypes.DomainEntry)
	for _, v := range dbEntries {
		entry := domainMap[v.Domain]
		entry.Domain = v.Domain
		entry.DomainGroup = v.DomainGroup
		entry.IP = append(entry.IP, v.IP)
		domainMap[v.Domain] = entry
	}
	for _, v := range domainMap {
		domains = append(domains, v)
	}

	return
}

func (db *pgDB) GetDomain(ctx context.Context, domain string) (entry rstypes.DomainEntry, err error) {
	db.log.WithField("domain", domain).Debug("get domain")

	query, args, _ := sqlx.Named( /* language=sql */
		`SELECT * FROM domains WHERE domain = :domain`,
		rstypes.Domain{Domain: domain})
	dbEntries := make([]rstypes.Domain, 0)
	err = sqlx.SelectContext(ctx, db.extLog, &dbEntries, db.extLog.Rebind(query), args...)
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = models.ErrLabeledResourceNotExists
		return
	default:
		err = models.WrapDBError(err)
		return
	}

	entry.Domain = domain
	entry.DomainGroup = dbEntries[0].DomainGroup
	for _, v := range dbEntries {
		entry.IP = append(entry.IP, v.IP)
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
