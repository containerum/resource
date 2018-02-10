package postgres

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
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
