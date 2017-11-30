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
	LockAccess(ownerUserID, resKind, resLabel string, lockState bool) error

	CreateVolume(ctx context.Context, userID, vLabel, tariffID string, adminAction bool) error
	DeleteVolume(ctx context.Context, userID, vLabel string) error
	ListVolumes(ctx context.Context, userID string, adminAction bool) ([]Volume, error)
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
	rm.mailer = other.NewMailerHTTP(m)
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
	var nsUUID, userUUID, permUUID uuid.UUID
	var tariff model.NamespaceTariff

	var rbNamespaceCreation bool

	userUUID, err = uuid.FromString(userID)
	if err != nil {
		return newBadInputError("invalid user ID, not a UUID: %v", err)
	}

	tariff, err = rs.getNSTariff(ctx, tariffID)
	if err != nil {
		return newOtherServiceError("failed to get namespace tariff: %v", err)
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
	permUUID, err = rs.db.permCreateOwner("Namespace", nsUUID, nsLabel, userUUID)
	if err != nil {
		return newError("database: creating permission: %v", err)
	}

	err = rs.kube.CreateNamespace(ctx, nsUUID.String(), uint(*tariff.CpuLimit), uint(*tariff.MemoryLimit), nsLabel, "owner")
	if err != nil {
		return newOtherServiceError("kube api error: create namespace: %v", err)
	}
	rbNamespaceCreation = true
	defer func() {
		if rbNamespaceCreation {
			rs.kube.DeleteNamespace(context.Background(), nsUUID.String())
			rs.db.permDelete(permUUID)
			rs.db.namespaceDelete(nsUUID)
		}
	}()

	err = rs.billing.Subscribe(ctx, userID, tariffID, nsUUID.String())
	if err != nil {
		return newOtherServiceError("billing error: subscribe: %v", err)
	}
	rbNamespaceCreation = false

	go func() {
		if err := rs.mailer.SendNamespaceCreated(userID, nsLabel, tariff); err != nil {
			logrus.Warnf("mailer error: %v", err)
		}
	}()
	go func() {
		if err := rs.authsvc.UpdateUserAccess(userID); err != nil {
			logrus.Warnf("auth svc error: %v", err)
		}
	}()

	return nil
}

func (rs *ResourceSvc) DeleteNamespace(ctx context.Context, userID, nsLabel string) error {
	var err error
	var userUUID uuid.UUID
	var ns Namespace

	userUUID, err = uuid.FromString(userID)
	if err != nil {
		return newBadInputError("invalid user ID, not a UUID: %v", err)
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

	ns, err = rs.db.namespaceGet(userUUID, nsLabel)
	if err != nil {
		return newError("database: get namespace: %v", err)
	}

	err = rs.billing.Unsubscribe(ctx, userID, nsUUID.String())
	if err != nil {
		// TODO: don't fail in the "already unsubscribed" case
		return newOtherServiceError("billing error: unsubscribe %v", err)
	}

	err = rs.authsvc.UpdateUserAccess(userID)
	if err != nil {
		logrus.Warnf("auth svc error: update user access: %v", err)
	}

	err = rs.kube.DeleteNamespace(ctx, nsUUID.String())
	if err != nil {
		logrus.Warnf("kube api error: delete namespace: %v", err)
	}

	err = rs.db.permDelete(permUUID)
	if err != nil {
		return newError("database: delete access %s: %v", permUUID, err)
	}

	err = rs.db.namespaceDelete(nsUUID)
	if err != nil {
		logrus.Errorf("database: %v", err)
	}

	go func() {
		tariff, err := rs.getNSTariff(context.TODO(), ns.TariffID.String())
		if err != nil {
			logrus.Warnf("failed to get namespace tariff %s: %v", ns.TariffID.String(), err)
			return
		}
		if err = rs.mailer.SendNamespaceDeleted(userID, nsLabel, tariff); err != nil {
			logrus.Warnf("Mailer.SendNamespaceDeleted userID=%s nsLabel=%s error: %v", userID, nsLabel, err)
		}
	}()

	return nil
}

func (rm *ResourceSvc) ListNamespaces(ctx context.Context, userID string, adminAction bool) ([]Namespace, error) {
	userUUID, err := uuid.FromString(userID)
	if err != nil {
		return nil, newBadInputError("invalid user ID, not a UUID: %v", userID, err)
	}
	namespaces, err := rm.db.namespaceList(&userUUID)
	if err != nil {
		return nil, newError("database: list namespaces: %v", err)
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
		err = newBadInputError("invalid user ID, not a UUID: %v", userID, err)
		return
	}
	ns, err = rs.db.namespaceGet(userUUID, nsLabel)
	if err != nil {
		err = newError("database: get namespace: %v", err)
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
		return newBadInputError("invalid owner user ID, not a UUID: %v", err)
	}
	otherUserUUID, err := uuid.FromString(otherUserID)
	if err != nil {
		return newBadInputError("invalid other user ID, not a UUID: %v", err)
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
			return newError("database: grant permission: %v", err)
		}
	} else {
		if permOther == "none" {
			err = rs.db.permDelete(permUUID)
			if err != nil {
				return newError("database: deleting access level: %v", err)
			}
		} else {
			err = rs.db.permSetLevel(permUUID, permOther)
			if err != nil {
				return newError("database: setting access level: %v", err)
			}
		}
	}

	err = rs.authsvc.UpdateUserAccess(otherUserID)
	if err != nil {
		logrus.Warnf("auth svc error: failed to update user access: %v", err)
	}

	return nil
}

func (rs *ResourceSvc) LockAccess(ownerUserID, resKind, resLabel string, lockState bool) error {
	ownerUserUUID, err := uuid.FromString(ownerUserID)
	if err != nil {
		return newBadInputError("invalid user ID, not a UUID: %v", err)
	}

	_, permUUID, lvl, err := rs.db.permGet(ownerUserUUID, resKind, resLabel)
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

	return
}

func (rs *ResourceSvc) getVolumeTariff(ctx context.Context, id string) (t model.VolumeTariff, err error) {
	if rs.tariffCache == nil {
		rs.tariffCache = cache.NewTimed(time.Second * 10)
	}

	if tmp, cached := rs.tariffCache.Get(id); cached && tmp != nil {
		t = tmp.(model.VolumeTariff)
	} else {
		t, err = rs.billing.GetVolumeTariff(ctx, id)
		if err != nil {
			return
		}
		rs.tariffCache.Set(id, t)
	}

	return
}

func (rs *ResourceSvc) CreateVolume(ctx context.Context, userID, label, tariffID string, adminAction bool) error {
	var err error
	var userUUID, permUUID, volUUID uuid.UUID
	var tariff model.VolumeTariff
	var rbDBVolumeCreate, rbDBPermCreate, rbAuthSvc bool

	// Parse input
	userUUID, err = uuid.FromString(userID)
	if err != nil {
		return newBadInputError("invalid user ID, not a UUID: %v", err)
	}

	// Get supplementary info
	if tariff, err = rs.getVolumeTariff(ctx, tariffID); err != nil {
		return newOtherServiceError("failed to get volume tariff %s: %v", tariffID, err)
	}

	// Create records in our db and prepare rollbacks
	if volUUID, err = rs.db.volumeCreate(tariff); err != nil {
		return newError("database: create volume: %v", err)
	}
	rbDBVolumeCreate = true
	defer func() {
		if rbDBVolumeCreate {
			rs.db.volumeDelete(volUUID)
		}
	}()
	if permUUID, err = rs.db.permCreateOwner("Volume", volUUID, label, userUUID); err != nil {
		return newError("database: create access level: %v", err)
	}
	rbDBPermCreate = true
	defer func() {
		if rbDBPermCreate {
			rs.db.permDelete(permUUID)
		}
	}()

	// Register new resource with other services
	if err = rs.authsvc.UpdateUserAccess(userID); err != nil {
		return newOtherServiceError("auth svc error: add access to volume: %v", err)
	}
	rbAuthSvc = true
	defer func() {
		if rbAuthSvc {
			rs.authsvc.UpdateUserAccess(userID)
		}
	}()

	// Create the volume
	if err = rs.volumesvc.CreateVolume(); err != nil {
		return newOtherServiceError("volume svc error: create volume: %v", err)
	}

	// Cancel rollbacks
	rbDBVolumeCreate = false
	rbDBPermCreate = false
	rbAuthSvc = false

	// Non-critical commands to other services
	go func() {
		if err := rs.mailer.SendVolumeCreated(userID, label, tariff); err != nil {
			logrus.Warnf("mailer error: send volume created: %v", err)
		}
	}()

	return nil
}

func (rs *ResourceSvc) DeleteVolume(ctx context.Context, userID, label string) (err error) {
	var permUUID, userUUID, volUUID uuid.UUID
	var accessLevel string
	var vol Volume

	volUUID, permUUID, accessLevel, err = rs.db.permGet(userUUID, "Volume", label)
	if err != nil {
		return newError("database: get access level: %v", err)
	}
	if !permCheck(accessLevel, "delete") {
		return newPermissionError("permission denied")
	}

	if vol, err = rs.db.volumeGetByID(volUUID); err != nil {
		return newError("database: get volume: %v", err)
	}

	if err = rs.billing.Unsubscribe(ctx, userID, volUUID.String()); err != nil {
		// TODO:
		//var canContinue bool
		//if errBilling, ok := err.(other.BillingError); ok {
		//	if errBilling.IsAlreadyUnsubscribed() {
		//		canContinue = true
		//	}
		//}
		//if !canContinue {
		//	return newOtherServiceError("billing: unsubscribe: %v", err)
		//}

		return newOtherServiceError("billing error: unsubscribe: %v", err)
	}

	if err = rs.volumesvc.DeleteVolume(); err != nil {
		return newOtherServiceError("volume svc error: deleting volume: %v", err)
	}

	if err = rs.db.permDelete(permUUID); err != nil {
		logrus.Warnf("database: delete access level %s: %v", err)
	}

	if err = rs.db.volumeDelete(volUUID); err != nil {
		logrus.Warnf("database: delete volume: %v", err)
	}

	go func() {
		tariff, err := rs.getVolumeTariff(context.TODO(), vol.TariffID.String())
		if err != nil {
			logrus.Warnf("failed to get volume tariff %s: %v", vol.TariffID.String(), err)
			return
		}
		if err := rs.mailer.SendVolumeDeleted(userID, label, tariff); err != nil {
			logrus.Warnf("mailer error: send volume deleted: %v", err)
		}
	}()

	return
}

func (rs *ResourceSvc) ListVolumes(ctx context.Context, userID string, adminAction bool) (volList []Volume, err error) {
	var userUUID uuid.UUID
	if userUUID, err = uuid.FromString(userID); err != nil {
		err = newBadInputError("invalid user ID, not a UUID: %v", err)
		return
	}
	if volList, err = rs.db.volumeList(&userUUID); err != nil {
		err = newError("database: list volumes: %v", err)
		return
	}
	if !adminAction {
		for i := range volList {
			volList[i].ID = nil
		}
	}
	return
}
