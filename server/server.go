package server

import (
	"net/http"
	"net/url"

	"bitbucket.org/exonch/resource-manager/server/model"
	"bitbucket.org/exonch/resource-manager/server/other"

	"github.com/sirupsen/logrus"
)

type ResourceManagerInterface interface {
	CreateNamespace(ctx context.Context, userID, nsLabel, tariffID string) error
}

type ResourceManager struct {
	billing   other.Billing
	mailer    other.Mailer
	kube      other.Kube
	volumesvc other.VolumeSvc
}

// TODO
//var _ ResourceManagerInterface = &ResourceManager{}

// TODO: arguments must be the respective interfaces
// from the "other" module.
func (rm *ResourceManager) Initialize(b, k, m, v *url.URL) {
	rm.billing = other.NewBillingHTTP(http.DefaultClient, b)
	rm.kube = other.NewKubeHTTP(http.DefaultClient, k)
	rm.mailer = other.NewMailerHTTP(http.DefaultClient, m)
	rm.volumesvc = other.NewVolumeSvcHTTP(http.DefaultClient, v)
}

func (rm *ResourceManager) CreateNamespace(ctx context.Context, userID, nsLabel, tariffID string) error {
	var resourceID string = rm.newResourceID("namespace", userID, nsLabel)
	var billingID string
	var errKube, errBilling, errMailer string
	var allSuccess bool

	// * get cpuQuota, memQuota somehow

	var rollbackID string = uuid.NewV4().String() + "-" + ctx.Value("request-id").(string)
	rm.rollbackQueueNew(rollbackID)
	ctx, cancelf := context.WithCancel(ctx)
	ctx = context.WithTimeout(ctx, time.Second*2)

	go func() {
		errKube = rm.kube.CreateNamespace(ctx, resourceID, cpuQuota, memQuota)
		if errKube != nil {
			cancelf()
		} else {
			rm.rollbackQueueAdd(rollbackID, "namespace", resourceID)
		}
	}()
	go func() {
		billingID, errBilling = rm.billing.Subscribe(ctx, userID, tariffID, resourceID)
		if errBilling != nil {
			cancelf()
		} else {
			rm.rollbackQueueAdd(rollbackID, "billing-sub", resourceID)
		}
	}()

	if errKube == nil && errBilling == nil {
		rm.rollbackQueueCancel(rollbackID)
	} else {
		rm.rollbackQueueExecute(rollbackID)
	}
}

func (rm *ResourceManager) newResourceID(seed ...string) {
	// hash of strigs.Join(seed, ",") concat with some constant salt (probably from DB)
}
