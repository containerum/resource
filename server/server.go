package server

import (
	"context"
	"database/sql"
	//"fmt"
	"net/url"
	//"strings"
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
	//RenameNamespace(ctx context.Context, userID, labelOld, labelNew string) error
	ListNamespaces(ctx context.Context, userID string, adminAction bool) ([]Namespace, error)
	GetNamespace(ctx context.Context, userID, nsLabel string, adminAction bool) (Namespace, error)
	ChangeAccess(ownerUserID, resKind, resLabel string, otherUserID, accessLevel string) error
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
	rm.authsvc = other.NewAuthSvcStub()
	rm.billing = other.NewBillingStub()
	rm.kube = other.NewKubeStub()
	rm.mailer = other.NewMailerStub()
	rm.volumesvc = other.NewVolumeSvcStub()

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

func (rs *ResourceSvc) CreateNamespace(ctx context.Context, userID, nsLabel, tariffID string, adminAction bool) error {
	var err error
	var nsUUID, userUUID uuid.UUID
	var tariff model.NamespaceTariff

	var rbNamespaceCreation, rbNamespaceDB bool

	userUUID, err = uuid.FromString(userID)
	if err != nil {
		return newBadInputError("invalid user ID, not a UUID: %v", err)
	}

	tariff, err = rs.getNSTariff(ctx, tariffID)
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

	nsUUID, err = rs.db.namespaceCreate(tariff)
	if err != nil {
		return newError("database: creating namespace: %v", err)
	}
	rbNamespaceDB = true

	defer func() {
		if rbNamespaceDB {
			rs.db.namespaceDelete(nsUUID)
		}
	}()

	err = rs.db.permCreateOwner("Namespace", nsUUID, nsLabel, userUUID)
	if err != nil {
		return newError("database: creating permission: %v", err)
	}

	err = rs.kube.CreateNamespace(ctx, nsUUID.String(), uint(*tariff.CpuLimit), uint(*tariff.MemoryLimit), nsLabel, "owner")
	if err != nil {
		return newOtherServiceError("cannot create namespace at kube api: %v", err)
	}
	rbNamespaceCreation = true

	defer func() {
		if rbNamespaceCreation {
			rs.kube.DeleteNamespace(context.Background(), nsUUID.String())
			if _, permUUID, _, err := rs.db.permGet(userUUID, "Namespace", nsLabel); err == nil {
				rs.db.permDelete(permUUID)
			}
			rs.db.namespaceDelete(nsUUID)
		}
	}()

	err = rs.billing.Subscribe(ctx, userID, tariffID, nsUUID.String())
	if err != nil {
		return newOtherServiceError("cannot subscribe user to tariff: %v", err)
	}
	rbNamespaceCreation = false
	rbNamespaceDB = false

	go func() {
		if err := rs.mailer.SendNamespaceCreated(userID, nsLabel); err != nil {
			logrus.Warnf("mailer error: %v", err)
		}
	}()
	go func() {
		if err := rs.authsvc.UpdateUserAccess(userID); err != nil {
			logrus.Warnf("auth error: %v", err)
		}
	}()

	return nil
}

func (rs *ResourceSvc) DeleteNamespace(ctx context.Context, userID, nsLabel string) error {
	var err error
	var userUUID uuid.UUID

	userUUID, err = uuid.FromString(userID)
	if err != nil {
		return newBadInputError("invalid user id: %v", err)
	}

	nsUUID, permUUID, lvl, err := rs.db.permGet(userUUID, "Namespace", nsLabel)
	if err != nil {
		if err == NoSuchResource {
			return err
		}
		return newError("database: fetch access level: %v", err)
	}

	if !permCheck(lvl, "delete") {
		return newPermissionError("permission denied")
	}

	_, err = rs.db.namespaceGet(userUUID, nsLabel)
	if err != nil {
		if err == NoSuchResource {
			return err
		}
		return newError("database: %v", err)
	}

	err = rs.db.permDelete(permUUID)
	if err != nil {
		return newError("database: delete access %s: %v", permUUID, err)
	}

	err = rs.billing.Unsubscribe(ctx, userID, nsUUID.String())
	if err != nil {
		return newOtherServiceError("cannot unsubscribe from billing: %v", err)
	}

	err = rs.authsvc.UpdateUserAccess(userID)
	if err != nil {
		logrus.Warnf("auth svc error: update user access: %v", err)
	}

	err = rs.kube.DeleteNamespace(ctx, nsUUID.String())
	if err != nil {
		logrus.Warnf("kube api error: delete namespace: %v", err)
	}

	err = rs.db.namespaceDelete(nsUUID)
	if err != nil {
		logrus.Errorf("database: %v", err)
	}

	return nil
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

// ChangeAccess adds or removes access to ownerUserID's resource for otherUserID.
func (rs *ResourceSvc) ChangeAccess(ownerUserID, resKind, resLabel string, otherUserID, permOther string) error {
	ownerUserUUID, err := uuid.FromString(ownerUserID)
	if err != nil {
		return newBadInputError("invalid ownerUserID: %v", err)
	}
	otherUserUUID, err := uuid.FromString(otherUserID)
	if err != nil {
		return newBadInputError("invalid otherUserID: %v", err)
	}

	resUUID, _, lvl, err := rs.db.permGet(ownerUserUUID, resKind, resLabel)
	if err != nil {
		if err == NoSuchResource {
			return err
		}
		return newError("database: fetch access: %v", err)
	}
	if lvl != "owner" {
		return newPermissionError("permission denied")
	}

	_, _, permUUID, lvl, err := rs.db.permGetByResourceID(resUUID, otherUserUUID)
	if err != nil {
		err = rs.db.permGrant(resUUID, resLabel, ownerUserUUID, otherUserUUID, permOther)
		if err != nil {
			return newError("database: %v", err)
		}
	} else {
		if permOther == "none" {
			err = rs.db.permDelete(permUUID)
			if err != nil {
				return newError("deleting access level: %v", err)
			}
		} else {
			err = rs.db.permSetLevel(permUUID, permOther)
			if err != nil {
				return newError("setting access level: %v", err)
			}
		}
	}

	err = rs.authsvc.UpdateUserAccess(otherUserID)
	if err != nil {
		logrus.Warnf("auth svc error: failed to update user access: %v", err)
	}

	return nil
}

func (rs *ResourceSvc) LockAccessToNamespace(ownerUserID, nsLabel string, lockState bool) error {
	ownerUserUUID, err := uuid.FromString(ownerUserID)
	if err != nil {
		return newBadInputError("invalid owner user id: %v", err)
	}

	_, permUUID, lvl, err := rs.db.permGet(ownerUserUUID, "Namespace", nsLabel)
	if err != nil {
		return newError("database: get access level: %v", err)
	}
	if lvl != "owner" {
		return newPermissionError("permission denied")
	}

	if err = rs.db.permSetLimited(permUUID, lockState); err != nil {
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
