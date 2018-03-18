package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/cherrylog"
	"git.containerum.net/ch/resource-service/pkg/models"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/sirupsen/logrus"
)

type DomainActionsDB struct {
	DomainDB models.DomainDBConstructor
}

type DomainActionsImpl struct {
	*server.ResourceServiceClients
	*DomainActionsDB

	log *cherrylog.LogrusAdapter
}

func NewDomainActionsImpl(clients *server.ResourceServiceClients, constructors *DomainActionsDB) *DomainActionsImpl {
	return &DomainActionsImpl{
		ResourceServiceClients: clients,
		DomainActionsDB:        constructors,
		log:                    cherrylog.NewLogrusAdapter(logrus.WithField("component", "domain_actions")),
	}
}

func (da *DomainActionsImpl) AddDomain(ctx context.Context, req rstypes.AddDomainRequest) error {
	da.log.Info("add domain %#v", req)

	err := da.DB.Transactional(ctx, func(ctx context.Context, tx models.RelationalDB) error {
		return da.DomainDB(tx).AddDomain(ctx, req)
	})

	return err
}

func (da *DomainActionsImpl) GetAllDomains(ctx context.Context, params rstypes.GetAllDomainsQueryParams) (rstypes.GetAllDomainsResponse, error) {
	da.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
	}).Info("get all domains")

	resp, err := da.DomainDB(da.DB).GetAllDomains(ctx, params)

	return resp, err
}

func (da *DomainActionsImpl) GetDomain(ctx context.Context, domain string) (rstypes.GetDomainResponse, error) {
	da.log.WithField("domain", domain).Info("get domain")

	resp, err := da.DomainDB(da.DB).GetDomain(ctx, domain)

	return resp, err
}

func (da *DomainActionsImpl) DeleteDomain(ctx context.Context, domain string) error {
	da.log.WithField("domain", domain).Info("delete domain")

	err := da.DomainDB(da.DB).DeleteDomain(ctx, domain)

	return err
}
