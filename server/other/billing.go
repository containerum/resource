package other

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"

	"git.containerum.net/ch/resource-service/server/model"

	uuid "github.com/satori/go.uuid"
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

type billingStub struct {}

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
		Price:            new(big.Rat),
		VV:               new(model.VolumeTariff),
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
	nt.Price.SetFrac64(11, 10)
	{
		*nt.VV = model.VolumeTariff{
			ID:            new(uuid.UUID),
			TariffID:      new(uuid.UUID),
			StorageLimit:  new(int),
			ReplicasLimit: new(int),
			IsActive:      new(bool),
			IsPublic:      new(bool),
			Price:         new(big.Rat),
		}
		*nt.VV.ID = uuid.FromStringOrNil("e1eab7c4-31d4-4eba-9daf-5a4e77413594")
		*nt.VV.TariffID = uuid.FromStringOrNil(tariffID[:len(tariffID)-4]+"1112")
		*nt.VV.StorageLimit = 2
		*nt.VV.ReplicasLimit = 1
		*nt.VV.IsActive = true
		*nt.VV.IsPublic = true
	}
	return nt, nil
}

func (billingStub) GetVolumeTariff(ctx context.Context, tariffID string) (model.VolumeTariff, error) {
	logrus.Infof("Billing.GetVolumeTariff tariffID=%s", tariffID)
	vt := model.VolumeTariff{
		ID:            new(uuid.UUID),
		TariffID:      new(uuid.UUID),
		StorageLimit:  new(int),
		ReplicasLimit: new(int),
		IsActive:      new(bool),
		IsPublic:      new(bool),
		Price:         new(big.Rat),
		IsPersistent:  new(bool),
	}
	*vt.ID = someUUID2
	*vt.TariffID = uuid.FromStringOrNil(tariffID)
	*vt.StorageLimit = 5
	*vt.ReplicasLimit = 2
	*vt.IsActive = true
	*vt.IsPublic = true
	vt.Price.SetFrac64(9, 10)
	*vt.IsPersistent = false
	return vt, nil
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
		return fmt.Sprintf("%s: %s: %v", e.ErrCode, e.Error, e.Cause)
	} else {
		return fmt.Sprintf("%s: %s", e.ErrCode, e.Error)
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
