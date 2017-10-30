package other

import (
	"context"
	"net/http"
	"net/url"

	"bitbucket.org/exonch/resource-manager/server/model"
)

type Billing interface {
	Subscribe(ctx context.Context, userID, tariffID, resourceID string) error
	Unsubscribe(ctx context.Context, resourceID string) error
	GetTariffByID(ctx context.Context, id string) (model.Tariff, error) // ...ByLabel?
}

type billingHTTP struct {
	c *http.Client
	u *url.URL
}

func NewBilling(u *url.URL) (b billingHTTP) {
	b.c = http.DefaultClient
	b.u = u
	return
}

func (b billingHTTP) Subscribe(trfLabel string, resType string, resLabel string, userID string) error {
	return nil
}

func (b billingHTTP) Unsubscribe(resID string) error {
	return nil
}

func (b billingHTTP) GetTariffByID(resID string) (model.Tariff, error) {
	return model.Tariff{}, nil
}
