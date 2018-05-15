package impl

import (
	"context"

	"git.containerum.net/ch/resource-service/pkg/db"
	"git.containerum.net/ch/resource-service/pkg/models/domain"
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

func (da *DomainActionsImpl) GetDomainsList(ctx context.Context) ([]domain.Domain, error) {
	da.log.Info("get all domains")
	//TODO Add pagination
	return da.mongo.GetDomainsList()
}

func (da *DomainActionsImpl) GetDomain(ctx context.Context, domain string) (*domain.Domain, error) {
	da.log.WithField("domain", domain).Info("get domain")
	return da.mongo.GetDomain(domain)
}

func (da *DomainActionsImpl) AddDomain(ctx context.Context, req domain.Domain) (*domain.Domain, error) {
	da.log.Infof("add domain %#v", req)

	return da.mongo.CreateDomain(req)
}

func (da *DomainActionsImpl) DeleteDomain(ctx context.Context, domain string) error {
	da.log.WithField("domain", domain).Info("delete domain")

	err := da.mongo.DeleteDomain(domain)

	return err
}
