package other

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	//"math/big"
	"net/http"
	"net/url"

	"git.containerum.net/ch/resource-service/server/model"

	//uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

type Billing interface {
	Subscribe(ctx context.Context, userID, tariffID, resourceID string) error
	Unsubscribe(ctx context.Context, userID, resourceID string) error

	GetNamespaceTariff(ctx context.Context, id string) (model.NamespaceTariff, error)
	GetVolumeTariff(ctx context.Context, id string) (model.VolumeTariff, error)
}

type billingHTTP struct {
	c *http.Client
	u *url.URL
}

func NewBillingHTTP(u *url.URL) Billing {
	return billingHTTP{
		http.DefaultClient,
		u,
	}
}

func (b billingHTTP) Subscribe(ctx context.Context, userID, tariffID, resourceID string) error {
	refURL := &url.URL{
		Path: "/user/subscribe",
	}
	reqData := map[string]string{
		//"tariff_label":   tariffLabel,
		//"resource_type":  resType,
		//"resource_label": resLabel,
		"resource_id": resourceID,
		"user_id":     userID,
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

func (b billingHTTP) Unsubscribe(ctx context.Context, userID, resourceID string) error {
	refURL := &url.URL{
		Path: "/user/unsubscribe",
	}
	reqData := map[string]string{
		"resource_id": resourceID,
		"user_id":     userID,
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

func (b billingHTTP) GetNamespaceTariff(ctx context.Context, tariffID string) (nst model.NamespaceTariff, err error) {
	refURL := &url.URL{
		Path: "/namespace_tariffs",
	}
	req, _ := http.NewRequest("POST", b.u.ResolveReference(refURL).String(), nil)
	resp, err := b.c.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return
	}
	if resp.StatusCode/100 != 2 {
		err = fmt.Errorf("billing: http status %s", resp.Status)
		return
	}
	jdec := json.NewDecoder(resp.Body)
	err = jdec.Decode(&nst)
	if err != nil {
		return
	}

	return
}

func (b billingHTTP) GetVolumeTariff(ctx context.Context, tariffID string) (vt model.VolumeTariff, err error) {
	refURL := &url.URL{
		Path: "/volume_tariffs",
	}
	req, _ := http.NewRequest("POST", b.u.ResolveReference(refURL).String(), nil)
	resp, err := b.c.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return
	}
	if resp.StatusCode/100 != 2 {
		err = fmt.Errorf("billing: http status %s", resp.Status)
		return
	}
	jdec := json.NewDecoder(resp.Body)
	err = jdec.Decode(&vt)
	if err != nil {
		return
	}

	return
}

func (b billingHTTP) String() string {
	return fmt.Sprintf("billing http client: url=%v", b.u)
}

type billingStub struct{}

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

func (billingStub) GetNamespaceTariff(ctx context.Context, tariffID string) (model.NamespaceTariff, error) {
	logrus.Infof("billingStub.GetNamespaceTariff tariffID=%s", tariffID)
	for _, ns := range fakeNSTariffs {
		if ns.ID != nil && ns.ID.String() == tariffID {
			return ns, nil
		}
	}
	return model.NamespaceTariff{}, BillingError{ErrCode: "NOT_FOUND", Cause: nil, error: "no such namespace tariff"}
}

func (billingStub) GetVolumeTariff(ctx context.Context, tariffID string) (model.VolumeTariff, error) {
	logrus.Infof("billingStub.GetVolumeTariff tariffID=%s", tariffID)
	for _, v := range fakeVolumeTariffs {
		if v.ID != nil && v.ID.String() == tariffID {
			return v, nil
		}
	}
	return model.VolumeTariff{}, BillingError{ErrCode: "NOT_FOUND", Cause: nil, error: "no such volume tariff"}
}

func (billingStub) String() string {
	return "billing service dummy"
}

type BillingError struct {
	ErrCode string
	Cause   error
	error   string
}

func (e BillingError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.ErrCode, e.error, e.Cause)
	} else {
		return fmt.Sprintf("%s: %s", e.ErrCode, e.error)
	}
}

func (e BillingError) IsAlreadySubscribed() bool {
	if e.ErrCode == "ALREADY_SUBSCRIBED" {
		return true
	}
	return false
}

func (e BillingError) IsAlreadyUnsubscribed() bool {
	if e.ErrCode == "NOT_SUBSCRIBED" {
		return true
	}
	return false
}
