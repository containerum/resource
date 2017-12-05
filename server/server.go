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
	RenameNamespace(ctx context.Context, userID, labelOld, labelNew string) error
	ListNamespaces(ctx context.Context, userID string, adminAction bool) ([]Namespace, error)
	GetNamespace(ctx context.Context, userID, nsLabel string, adminAction bool) (Namespace, error)

	CreateVolume(ctx context.Context, userID, vLabel, tariffID string, adminAction bool) error
	DeleteVolume(ctx context.Context, userID, vLabel string) error
	RenameVolume(ctx context.Context, userID, labelOld, labelNew string) error
	ListVolumes(ctx context.Context, userID string, adminAction bool) ([]Volume, error)
	GetVolume(ctx context.Context, userID, label string, adminAction bool) (Volume, error)

	ChangeAccess(ownerUserID, resKind, resLabel string, otherUserID, accessLevel string) error
	LockAccess(ownerUserID, resKind, resLabel string, lockState bool) error

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
	var nsUUID, userUUID uuid.UUID
	var tariff model.NamespaceTariff

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

	var tr *dbTransaction
	tr, nsUUID, err = rs.db.namespaceCreate(tariff, userUUID, nsLabel)
	if err != nil {
		return newError("database: %v", err)
	}
	defer tr.Rollback()

	err = rs.kube.CreateNamespace(ctx, nsUUID.String(), uint(*tariff.CpuLimit), uint(*tariff.MemoryLimit), nsLabel, "owner")
	if err != nil {
		// TODO: don't fail if already exists
		return newOtherServiceError("kube api error: create namespace: %v", err)
	}
	err = rs.billing.Subscribe(ctx, userID, tariffID, nsUUID.String())
	if err != nil {
		return newOtherServiceError("billing error: subscribe: %v", err)
	}
	tr.Commit()

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
	var userUUID, nsUUID, nsTariffUUID uuid.UUID
	var fail bool
	var tr *dbTransaction

	userUUID, err = uuid.FromString(userID)
	if err != nil {
		return newBadInputError("invalid user ID, not a UUID: %v", err)
	}

	{
		var nss []Namespace
		nss, err = rs.db.namespaceList(userUUID)
		if err != nil {
			return newError("database: %v", err)
		}
		for _, ns := range nss {
			if ns.Label != nil && *ns.Label == nsLabel && ns.ID != nil && ns.TariffID != nil {
				nsUUID = *ns.ID
				nsTariffUUID = *ns.TariffID
			}
		}
		if nsUUID == uuid.Nil || nsTariffUUID == uuid.Nil {
			return ErrNoSuchResource
		}
	}

	tr, err = rs.db.namespaceDelete(userUUID, nsLabel)
	if err != nil {
		if err == ErrDenied || err == ErrNoSuchResource {
			return err
		} else if _, ok := err.(Error); ok {
			return err
		} else {
			return newError("database: %v", err)
		}
	}
	defer tr.Rollback()

	err = rs.billing.Unsubscribe(ctx, userID, nsUUID.String())
	if err != nil {
		// TODO: don't fail in the "already unsubscribed" case
		//fail = true
		return newOtherServiceError("billing error: unsubscribe %v", err)
	}

	err = rs.authsvc.UpdateUserAccess(userID)
	if err != nil {
		fail = true
		logrus.Warnf("auth svc error: update user access: %v", err)
	}

	err = rs.kube.DeleteNamespace(ctx, nsUUID.String())
	if err != nil {
		fail = true
		logrus.Warnf("kube api error: delete namespace: %v", err)
	}

	if !fail {
		tr.Commit()
	}

	go func() {
		tariff, err := rs.getNSTariff(context.TODO(), nsTariffUUID.String())
		if err != nil {
			logrus.Warnf("failed to get namespace tariff %s: %v", nsTariffUUID.String(), err)
			return
		}
		if err = rs.mailer.SendNamespaceDeleted(userID, nsLabel, tariff); err != nil {
			logrus.Warnf("Mailer.SendNamespaceDeleted userID=%s nsLabel=%s error: %v", userID, nsLabel, err)
		}
	}()

	return nil
}

func (rs *ResourceSvc) RenameNamespace(ctx context.Context, userID, labelOld, labelNew string) error {
	userUUID, err := uuid.FromString(userID)
	tr, err := rs.db.namespaceRename(userUUID, labelOld, labelNew)
	if err != nil {
		return newError("database: rename namespace: %v", err)
	}
	defer tr.Rollback()

	err = rs.authsvc.UpdateUserAccess(userID)
	if err != nil {
		return newOtherServiceError("auth svc error: %v", err)
	}
	tr.Commit()

	return nil
}

func (rm *ResourceSvc) ListNamespaces(ctx context.Context, userID string, adminAction bool) ([]Namespace, error) {
	userUUID, err := uuid.FromString(userID)
	if err != nil {
		return nil, newBadInputError("invalid user ID, not a UUID: %v", userID, err)
	}
	namespaces, err := rm.db.namespaceList(userUUID)
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
	var nss []Namespace
	userUUID, err := uuid.FromString(userID)
	if err != nil {
		err = newBadInputError("invalid user ID, not a UUID: %v", userID, err)
		return
	}
	nss, err = rs.db.namespaceList(userUUID)
	if err != nil {
		err = newError("database: get namespace: %v", err)
		return
	}
	for i := range nss {
		if *nss[i].Label == nsLabel {
			ns = nss[i]
			break
		}
	}
	if ns.ID == nil {
		err = ErrNoSuchResource
		return
	}
	if !adminAction {
		ns.ID = nil
	}
	return
}

// ChangeAccess adds or removes access to ownerUserID's resource for otherUserID.
func (rs *ResourceSvc) ChangeAccess(ownerUserID, resKind, resLabel string, otherUserID, permOther string) error {
	var tr *dbTransaction
	var err error

	ownerUserUUID, err := uuid.FromString(ownerUserID)
	if err != nil {
		return newBadInputError("invalid owner user ID, not a UUID: %v", err)
	}
	otherUserUUID, err := uuid.FromString(otherUserID)
	if err != nil {
		return newBadInputError("invalid other user ID, not a UUID: %v", err)
	}

	switch resKind {
	case "Namespace":
		tr, err = rs.db.namespaceSetAccess(ownerUserUUID, resLabel, otherUserUUID, permOther)
	case "Volume":
		tr, err = rs.db.volumeSetAccess(ownerUserUUID, resLabel, otherUserUUID, permOther)
	default:
		return newBadInputError("invalid resource kind")
	}
	if err != nil {
		return newError("database, set access: %v", err)
	}
	defer tr.Rollback()

	err = rs.authsvc.UpdateUserAccess(otherUserID)
	if err != nil {
		return newOtherServiceError("auth svc error: failed to update user access: %v", err)
	}
	tr.Commit()

	return nil
}

func (rs *ResourceSvc) LockAccess(ownerUserID, resKind, resLabel string, lockState bool) error {
	var tr *dbTransaction
	var err error

	ownerUserUUID, err := uuid.FromString(ownerUserID)
	if err != nil {
		return newBadInputError("invalid user ID, not a UUID: %v", err)
	}

	switch resKind {
	case "Namespace":
		tr, err = rs.db.namespaceSetLimited(ownerUserUUID, resLabel, lockState)
	case "Volume":
		tr, err = rs.db.volumeSetLimited(ownerUserUUID, resLabel, lockState)
	default:
		return newBadInputError("invalid resource kind")
	}
	if err != nil {
		return newError("database, set limited: %v", err)
	}
	defer tr.Rollback()

	err = rs.authsvc.UpdateUserAccess(ownerUserID)
	if err != nil {
		return newOtherServiceError("auth svc error: failed to update user access: %v", err)
	}
	tr.Commit()

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
	var userUUID uuid.UUID
	var tariff model.VolumeTariff
	var tr *dbTransaction

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
	if tr, _, err = rs.db.volumeCreate(tariff, userUUID, label); err != nil {
		return newError("database: create volume: %v", err)
	}
	defer tr.Rollback()

	// Create the volume
	if err = rs.volumesvc.CreateVolume(); err != nil {
		// TODO: don't fail if already exists
		return newOtherServiceError("volume svc error: create volume: %v", err)
	}

	// Update accesses in auth service
	if err = rs.authsvc.UpdateUserAccess(userID); err != nil {
		return newOtherServiceError("auth svc error: add access to volume: %v", err)
	}
	tr.Commit()

	// Non-critical commands to other services
	go func() {
		if err := rs.mailer.SendVolumeCreated(userID, label, tariff); err != nil {
			logrus.Warnf("mailer error: send volume created: %v", err)
		}
	}()

	return nil
}

func (rs *ResourceSvc) DeleteVolume(ctx context.Context, userID, label string) (err error) {
	var userUUID, volUUID uuid.UUID
	var vol Volume
	var tr *dbTransaction

	userUUID, err = uuid.FromString(userID)
	if err != nil {
		err = newBadInputError("invalid user ID, not a UUID: %v", err)
		return
	}

	{
		var vols []Volume
		vols, err = rs.db.volumeList(userUUID)
		if err != nil {
			err = newError("database: list volumes: %v", err)
			return
		}
		for i := range vols {
			if *vols[i].Label == label {
				volUUID = *vols[i].ID
				break
			}
		}
		if vol.ID == nil {
			err = ErrNoSuchResource
			return
		}
	}

	tr, err = rs.db.volumeDelete(userUUID, label)
	if err != nil {
		err = newError("database: delete volume: %v", err)
		return
	}
	defer tr.Rollback()

	if err = rs.billing.Unsubscribe(ctx, userID, volUUID.String()); err != nil {
		// TODO:
		//var canContinue bool
		//if errBilling, ok := err.(other.BillingError); ok {
		//	if errBilling.IsAlreadyUnsubscribed() {
		//		canContinue = true
		//	}
		//}
		//if !canContinue {
		//	return newOtherServiceError("billing error: unsubscribe: %v", err)
		//}

		return newOtherServiceError("billing error: unsubscribe: %v", err)
	}

	err = rs.volumesvc.DeleteVolume()
	if err != nil {
		return newOtherServiceError("volume svc error: deleting volume: %v", err)
	}
	tr.Commit()

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
	if volList, err = rs.db.volumeList(userUUID); err != nil {
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

func (rs *ResourceSvc) GetVolume(ctx context.Context, userID, label string, adminAction bool) (vol Volume, err error) {
	userUUID, err := uuid.FromString(userID)
	if err != nil {
		err = newBadInputError("invalid user ID, not a UUID: %v", err)
		return
	}

	var vols []Volume
	vols, err = rs.db.volumeList(userUUID)
	if err != nil {
		err = newError("database: list volumes: %v", err)
		return
	}

	for i := range vols {
		if *vols[i].Label == label {
			vol = vols[i]
			break
		}
	}
	if vol.ID == nil {
		err = ErrNoSuchResource
		return
	}
	return
}

func (rs *ResourceSvc) RenameVolume(ctx context.Context, userID, labelOld, labelNew string) error {
	userUUID, err := uuid.FromString(userID)
	tr, err := rs.db.volumeRename(userUUID, labelOld, labelNew)
	if err != nil {
		return newError("database: rename volume: %v", err)
	}
	defer tr.Rollback()

	err = rs.authsvc.UpdateUserAccess(userID)
	if err != nil {
		return newOtherServiceError("auth svc error: %v", err)
	}
	tr.Commit()

	return nil
}
