package server

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"git.containerum.net/ch/resource-service/server/model"
	"git.containerum.net/ch/resource-service/server/other"
	"git.containerum.net/ch/resource-service/util/cache"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

type ResourceSvcInterface interface {
	CreateNamespace(ctx context.Context, userID, nsLabel, tariffID string, adminAction bool) error
	DeleteNamespace(ctx context.Context, userID, nsLabel string) error
	RenameNamespace(ctx context.Context, userID, labelOld, labelNew string) error
	ListNamespaces(ctx context.Context, userID string, adminAction bool) ([]Namespace, error)
	GetNamespace(ctx context.Context, userID, nsLabel string, adminAction bool) (Namespace, error)
	ChangeNamespaceAccess(ctx context.Context, userID, label, otherUserID, access string) error
	LockNamespace(ctx context.Context, userID, label string, lockState bool) error

	CreateVolume(ctx context.Context, userID, vLabel, tariffID string, adminAction bool) error
	DeleteVolume(ctx context.Context, userID, vLabel string) error
	RenameVolume(ctx context.Context, userID, labelOld, labelNew string) error
	ListVolumes(ctx context.Context, userID string, adminAction bool) ([]Volume, error)
	GetVolume(ctx context.Context, userID, label string, adminAction bool) (Volume, error)
	ChangeVolumeAccess(ctx context.Context, userID, label, otherUserID, access string) error
	LockVolume(ctx context.Context, userID, label string, lockState bool) error

	// ListAll… methods don't ask for authorization, the frontend must bother with that.
	// Obviously, the required access level is implied to be 'admin'.
	//
	// Context varialbes queried:
	//    sort-by (string enum: "time", "owner_user_id")
	//    sort-direction (string enum: "asc", "desc")
	//    after-time (time.Time)
	//    after-user (UUID)
	//    count (uint)
	//    limited (bool)
	//    deleted (bool)
	//
	ListAllNamespaces(ctx context.Context) (<-chan Namespace, error)
	ListAllVolumes(ctx context.Context) (<-chan Volume, error)
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
func (rm *ResourceSvc) Initialize(a other.AuthSvc, b other.Billing, k other.Kube, m other.Mailer, v other.VolumeSvc, dbDSN string) error {
	rm.authsvc = a
	rm.billing = b
	rm.kube = k
	rm.mailer = m
	rm.volumesvc = v

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
		switch err.(type) {
		case dbError:
			return newError("database: create namespace: %v", err)
		default:
			return err
		}

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

	if tariff.VV != nil && tariff.VV.TariffID != nil {
		err = rs.CreateVolume(context.TODO(), userID, nsLabel+"-volume", tariff.VV.TariffID.String(), adminAction)
		if err != nil {
			logrus.Errorf("ResourceSvc: create namespace userID=%s label=%s: failed to create volume: %v <%[1]T>", userID, nsLabel, err)
			return newError("create volume: %[1]v <%[1]T>", err)
		}
		var vol Volume
		vol, err = rs.GetVolume(context.TODO(), userID, nsLabel+"-volume", true)
		if err != nil {
			logrus.Errorf("ResourceSvc: create namespace userID=%s label=%s: failed to get new volume: %v <%[1]T>", userID, nsLabel, err)
			return newError("get volume: %[1]v <%[1]T>", err)
		}
		var trVol *dbTransaction
		trVol, err = rs.db.namespaceVolumeAssociate(nsUUID, *vol.ID)
		if err != nil {
			logrus.Errorf("ResourceSvc: create namespace userID=%s label=%s: failed to associate namespace and volume: %v",
				userID, nsLabel, err)
			return newError("database: associate volume: %v", err)
		}
		trVol.Commit()
	}

	tr.Commit()

	go func() {
		defer keepCalmAndDontPanic("CreateNamespace/mailer")
		if err := rs.mailer.SendNamespaceCreated(userID, nsLabel, tariff); err != nil {
			logrus.Warnf("mailer error: %v", err)
		}
	}()
	go func() {
		defer keepCalmAndDontPanic("CreateNamespace/authsvc")
		if err := rs.authsvc.UpdateUserAccess(userID); err != nil {
			logrus.Warnf("auth svc error: %v", err)
		}
	}()

	return nil
}

func (rs *ResourceSvc) DeleteNamespace(ctx context.Context, userID, nsLabel string) error {
	var err error
	var userUUID, nsUUID, nsTariffUUID uuid.UUID
	var tr *dbTransaction
	var avols []Volume

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

	// Delete volumes
	{
		avols, err = rs.db.namespaceVolumeListAssoc(nsUUID)
		if err != nil {
			return newError("database: list associated volumes: %v", err)
		}
		for _, avol := range avols {
			logrus.Infof("ResourceSvc.DeleteNamespace: deleting volume userID=%q label=%q", userID, *avol.Label)
			err = rs.DeleteVolume(context.TODO(), userID, *avol.Label)
			if err != nil {
				return newError("delete volume: %[1]v <%[1]T>", err)
			}
		}
	}

	tr, err = rs.db.namespaceDelete(userUUID, nsLabel)
	if err != nil {
		if err == ErrDenied || err == ErrNoSuchResource {
			return err
		} else if _, ok := err.(Err); ok {
			return err
		} else {
			return newError("database: %v", err)
		}
	}
	defer tr.Rollback()

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
		//fail = true
		logrus.Warnf("kube api error: delete namespace: %v", err)
	}

	go func() {
		defer keepCalmAndDontPanic("DeleteNamespace/mailer")
		tariff, err := rs.getNSTariff(context.TODO(), nsTariffUUID.String())
		if err != nil {
			logrus.Warnf("failed to get namespace tariff %s: %v", nsTariffUUID.String(), err)
			return
		}
		if err = rs.mailer.SendNamespaceDeleted(userID, nsLabel, tariff); err != nil {
			logrus.Warnf("Mailer.SendNamespaceDeleted userID=%s nsLabel=%s error: %v", userID, nsLabel, err)
		}
	}()

	tr.Commit()

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
	for i, ns := range namespaces {
		namespaces[i].Volumes, err = rm.db.namespaceVolumeListAssoc(*ns.ID)
		if err != nil {
			return nil, newError("database: list associated volumes for ns %s: %v", *ns.ID, err)
		}
	}

	if !adminAction {
		for i := range namespaces {
			namespaces[i].ID = nil
			for j := range namespaces[i].Volumes {
				namespaces[i].Volumes[j].ID = nil
			}
		}
	}
	if namespaces == nil {
		namespaces = []Namespace{}
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

	ns.Volumes, err = rs.db.namespaceVolumeListAssoc(*ns.ID)
	if err != nil {
		err = newError("database: list associated volumes for ns %s: %v", *ns.ID, err)
		return
	}

	if !adminAction {
		ns.ID = nil
		for i := range ns.Volumes {
			ns.Volumes[i].ID = nil
		}
	}
	return
}

func (rs *ResourceSvc) ChangeNamespaceAccess(ctx context.Context, ownerUserID, label string, otherUserID, access string) error {
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

	tr, err = rs.db.namespaceSetAccess(ownerUserUUID, label, otherUserUUID, access)
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

func (rs *ResourceSvc) LockNamespace(ctx context.Context, userID, label string, lockState bool) error {
	var tr *dbTransaction
	var err error

	userUUID, err := uuid.FromString(userID)
	if err != nil {
		return newBadInputError("invalid user ID, not a UUID: %v", err)
	}

	tr, err = rs.db.namespaceSetLimited(userUUID, label, lockState)
	if err != nil {
		return newError("database, set limited: %v", err)
	}
	defer tr.Rollback()

	err = rs.authsvc.UpdateUserAccess(userID)
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
		if err == ErrAlreadyExists || err == ErrDenied {
			return err
		}
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
		defer keepCalmAndDontPanic("CreateVolume/mailer")
		if err := rs.mailer.SendVolumeCreated(userID, label, tariff); err != nil {
			logrus.Warnf("mailer error: send volume created: %v", err)
		}
	}()

	return nil
}

func (rs *ResourceSvc) DeleteVolume(ctx context.Context, userID, label string) (err error) {
	var userUUID uuid.UUID
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
				vol = vols[i]
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

	if err = rs.billing.Unsubscribe(ctx, userID, vol.ID.String()); err != nil {
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
		defer keepCalmAndDontPanic("DeleteVolume/mailer")
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
	if volList == nil {
		volList = []Volume{}
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

func (rs *ResourceSvc) ChangeVolumeAccess(ctx context.Context, ownerUserID, label string, otherUserID, access string) error {
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

	tr, err = rs.db.volumeSetAccess(ownerUserUUID, label, otherUserUUID, access)
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

func (rs *ResourceSvc) LockVolume(ctx context.Context, userID, label string, lockState bool) error {
	var tr *dbTransaction
	var err error

	userUUID, err := uuid.FromString(userID)
	if err != nil {
		return newBadInputError("invalid user ID, not a UUID: %v", err)
	}

	tr, err = rs.db.volumeSetLimited(userUUID, label, lockState)
	if err != nil {
		return newError("database, set limited: %v", err)
	}
	defer tr.Rollback()

	err = rs.authsvc.UpdateUserAccess(userID)
	if err != nil {
		return newOtherServiceError("auth svc error: failed to update user access: %v", err)
	}
	tr.Commit()

	return nil
}

func keepCalmAndDontPanic(tag string) {
	if r := recover(); r != nil {
		logrus.Errorf("%s: caught panic: %v", tag, r)
	}
}

func (rs *ResourceSvc) ListAllNamespaces(ctx context.Context) (<-chan Namespace, error) {
	var filterCount = func(count uint, cancel context.CancelFunc, output chan<- Namespace, input <-chan Namespace) {
		defer cancel()
		defer close(output)
		for count >= 0 {
			ns, ok := <-input
			if ok {
				output <- ns
			} else {
				return
			}
			count--
		}
	}
	var filterLimited = func(lim bool, output chan<- Namespace, input <-chan Namespace) {
		defer close(output)
		for ns := range input {
			// TODO
			output <- ns
		}
	}
	var filterDeleted = func(del bool, output chan<- Namespace, input <-chan Namespace) {
		defer close(output)
		for ns := range input {
			if del && ns.Deleted != nil && *ns.Deleted {
				output <- ns
			} else if !del && ns.Deleted != nil && !*ns.Deleted {
				output <- ns
			}
		}
	}
	//var insertVolumes = func(output chan<- Namespace, input <-chan Namespace){}
	var err error
	var ok bool
	var CS <-chan Namespace
	var C1, C2 chan Namespace
	var sortBy, sortDir string // "create_time" or "owner_user_id"
	var afterTime time.Time
	var afterUser uuid.UUID
	var count uint
	var x interface{}
	var str string

	C1 = make(chan Namespace)
	C1save := C1
	C2 = make(chan Namespace)

	if x = ctx.Value("sort-by"); x == nil {
		sortBy = "create_time"
	} else if sortBy, ok = x.(string); !ok {
		return nil, BadInputError{Err{nil, "INTERNAL", `context value "sort-by" was not string`}}
	}

	if x = ctx.Value("sort-direction"); x == nil {
		ctx = context.WithValue(ctx, "sort-direction", "ASC")
	} else if sortDir, ok = x.(string); !ok {
		return nil, BadInputError{Err{nil, "INTERNAL", `context value "sort-direction" was not string`}}
	} else {
		sortDir = strings.ToUpper(sortDir)
		switch sortDir {
		case "ASC", "DESC":
		default:
			return nil, BadInputError{Err{
				nil,
				"INTERNAL",
				`context value "sort-direction" was neither of: ASC, DESC`,
			}}
		}
	}

	if x = ctx.Value("after-time"); x != nil {
		if _, ok = x.(time.Time); !ok {
			return nil, BadInputError{Err{
				nil,
				"",
				``,
			}}
		}
		afterTime = x.(time.Time)
	}

	if x = ctx.Value("after-user"); x != nil {
		if str, ok = x.(string); !ok {
			return nil, BadInputError{Err{
				nil,
				"",
				`context value "after-user" was not string`,
			}}
		}
		if afterUser, err = uuid.FromString(str); err != nil {
			return nil, BadInputError{Err{
				err,
				"",
				fmt.Sprintf("invalid context value after-user: cannot parse UUID: %v", err),
			}}
		}
	}

	if x = ctx.Value("count"); x != nil {
		count = 20
	} else if count, ok = x.(uint); !ok {
		return nil, BadInputError{
			Err{
				nil,
				"INTERNAL",
				`context value "count" was not uint`,
			},
		}
	}

	var ctxCancel context.CancelFunc
	ctx, ctxCancel = context.WithCancel(ctx)
	go filterCount(count, ctxCancel, C2, C1)
	C1 = C2
	C2 = make(chan Namespace)

	if x = ctx.Value("limited"); x != nil {
		var b bool
		if b, ok = x.(bool); !ok {
			return nil, BadInputError{Err{nil, "INTERNAL", `context value "limited" was not bool`}}
		}
		go filterLimited(b, C2, C1)
		C1 = C2
		C2 = make(chan Namespace)
	}

	if x = ctx.Value("deleted"); x != nil {
		var b bool
		if b, ok = x.(bool); !ok {
			return nil, BadInputError{Err{nil, "INTERNAL", `context value "deleted" was not bool`}}
		}
		go filterDeleted(b, C2, C1)
		C1 = C2
		C2 = make(chan Namespace)
	}

	switch sortBy {
	case "time":
		CS, err = rs.db.namespaceListAllByTime(ctx, afterTime, count)
	case "owner_user_id":
		CS, err = rs.db.namespaceListAllByOwner(ctx, afterUser, count)
	default:
		return nil, BadInputError{Err{nil, "", "unknown sort key: " + sortBy}}
	}
	if err != nil {
		switch err.(type) {
		case BadInputError, Err, PermissionError:
			return nil, err
		default:
			return nil, Err{err, "", fmt.Sprintf("database: %v", err)}
		}
	}
	go func() {
		defer close(C1save)
		for ns := range CS {
			C1save <- ns
		}
	}()

	return C1, nil
}

func (rs *ResourceSvc) ListAllVolumes(ctx context.Context) (<-chan Volume, error) {
	return nil, Err{nil, "INTERNAL", `not implemented`}
}
