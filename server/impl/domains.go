package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"github.com/sirupsen/logrus"
)

func (rs *resourceServiceImpl) AddDomain(ctx context.Context, req rstypes.AddDomainRequest) error {
	rs.log.Info("add domain %#v", req)

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		return tx.AddDomain(ctx, req)
	})

	return err
}

func (rs *resourceServiceImpl) GetAllDomains(ctx context.Context, params rstypes.GetAllDomainsQueryParams) (rstypes.GetAllDomainsResponse, error) {
	rs.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
	}).Info("get all domains")

	resp, err := rs.DB.GetAllDomains(ctx, params)

	return resp, err
}

func (rs *resourceServiceImpl) GetDomain(ctx context.Context, domain string) (rstypes.GetDomainResponse, error) {
	rs.log.WithField("domain", domain).Info("get domain")

	resp, err := rs.DB.GetDomain(ctx, domain)

	return resp, err
}

func (rs *resourceServiceImpl) DeleteDomain(ctx context.Context, domain string) error {
	rs.log.WithField("domain", domain).Info("delete domain")

	err := rs.DB.DeleteDomain(ctx, domain)

	return err
}
