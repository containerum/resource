package other

import (
	"context"
	"net/http"
	"net/url"

	"bitbucket.org/exonch/resource-manager/server/model"
)

type Billing interface {
	Subscribe(ctx context.Context) (billingID string, err error)
	Unsubscribe(ctx context.Context) error
	GetNamespaceTariff(ctx context.Context, id string) (model.NamespaceTariff, error)
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

func (b billingHTTP) Subscribe(ctx context.Context) (string, error) {
	return "", nil
}

func (b billingHTTP) Unsubscribe(ctx context.Context) error {
	return nil
}

func (b billingHTTP) GetNamespaceTariff(ctx context.Context, tariffID string) (model.NamespaceTariff, error) {
	return model.NamespaceTariff{}, nil
}
