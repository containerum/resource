package other

import (
	"context"
	"fmt"
	"net/url"

	rstypes "git.containerum.net/ch/json-types/resource-service"

	"git.containerum.net/ch/json-types/errors"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

type Billing interface {
	Subscribe(ctx context.Context, userID, tariffID, resourceID string) error
	Unsubscribe(ctx context.Context, userID, resourceID string) error

	GetNamespaceTariff(ctx context.Context, id string) (rstypes.NamespaceTariff, error)
	GetVolumeTariff(ctx context.Context, id string) (rstypes.VolumeTariff, error)
}

type billingHTTP struct {
	client *resty.Client
	log    *logrus.Entry
}

func NewBillingHTTP(u *url.URL) Billing {
	log := logrus.WithField("component", "billing_client")
	client := resty.New().
		SetHostURL(u.String()).
		SetLogger(log.WriterLevel(logrus.DebugLevel)).
		SetDebug(true).
		SetError(errors.Error{})
	client.JSONMarshal = jsoniter.Marshal
	client.JSONUnmarshal = jsoniter.Unmarshal
	return &billingHTTP{
		client: client,
		log:    log,
	}
}

func (b billingHTTP) Subscribe(ctx context.Context, userID, tariffID, resourceID string) error {
	b.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"tariff_id":   tariffID,
		"resource_id": resourceID,
	}).Debugln("subscribing")
	resp, err := b.client.R().SetBody(map[string]string{ // TODO: replace with type from 'json-types'
		//"tariff_label":   tariffLabel,
		//"resource_type":  resType,
		//"resource_label": resLabel,
		"resource_id": resourceID,
		"user_id":     userID,
	}).Post("/user/subscribe")
	if err != nil {
		return err
	}
	if resp.Error() != nil {
		return resp.Error().(*errors.Error)
	}
	return nil
}

func (b billingHTTP) Unsubscribe(ctx context.Context, userID, resourceID string) error {
	b.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"resource_id": resourceID,
	}).Debugln("unsubscribing")
	resp, err := b.client.R().SetBody(map[string]string{
		"resource_id": resourceID,
		"user_id":     userID,
	}).Post("/user/unsubscribe")
	if err != nil {
		return err
	}
	if resp.Error() != nil {
		return resp.Error().(*errors.Error)
	}
	return nil
}

func (b billingHTTP) GetNamespaceTariff(ctx context.Context, tariffID string) (nst rstypes.NamespaceTariff, err error) {
	b.log.WithField("tariff_id", tariffID).Debugln("get namespace tariff")
	resp, err := b.client.R().SetResult(&nst).Get("/namespace_tariffs/" + tariffID)
	if err != nil {
		return
	}
	if resp.Error() != nil {
		err = resp.Error().(*errors.Error)
	}
	return
}

func (b billingHTTP) GetVolumeTariff(ctx context.Context, tariffID string) (vt rstypes.VolumeTariff, err error) {
	b.log.WithField("tariff_id", tariffID).Debugln("get volume tariff")
	resp, err := b.client.R().SetResult(&vt).Get("/volume_tariffs/" + tariffID)
	if err != nil {
		return
	}
	if resp.Error() != nil {
		err = resp.Error().(*errors.Error)
	}
	return
}

func (b billingHTTP) String() string {
	return fmt.Sprintf("billing http client: url=%v", b.client.HostURL)
}

type billingStub struct {
	log *logrus.Entry
}

func NewBillingStub() Billing {
	return billingStub{log: logrus.WithField("component", "billing_stub")}
}

func (b billingStub) Subscribe(ctx context.Context, userID, tariffID, resourceID string) error {
	b.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"tariff_id":   tariffID,
		"resource_id": resourceID,
	}).Debugln("subscribing")
	return nil
}

func (b billingStub) Unsubscribe(ctx context.Context, userID, resourceID string) error {
	b.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"resource_id": resourceID,
	}).Debugln("unsubscribing")
	return nil
}

func (b billingStub) GetNamespaceTariff(ctx context.Context, tariffID string) (rstypes.NamespaceTariff, error) {
	b.log.WithField("tariff_id", tariffID).Debugln("get namespace tariff")
	for _, ns := range fakeNSTariffs {
		if ns.TariffID != "" && ns.TariffID == tariffID {
			return ns, nil
		}
	}
	return rstypes.NamespaceTariff{}, errors.New("no such namespace tariff")
}

func (b billingStub) GetVolumeTariff(ctx context.Context, tariffID string) (rstypes.VolumeTariff, error) {
	b.log.WithField("tariff_id", tariffID).Debugln("get volume tariff")
	for _, v := range fakeVolumeTariffs {
		if v.TariffID != "" && v.TariffID == tariffID {
			return v, nil
		}
	}
	return rstypes.VolumeTariff{}, errors.New("no such volume tariff")
}

func (b billingStub) String() string {
	return "billing service dummy"
}
