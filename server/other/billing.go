package other

import (
	//"bytes"
	"context"
	//"encoding/json"
	//"fmt"
	"net/http"
	"net/url"

	"bitbucket.org/exonch/resource-service/server/model"

	"github.com/sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
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
	return nil
}

func (b billingHTTP) GetNamespaceTariff(ctx context.Context, tariffID string) (model.NamespaceTariff, error) {
	return model.NamespaceTariff{}, nil
}

type billingStub struct {
}

func NewBillingStub(_ ...interface{}) Billing {
	return billingStub{}
}

func (billingStub) Subscribe(ctx context.Context, userID, tariffID, resourceID string) error {
	logrus.Warnf("billingStub.Subscribe(%v, %v, %v, %v)", ctx, userID, tariffID, resourceID)
	return nil
}

func (billingStub) Unsubscribe(ctx context.Context, userID, resourceID string) error {
	logrus.Warnf("billingStub.Unsubscribe(%v, %v, %v)", ctx, userID, resourceID)
	return nil
}

var someUUID = uuid.NewV4()

func (billingStub) GetNamespaceTariff(ctx context.Context, tariffID string) (model.NamespaceTariff, error) {
	logrus.Warnf("billingStub.GetNamespaceTariff(%v, %v)", ctx, tariffID)
	nt := model.NamespaceTariff{
		ID: new(uuid.UUID),
		CpuLimit: new(int),
		MemoryLimit: new(int),
		IsActive: new(bool),
		IsPublic: new(bool),
	}
	*nt.ID = someUUID
	*nt.CpuLimit = 20
	*nt.MemoryLimit = 512
	*nt.IsActive = true
	*nt.IsPublic = true
	return nt, nil
}
