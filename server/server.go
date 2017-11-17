package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"bitbucket.org/exonch/resource-service/server/model"
	"bitbucket.org/exonch/resource-service/server/other"
	"bitbucket.org/exonch/resource-service/util/cache"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

type ResourceSvcInterface interface {
	CreateNamespace(ctx context.Context, userID, nsLabel, tariffID string, adminAction bool) error
	DeleteNamespace(ctx context.Context, userID, nsLabel string) error
	ListNamespaces(ctx context.Context, userID string, adminAction bool) ([]Namespace, error)
	GetNamespace(ctx context.Context, userID, nsLabel string, adminAction bool) (Namespace, error)
	ChangeAccessToNamespace(ownerUserID, nsLabel string, otherUserID, accessLevel string) error
	LockAccessToNamespace(ownerUserID, nsLabel string, lockState bool) error

	//CreateExtService(ctx context.Context, userID, svLabel string, adminAction bool) (Service, error)
	//DeleteExtService(ctx context.Context, userID, svLabel string, adminAction bool) error
	//ListExtServices(ctx context.Context, userID string, adminAction bool) error
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
	rm.billing = other.NewBillingStub()
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

	err = rm.db.namespaceCreate(resourceUUID, nsLabel, tariff)
	if err != nil {
		return newError("database: creating namespace: %v", err)
	}

	err = rm.db.permCreateOwner("Namespace", resourceUUID, userUUID)
	if err != nil {
		return newError("database: creating permission: %v", err)
	}

	ns, err := rm.db.namespaceGet(userUUID, nsLabel)
	if err != nil {
		return newError("database: getting namespace: %v", err)
	}

	ctx, cancelf := context.WithCancel(ctx)
	waitch := make(chan struct{})
	go func() {
		errKube = rm.kube.CreateNamespace(ctx, resourceUUID.String(), uint(*tariff.CpuLimit), uint(*tariff.MemoryLimit),
			nsLabel, "owner")
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

	if errKube != nil || errBilling != nil {
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
	} else {
		go func() {
			if err := rm.mailer.SendNamespaceCreated(userID, nsLabel); err != nil {
				logrus.Warnf("mailer error: %v", err)
			}
		}()
		go func() {
			if err := rm.authsvc.UpdateUserAccess(userID); err != nil {
				logrus.Warnf("auth error: %v", err)
			}
		}()
	}

	return nil
}

func (rs *ResourceSvc) DeleteNamespace(ctx context.Context, userID, nsLabel string) error {
	var err error
	var userUUID uuid.UUID

	userUUID, err = uuid.FromString(userID)
	if err != nil {
		return newBadInputError("invalid user id: %v", err)
	}

	ns, err := rs.db.namespaceGet(userUUID, nsLabel)

	perm, err := rs.permFetch(
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

func (rs *ResourceSvc) GetNamespace(ctx context.Context, userID, nsLabel string, adminAction bool) (ns Namespace, err error) {
	userUUID, err := uuid.FromString(userID)
	if err != nil {
		err = newBadInputError("invalid user id %q: %v", userID, err)
		return
	}
	ns, err = rs.db.namespaceGet(userUUID, nsLabel)
	if err != nil {
		err = newError("database: %v", err)
		return
	}
	if !adminAction {
		ns.ID = nil
	}
	return
}

// ChangeAccessToNamespace adds or removes access to ownerUserID's resource for
// otherUserID.
func (rs *ResourceSvc) ChangeAccessToNamespace(ownerUserID, nsLabel string, otherUserID, permOther string) error {
	ownerUserUUID, err := uuid.FromString(ownerUserID)
	if err != nil {
		return newBadInputError("invalid ownerUserID: %v", err)
	}
	otherUserUUID, err := uuid.FromString(otherUserID)
	if err != nil {
		return newBadInputError("invalid otherUserID: %v", err)
	}
	ns, err := rs.db.namespaceGet(ownerUserUUID, nsLabel)
	if err != nil {
		switch err.(type) {
		case Error:
			return err
		default:
			return newError("database: %v", err)
		}
	}

	perm, err := rs.db.permFetch(*ns.ID, ownerUserUUID)
	if err != nil {
		return newError("database: %v", err)
	}
	if permCheck(perm, "owner") == false {
		return newPermissionError("permission denied")
	}

	if err = rs.db.permSetOtherUser(*ns.ID, otherUserUUID, permOther); err != nil {
		switch err.(type) {
		case Error, BadInputError:
			return err
		default:
			return newError("database: %v", err)
		}
	}

	return nil
}

func (rs *ResourceSvc) LockAccessToNamespace(ownerUserID, nsLabel string, lockState bool) error {
	ownerUserUUID, err := uuid.FromString(ownerUserID)
	if err != nil {
		return newBadInputError("invalid owner user id: %v", err)
	}
	ns, err := rs.db.namespaceGet(ownerUserUUID, nsLabel)
	if err != nil {
		switch err.(type) {
		case Error:
			return err
		default:
			return newError("database: %v", err)
		}
	}
	perm, err := rs.db.permFetch(*ns.ID, ownerUserUUID)
	if err != nil {
		return newError("database: %v", err)
	}
	if permCheck(perm, "owner") == false {
		return newPermissionError("permission denied")
	}

	if err = rs.db.permSetLimited(*ns.ID, lockState); err != nil {
		return newError("database: %v", err)
	}

	return nil
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
