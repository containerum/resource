package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"
	"strings"

	"bitbucket.org/exonch/resource-service/server/model"
	"bitbucket.org/exonch/resource-service/server/other"
	"bitbucket.org/exonch/resource-service/util/cache"

	"github.com/sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

var _ = logrus.StandardLogger()

type ResourceSvcInterface interface {
	CreateNamespace(ctx context.Context, userID, nsLabel, tariffID string, adminAction bool) error
	DeleteNamespace(ctx context.Context, userID, nsLabel string) error
	ListNamespaces(ctx context.Context, userID string, adminAction bool) ([]Namespace, error)
}

type ResourceSvc struct {
	authsvc   other.AuthSvc
	billing   other.Billing
	mailer    other.Mailer
	kube      other.Kube
	volumesvc other.VolumeSvc

	db          resourceSvcDB
	tariffCache cache.Cache
}

// TODO
var _ ResourceSvcInterface = &ResourceSvc{}

// TODO: arguments must be the respective interfaces
// from the "other" module.
func (rm *ResourceSvc) Initialize(a, b, k, m, v *url.URL, dbDSN string) error {
	rm.authsvc = other.NewAuthSvcStub(a)
	rm.billing = other.NewBillingStub(b)
	rm.kube = other.NewKube(k)
	rm.mailer = other.NewMailerHTTP(m)
	rm.volumesvc = other.NewVolumeSvcHTTP(v)

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

func (rm *ResourceSvc) CreateNamespace(ctx context.Context, userID, nsLabel, tariffID string, adminAction bool) error {
	var err error
	var resourceUUID uuid.UUID = uuid.NewV4()
	var userUUID uuid.UUID
	var errKube, errBilling error
	var tariff model.NamespaceTariff

	userUUID, err = uuid.FromString(userID)
	if err != nil {
		return newError("invalid user ID, not a UUID: %v", err)
	}

	tariff, err = rm.getNSTariff(ctx, tariffID)
	if err != nil {
		return newOtherServiceError("cannot get tariff quota: %v", err)
	}

	if !*tariff.IsActive {
		return newPermissionError("cannot subscribe to inactive tariff")
	}
	if !adminAction {
		if !*tariff.IsPublic {
			return newPermissionError("tariff unavailable")
		}
	}

	err = rm.db.permCreate("Namespace", resourceUUID, userUUID)
	if err != nil {
		return newError("database error")
	}

	ctx, cancelf := context.WithCancel(ctx)
	waitch := make(chan struct{})
	go func() {
		errKube = rm.kube.CreateNamespace(ctx, resourceUUID.String(), uint(*tariff.CpuLimit), uint(*tariff.MemoryLimit))
		if errKube != nil {
			cancelf()
		}
		waitch <- struct{}{}
	}()
	go func() {
		errBilling = rm.billing.Subscribe(ctx, userID, tariffID, resourceUUID.String())
		if errBilling != nil {
			cancelf()
		}
		waitch <- struct{}{}
	}()
	<-waitch
	<-waitch

	go func() {
		if errKube == nil {
			rm.kube.DeleteNamespace(context.Background(), resourceUUID.String())
		}
		if errBilling == nil {
			logrus.Warnf("would unsubscribe user %q from tariff %q", userID, tariffID)
			rm.billing.Unsubscribe(context.Background(), userID, resourceUUID.String())
		}
	}()

	var errs []string
	if errKube != nil {
		errs = append(errs, fmt.Sprintf("kube api error: %v", errKube))
	}
	if errBilling != nil {
		errs = append(errs, fmt.Sprintf("billing error: %v", errBilling))
	}
	if len(errs) > 0 {
		err = newOtherServiceError("%s", strings.Join(errs, "; "))
		return err
	}

	go rm.mailer.SendNamespaceCreated(model.User{ID: &userID}, nsLabel, model.Tariff{ID: &tariffID})
	go rm.authsvc.UpdateUserAccess(userID)
	return nil
}

func (rm *ResourceSvc) DeleteNamespace(ctx context.Context, userID, nsLabel string) error {
	return fmt.Errorf("not implemented")
}

func (rm *ResourceSvc) ListNamespaces(ctx context.Context, userID string, adminAction bool) ([]Namespace, error) {
	userUUID, err := uuid.FromString(userID)
	if err != nil {
		return nil, newError("invalid user id %q: %v", userID, err)
	}
	namespaces, err := rm.db.namespaceList(&userUUID)
	if err != nil {
		return nil, newError("database: %v", err)
	}
	if !adminAction {
		for i := range namespaces {
			namespaces[i].ID = nil
		}
	}
	return namespaces, nil
}

func (rm *ResourceSvc) getNSTariff(ctx context.Context, id string) (t model.NamespaceTariff, err error) {
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
