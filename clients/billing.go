package clients

import (
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"

	"context"

	btypes "git.containerum.net/ch/json-types/billing"
	"git.containerum.net/ch/json-types/errors"
	rstypes "git.containerum.net/ch/json-types/resource-service"
)

// Billing is an interface to billing service
type Billing interface {
	Subscribe(ctx context.Context, userID string, resource rstypes.Resource, resourceKind rstypes.Kind) error
	Unsubscribe(ctx context.Context, userID string, resource rstypes.Resource) error

	GetNamespaceTariff(ctx context.Context, tariffID string) (btypes.NamespaceTariff, error)
	GetVolumeTariff(ctx context.Context, tariffID string) (btypes.VolumeTariff, error)

	//ActivateNamespaceTariff(ctx context.Context, ...)
	//ActivateVolumeTariff(ctx, ...)
}

// Data for dummy client

type dummyBillingClient struct {
	log *logrus.Entry
}

var fakeNSData = `
[
  {
    "id": "3c9d98af-ef8c-4486-ba28-01d83bdd2ddd",
    "tariff_id": "f3091cc9-6dc3-470e-ac54-84defe011111",
    "created_at": "2017-12-26T13:53:56Z",
    "cpu_limit": 500,
    "memory_limit": 512,
    "traffic": 20,
    "traffic_price": 0.333,
    "external_services": 2,
    "internal_services": 5,
    "is_active": true,
    "is_public": true,
    "price": 0
  },
  {
    "id": "2f7f294d-3f53-4b10-94e2-e7411570d9a7",
    "tariff_id": "4563e8c1-fb41-416a-9798-e949a2616260",
    "created_at": "2017-12-26T13:57:45Z",
    "cpu_limit": 900,
    "memory_limit": 1024,
    "traffic": 50,
    "traffic_price": 0.5,
    "external_services": 10,
    "internal_services": 20,
    "is_active": true,
    "is_public": true,
    "price": 0
  }
]
`

var fakeVolumeData = `
[
  {
    "id": "cc2ac926-1ead-4ee6-9218-ee64d92fca2a",
    "tariff_id": "15348470-e98f-4da0-8d2e-8c65e15d6eeb",
    "created_at": "2017-12-27T07:55:22Z",
    "storage_limit": 1,
    "replicas_limit": 1,
    "is_persistent": false,
    "is_active": true,
    "is_public": true,
    "price": 0
  },
  {
    "id": "f853e3f9-1752-42a7-ab07-0ef82cd8e918",
    "tariff_id": "11a35f90-c343-4fc1-a966-381f75568036",
    "created_at": "2017-12-27T07:55:22Z",
    "storage_limit": 2,
    "replicas_limit": 1,
    "is_persistent": false,
    "is_active": true,
    "is_public": true,
    "price": 0
  }
]
`

var (
	fakeNSTariffs     []btypes.NamespaceTariff
	fakeVolumeTariffs []btypes.VolumeTariff
)

func init() {
	var err error
	err = jsoniter.Unmarshal([]byte(fakeNSData), &fakeNSTariffs)
	if err != nil {
		panic(err)
	}
	err = jsoniter.Unmarshal([]byte(fakeVolumeData), &fakeVolumeTariffs)
	if err != nil {
		panic(err)
	}
}

// NewDummyBilling creates a dummy billing service client. It does nothing but logs actions.
func NewDummyBillingClient() Billing {
	return dummyBillingClient{
		log: logrus.WithField("component", "billing_dummy"),
	}
}

func (b dummyBillingClient) Subscribe(ctx context.Context, userID string, resource rstypes.Resource, resourceKind rstypes.Kind) error {
	b.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"tariff_id":   resource.TariffID,
		"resource_id": resource.ID,
		"kind":        resourceKind,
	}).Infoln("subscribing")
	return nil
}

func (b dummyBillingClient) Unsubscribe(ctx context.Context, userID string, resource rstypes.Resource) error {
	b.log.WithFields(logrus.Fields{
		"user_id":     userID,
		"resource_id": resource.ID,
	}).Infoln("unsubscribing")
	return nil
}

func (b dummyBillingClient) GetNamespaceTariff(ctx context.Context, tariffID string) (btypes.NamespaceTariff, error) {
	b.log.WithField("tariff_id", tariffID).Infoln("get namespace tariff")
	for _, nsTariff := range fakeNSTariffs {
		if nsTariff.ID != "" && nsTariff.ID == tariffID {
			return nsTariff, nil
		}
	}
	return btypes.NamespaceTariff{}, errors.New("no such namespace tariff")
}

func (b dummyBillingClient) GetVolumeTariff(ctx context.Context, tariffID string) (btypes.VolumeTariff, error) {
	b.log.WithField("tariff_id", tariffID).Infoln("get volume tariff")
	for _, volumeTariff := range fakeVolumeTariffs {
		if volumeTariff.ID != "" && volumeTariff.ID == tariffID {
			return volumeTariff, nil
		}
	}
	return btypes.VolumeTariff{}, errors.New("no such volume tariff")
}

func (b dummyBillingClient) String() string {
	return "billing service dummy"
}
