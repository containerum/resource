package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"bitbucket.org/exonch/resource-manager/server/model"
	"bitbucket.org/exonch/resource-manager/server/other"
	"bitbucket.org/exonch/resource-manager/util/cache"

	"github.com/sirupsen/logrus"
	//uuid "github.com/satori/go.uuid"
)

var _ = logrus.StandardLogger()

type ResourceManagerInterface interface {
	CreateNamespace(ctx context.Context, userID, nsLabel, tariffID string) error
	//DeleteNamespace(ctx context.Context, userID, nsLabel string) error
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

func (rm *ResourceManager) CreateNamespace(ctx context.Context, userID, nsLabel, tariffID string) error {
	var err error
	var resourceID string = rm.newResourceID("namespace", userID, nsLabel)
	var billingID string
	var errKube, errBilling error
	var cpuQuota, memQuota uint

	cpuQuota, memQuota, err = rm.getNSTariffQuota(ctx, tariffID)
	if err != nil {
		return newError("cannot get tariff quota: %v", err)
	}

	ctx, cancelf := context.WithCancel(ctx)
	//ctx = context.WithTimeout(ctx, time.Second*2)

	waitch := make(chan struct{})
	go func() {
		errKube = rm.kube.CreateNamespace(ctx, resourceID, cpuQuota, memQuota)
		if errKube != nil {
			cancelf()
		}
		waitch <- struct{}{}
	}()
	go func() {
		billingID, errBilling = rm.billing.Subscribe(ctx /*, userID, tariffID, resourceID*/)
		if errBilling != nil {
			cancelf()
		}
		waitch <- struct{}{}
	}()
	<-waitch
	<-waitch

	go func() {
		if errKube == nil {
			rm.kube.DeleteNamespace(context.Background(), resourceID)
		}
		if errBilling == nil {
			rm.billing.Unsubscribe(context.Background() /*, userID, billingID*/)
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

func (rm *ResourceManager) newResourceID(seed ...string) string {
	// hash of strigs.Join(seed, ",") concat with some constant salt (probably from DB)
	return ""
}

func (rm *ResourceManager) getNSTariffQuota(ctx context.Context, id string) (cpu, mem uint, err error) {
	var nstariff model.NamespaceTariff

	if rm.tariffCache == nil {
		rm.tariffCache = cache.NewTimed(time.Second * 10)
	}

	if tmp, cached := rm.tariffCache.Get(id); cached && tmp != nil {
		nstariff = tmp.(model.NamespaceTariff)
	} else {
		nstariff, err = rm.billing.GetNamespaceTariff(ctx, id)
		if err != nil {
			return
		}
		rm.tariffCache.Set(id, nstariff)
	}

	if nstariff.CpuLimit == nil || nstariff.MemoryLimit == nil {
		err = newError("malformed tariff in response: %#v", nstariff)
		return
	}

	cpu = uint(*nstariff.CpuLimit)
	mem = uint(*nstariff.MemoryLimit)
	return
}
