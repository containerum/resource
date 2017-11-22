package other

import (
	//"bytes"
	"context"
	//"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"bitbucket.org/exonch/resource-service/server/model"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

type Billing interface {
	Subscribe(ctx context.Context, userID, tariffID, resourceID string) error
	Unsubscribe(ctx context.Context, userID, resourceID string) error
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

func (b billingHTTP) Subscribe(ctx context.Context, userID, tariffID, resourceID string) error {
	//refURL := &url.URL{
	//	Path: "/user/subscribe",
	//}
	//reqData := map[string]string{
	//	"tariff_label":   tariffLabel,
	//	"resource_type":  resType,
	//	"resource_label": resLabel,
	//	"user_id":        userID,
	//}
	//reqBytes, _ := json.Marshal(reqData)
	//reqBuf := bytes.NewReader(reqBytes)
	//req, _ := http.NewRequest("POST", b.u.ResolveReference(refURL).String(), reqBuf)
	//resp, err := b.c.Do(req)
	//if err != nil {
	//	return err
	//}

	//if resp.StatusCode/100 != 2 {
	//	return fmt.Errorf("billing: http status %s", resp.Status)
	//}

	return nil
}

func (b billingHTTP) Unsubscribe(ctx context.Context, userID, resourceID string) error {
	return fmt.Errorf("not implemented")
}

func (b billingHTTP) GetNamespaceTariff(ctx context.Context, tariffID string) (model.NamespaceTariff, error) {
	return model.NamespaceTariff{}, nil
}

type billingStub struct {
}

func NewBillingStub() Billing {
	return billingStub{}
}

func (billingStub) Subscribe(ctx context.Context, userID, tariffID, resourceID string) error {
	logrus.Infof("billingStub.Subscribe userID=%s tariffID=%s resourceID=%s",
		userID, tariffID, resourceID)
	return nil
}

func (billingStub) Unsubscribe(ctx context.Context, userID, resourceID string) error {
	logrus.Infof("billingStub.Unsubscribe userID=%s resourceID=%s",
		userID, resourceID)
	return nil
}

var someUUID = uuid.NewV4()

func (billingStub) GetNamespaceTariff(ctx context.Context, tariffID string) (model.NamespaceTariff, error) {
	logrus.Infof("Billing.GetNamespaceTariff tariffID=%s", tariffID)
	nt := model.NamespaceTariff{
		ID:               new(uuid.UUID),
		TariffID:         new(uuid.UUID),
		CpuLimit:         new(int),
		MemoryLimit:      new(int),
		Traffic:          new(int),
		ExternalServices: new(int),
		InternalServices: new(int),
		IsActive:         new(bool),
		IsPublic:         new(bool),
	}
	*nt.ID = someUUID
	*nt.TariffID = uuid.FromStringOrNil(tariffID)
	*nt.CpuLimit = 20
	*nt.MemoryLimit = 512
	*nt.Traffic = 1000
	*nt.ExternalServices = 10
	*nt.InternalServices = 100
	*nt.IsActive = true
	*nt.IsPublic = true
	return nt, nil
}
