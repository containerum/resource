package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"git.containerum.net/ch/resource-service/server"
	"github.com/sirupsen/logrus"
)

func (rs *resourceServiceImpl) AddDomain(ctx context.Context, req rstypes.AddDomainRequest) error {
	rs.log.Info("add domain %#v", req)

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		return tx.AddDomain(ctx, req)
	})
	if err != nil {
		err = server.HandleDBError(err)
	}

	return err
}

func (rs *resourceServiceImpl) GetAllDomains(ctx context.Context, params rstypes.GetAllDomainsQueryParams) (rstypes.GetAllDomainsResponse, error) {
	rs.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
	}).Info("get all domains")

	resp, err := rs.DB.GetAllDomains(ctx, params)
	if err != nil {
		err = server.HandleDBError(err)
		return nil, err
	}

	return resp, nil
}

func (rs *resourceServiceImpl) GetDomain(ctx context.Context, domain string) (rstypes.GetDomainResponse, error) {
	rs.log.WithField("domain", domain).Info("get domain")

	resp, err := rs.DB.GetDomain(ctx, domain)
	if err != nil {
		err = server.HandleDBError(err)
		return rstypes.GetDomainResponse{}, err
	}

	return resp, nil
}

func (rs *resourceServiceImpl) DeleteDomain(ctx context.Context, domain string) error {
	rs.log.WithField("domain", domain).Info("delete domain")

	if err := rs.DB.DeleteDomain(ctx, domain); err != nil {
		err = server.HandleDBError(err)
		return err
	}

	return nil
}
