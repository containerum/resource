package other

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"bitbucket.org/exonch/resource-manager/server/model"
)

type Billing interface {
	Subscribe(ctx context.Context, tariffLabel, resType, resLabel, userID string) (err error)
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

func (b billingHTTP) Subscribe(ctx context.Context, tariffLabel, resType, resLabel, userID string) error {
	refURL := &url.URL{
		Path: "/user/subscribe",
	}
	reqData := map[string]string{
		"tariff_label":   tariffLabel,
		"resource_type":  resType,
		"resource_label": resLabel,
		"user_id":        userID,
	}
	reqBytes, _ := json.Marshal(reqData)
	reqBuf := bytes.NewReader(reqBytes)
	req, _ := http.NewRequest("POST", b.u.ResolveReference(refURL).String(), reqBuf)
	resp, err := b.c.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("billing: http status %s", resp.Status)
	}

	return nil
}

func (b billingHTTP) Unsubscribe(ctx context.Context) error {
	return nil
}

func (b billingHTTP) GetNamespaceTariff(ctx context.Context, tariffID string) (model.NamespaceTariff, error) {
	return model.NamespaceTariff{}, nil
}
