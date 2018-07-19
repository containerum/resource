package impl

import (
	"context"

	"strconv"

	"git.containerum.net/ch/resource-service/pkg/db"
	"git.containerum.net/ch/resource-service/pkg/models/domain"
	"git.containerum.net/ch/resource-service/pkg/util/coblog"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/sirupsen/logrus"
)

type DomainActionsImpl struct {
	mongo *db.MongoStorage
	log   *cherrylog.LogrusAdapter
}

func NewDomainActionsImpl(mongo *db.MongoStorage) *DomainActionsImpl {
	return &DomainActionsImpl{
		mongo: mongo,
		log:   cherrylog.NewLogrusAdapter(logrus.WithField("component", "domain_actions")),
	}
}

func (da *DomainActionsImpl) GetDomainsList(ctx context.Context, page, perPage string) (*domain.DomainsResponse, error) {
	da.log.Infof("get all domains page %q per_page %q", page, perPage)

	pagei, pageerr := strconv.Atoi(page)
	perpagei, perpageerr := strconv.Atoi(perPage)

	if pageerr == nil && perpageerr == nil {
		if pagei > 0 && perpagei > 0 {
			domains, err := da.mongo.GetDomainsList(&db.PageInfo{
				Page:    pagei,
				PerPage: perpagei,
			})
			if err != nil {
				return nil, err
			}
			return &domain.DomainsResponse{Domains: domains}, nil
		}
	}
	domains, err := da.mongo.GetDomainsList(&db.PageInfo{
		Page:    1,
		PerPage: 100,
	})
	if err != nil {
		return nil, err
	}
	return &domain.DomainsResponse{Domains: domains}, nil
}

func (da *DomainActionsImpl) GetDomain(ctx context.Context, domain string) (*domain.Domain, error) {
	da.log.WithField("domain", domain).Info("get domain")
	return da.mongo.GetDomain(domain)
}

func (da *DomainActionsImpl) AddDomain(ctx context.Context, req domain.Domain) (*domain.Domain, error) {
	da.log.Info("add domain")
	coblog.Std.Struct(req)
	return da.mongo.CreateDomain(req)
}

func (da *DomainActionsImpl) DeleteDomain(ctx context.Context, domain string) error {
	da.log.WithField("domain", domain).Info("delete domain")

	err := da.mongo.DeleteDomain(domain)

	return err
}
