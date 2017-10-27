package other

type Billing interface {
	Subscribe(ctx context.Context, userID, tariffID, resourceID string) error
	Unsubscribe(ctx context.Context, resourceID string) error
}

type billingHTTP struct {
	c *http.Client
	u *url.URL
}

func NewBillingHTTP(c *http.Client, u *url.URL) (b billingHTTP) {
	b.c = c
	b.u = u
	return
}

func (b billingHTTP) Subscribe(trfLabel string, resType string, resLabel string, userID string) error {
}

func (b billingHTTP) Unsubscribe(resID string) error {
}
