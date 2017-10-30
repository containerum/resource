package server

import (
	"context"
	"net/url"

	"bitbucket.org/exonch/resource-manager/server/model"
	"bitbucket.org/exonch/resource-manager/server/other"
	"bitbucket.org/exonch/resource-manager/util/cache"
	//"github.com/sirupsen/logrus"
)

type ResourceManagerInterface interface {
	CreateNamespace(ctx context.Context, userID, nsLabel, tariffID string) error
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
//var _ ResourceManagerInterface = &ResourceManager{}

// TODO: arguments must be the respective interfaces
// from the "other" module.
func (rm *ResourceManager) Initialize(b, k, m, v *url.URL, dbDSN string) error {
	rm.billing = other.NewBilling(b)
	rm.kube = other.NewKube(k)
	rm.mailer = other.NewMailer(m)
	rm.volumesvc = other.NewVolumeSvc(v)

	rm.db.con, err = sql.Open("postgres", dbDSN)
	if err != nil {
		return err
	}
	rm.db.initialize()
}

func (rm *ResourceManager) CreateNamespace(ctx context.Context, userID, nsLabel, tariffID string) error {
	var resourceID string = rm.newResourceID("namespace", userID, nsLabel)
	var billingID string
	var errKube, errBilling, errMailer string
	var cpuQuota, memQuota int

	cpuQuota, memQuota, error = rm.getTariffQuotaByID(ctx, tariffID)

	var rollbackID string = uuid.NewV4().String() + "-" + ctx.Value("request-id").(string)
	rm.rollbackQueueNew(rollbackID)
	ctx, cancelf := context.WithCancel(ctx)
	//ctx = context.WithTimeout(ctx, time.Second*2)

	waitch := make(chan struct{})
	go func() {
		errKube = rm.kube.CreateNamespace(ctx, resourceID, cpuQuota, memQuota)
		if errKube != nil {
			cancelf()
		} else {
			rm.rollbackQueueAdd(rollbackID, "namespace", resourceID)
		}
		waitch <- struct{}{}
	}()
	go func() {
		billingID, errBilling = rm.billing.Subscribe(ctx, userID, tariffID, resourceID)
		if errBilling != nil {
			cancelf()
		} else {
			rm.rollbackQueueAdd(rollbackID, "billing-sub", resourceID)
		}
		waitch <- struct{}{}
	}()
	<-waitch
	<-waitch

	if errKube == nil && errBilling == nil {
		go rm.rollbackQueueCancel(rollbackID)
	} else {
		go rm.rollbackQueueExecute(rollbackID)
	}
}

func (rm *ResourceManager) newResourceID(seed ...string) {
	// hash of strigs.Join(seed, ",") concat with some constant salt (probably from DB)
}

func (rm *ResourceManager) getTariffQuotaByID(ctx context.Context, id string) (cpu, mem int, err error) {
	var tariff model.Tariff

	if rm.tariffCache == nil {
		rm.tariffCache = cache.NewTimed(time.Second * 10)
	}

	if tmp, cached := rm.tariffCache.Get(id); cached && tmp != nil {
		tariff = tmp.(model.Tariff)
	} else {
		tariff, err = rm.billing.GetTariffByID(ctx, tariffID)
		if err != nil {
			return
		}
		rm.tariffCache.Set(id, tariff)
	}

	if tariff.CpuLimit == nil || tariff.MemoryLimit == nil {
		rm.tariffCache.Unset(id)
		return newError(fmt.Sprintf("malformed tariff in response: %#v", tariff))
	}

	cpu = *tariff.CpuLimit
	mem = *tariff.MemLimit
	return
}
