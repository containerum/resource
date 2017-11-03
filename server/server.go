package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"bitbucket.org/exonch/resource-service/server/model"
	"bitbucket.org/exonch/resource-service/server/other"
	"bitbucket.org/exonch/resource-service/util/cache"

	"github.com/sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

var _ = logrus.StandardLogger()

type ResourceManagerInterface interface {
	CreateNamespace(ctx context.Context, userID, nsLabel, tariffID string, adminAction bool) error
	DeleteNamespace(ctx context.Context, userID, nsLabel string) error
	ListNamespaces(ctx context.Context, userID string, adminAction bool) ([]Namespace, error)
}

type ResourceManager struct {
	billing   other.Billing
	mailer    other.Mailer
	kube      other.Kube
	volumesvc other.VolumeSvc

	db          resourceManagerDB
	tariffCache cache.Cache
}

// TODO
var _ ResourceManagerInterface = &ResourceManager{}

// TODO: arguments must be the respective interfaces
// from the "other" module.
func (rm *ResourceManager) Initialize(b, k, m, v *url.URL, dbDSN string) error {
	rm.billing = other.NewBilling(b)
	rm.kube = other.NewKube(k)
	rm.mailer = other.NewMailer(m)
	rm.volumesvc = other.NewVolumeSvc(v)

	var err error
	rm.db.con, err = sql.Open("postgres", dbDSN)
	if err != nil {
		return err
	}
	if err = rm.db.initialize(); err != nil {
		return err
	}
	return nil
}

func (rm *ResourceManager) CreateNamespace(ctx context.Context, userID, nsLabel, tariffID string, adminAction bool) error {
	var err error
	var resourceID string = rm.newResourceID("namespace", userID, nsLabel)
	//var billingID string
	var errKube, errBilling error
	var tariff model.NamespaceTariff

	tariff, err = rm.getNSTariff(ctx, tariffID)
	if err != nil {
		return newError("cannot get tariff quota: %v", err)
	}

	if !*tariff.IsActive {
		return newPermissionError("cannot subscribe to inactive tariff")
	}
	if !adminAction {
		if !*tariff.IsPublic {
			return newPermissionError("tariff unavailable")
		}
	}

	ctx, cancelf := context.WithCancel(ctx)
	//ctx = context.WithTimeout(ctx, time.Second*2)

	waitch := make(chan struct{})
	{
		errKube = rm.kube.CreateNamespace(ctx, resourceID, uint(*tariff.CpuLimit), uint(*tariff.MemoryLimit))
		if errKube != nil {
			cancelf()
		}
		waitch <- struct{}{}
	}
	{
		logrus.Warnf("would subscribe user %q to tariff %q", userID, tariffID)
		//errBilling = rm.billing.Subscribe(ctx, userID, tariffID, resourceID)
		if errBilling != nil {
			cancelf()
		}
		waitch <- struct{}{}
	}
	<-waitch
	<-waitch

	go func() {
		if errKube == nil {
			rm.kube.DeleteNamespace(context.Background(), resourceID)
		}
		if errBilling == nil {
			logrus.Warnf("would unsubscribe user %q from tariff %q", userID, tariffID)
			//rm.billing.Unsubscribe(context.Background(), userID, billingID)
		}
	}()

	var errstr string
	if errKube != nil {
		errstr = errstr + fmt.Sprintf("kube api error: %v; ", err)
	}
	if errBilling != nil {
		errstr = errstr + fmt.Sprintf("billing error: %v; ", errBilling)
	}
	if errstr != "" {
		err = newOtherServiceError("%s", errstr)
	}
	return err
}

func (rm *ResourceManager) DeleteNamespace(ctx context.Context, userID, nsLabel string) error {
	return fmt.Errorf("not implemented")
}

func (rm *ResourceManager) ListNamespaces(ctx context.Context, userID string, adminAction bool) ([]Namespace, error) {
	userUUID, err := uuid.FromString(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id %q: %v", userID, err)
	}
	namespaces, err := rm.db.namespaceList(&userUUID)
	if err != nil {
		return nil, fmt.Errorf("database: %v", err)
	}
	if !adminAction {
		for i := range namespaces {
			namespaces[i].ID = nil
		}
	}
	return namespaces, nil
}

func (rm *ResourceManager) newResourceID(seeds ...string) string {
	in := []byte{0xAB, 0xBA}
	for i := range seeds {
		in = append(in, []byte(seeds[i])...)
	}
	h := sha256.Sum256(in)
	return fmt.Sprintf("%x", h)
}

func (rm *ResourceManager) getNSTariff(ctx context.Context, id string) (t model.NamespaceTariff, err error) {
	if rm.tariffCache == nil {
		rm.tariffCache = cache.NewTimed(time.Second * 10)
	}

	if tmp, cached := rm.tariffCache.Get(id); cached && tmp != nil {
		t = tmp.(model.NamespaceTariff)
	} else {
		t, err = rm.billing.GetNamespaceTariff(ctx, id)
		if err != nil {
			return
		}
		rm.tariffCache.Set(id, t)
	}

	if t.CpuLimit == nil || t.MemoryLimit == nil {
		err = newError("malformed tariff in response: %#v", t)
		return
	}

	return
}
