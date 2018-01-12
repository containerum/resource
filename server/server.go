package server

import (
	"context"
	"strings"
	"time"

	"git.containerum.net/ch/json-types/errors"
	rstypes "git.containerum.net/ch/json-types/resource-service"
	rserrors "git.containerum.net/ch/resource-service/server/errors"
	. "git.containerum.net/ch/resource-service/server/models" // too hard to fix package name after movement :)
	"git.containerum.net/ch/resource-service/server/other"
	"git.containerum.net/ch/resource-service/util/cache"

	"io"

	"reflect"

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
	ResizeNamespace(ctx context.Context, userID, label, newTariffID string) error

	CreateVolume(ctx context.Context, userID, vLabel, tariffID string, adminAction bool) error
	DeleteVolume(ctx context.Context, userID, vLabel string) error
	RenameVolume(ctx context.Context, userID, labelOld, labelNew string) error
	ListVolumes(ctx context.Context, userID string, adminAction bool) ([]Volume, error)
	GetVolume(ctx context.Context, userID, label string, adminAction bool) (Volume, error)
	ChangeVolumeAccess(ctx context.Context, userID, label, otherUserID, access string) error
	LockVolume(ctx context.Context, userID, label string, lockState bool) error
	ResizeVolume(ctx context.Context, userID, label, newTariffID string) error

	// ListAll… methods don't ask for authorization, the frontend must bother with that.
	// Obviously, the required access level is implied to be 'admin'. Output is always
	// paginated.
	//
	// Context varialbes queried:
	//    sort-direction (string enum: "asc", "desc")
	//    after-time (time.Time)
	//    after-user (ID)
	//    count (uint)
	//    limited (bool)
	//    deleted (bool)
	//
	ListAllNamespaces(ctx context.Context) (<-chan Namespace, error)
	ListAllVolumes(ctx context.Context) (<-chan Volume, error)

	// Admin-only access.
	GetNamespaceAccesses(ctx context.Context, userID, label string) (Namespace, error)
	GetVolumeAccesses(ctx context.Context, userID, label string) (Volume, error)

	// To close connections
	io.Closer
}

type ResourceSvcClients struct {
	Auth    other.AuthSvc
	Billing other.Billing
	Mailer  other.Mailer
	Kube    other.Kube
	Volume  other.VolumeSvc
}

type ResourceSvc struct {
	ResourceSvcClients

	db          *ResourceSvcDB
	tariffCache cache.Cache
	log         *logrus.Entry
}

// TODO
var _ ResourceSvcInterface = &ResourceSvc{}

func NewResourceSvc(clients ResourceSvcClients, dbDSN string) (ResourceSvcInterface, error) {
	rs := &ResourceSvc{
		ResourceSvcClients: clients,
		log:                logrus.WithField("component", "resource_service"),
	}

	var err error
	if rs.db, err = DBConnect(dbDSN); err != nil {
		return nil, err
	}
	return rs, nil
}

func (rs *ResourceSvc) CreateNamespace(ctx context.Context, userID, nsLabel, tariffID string, adminAction bool) error {
	var err error
	var nsID string
	var tariff rstypes.NamespaceTariff

	tariff, err = rs.getNSTariff(ctx, tariffID)
	if err != nil {
		return rserrors.NewOtherServiceError("failed to get namespace tariff: %v", err)
	}

	if !tariff.IsActive {
		return rserrors.NewPermissionError("cannot subscribe to inactive tariff")
	}
	if !adminAction {
		if !tariff.IsPublic {
			return rserrors.NewPermissionError("tariff unavailable")
		}
	}

	err = rs.db.Transactional(func(tx ResourceSvcDB) error {
		var err error
		if nsID, err = tx.NamespaceCreate(tariff, userID, nsLabel); err != nil {
			return err
		}

		err = rs.Kube.CreateNamespace(ctx, nsID, uint(tariff.CpuLimit), uint(tariff.MemoryLimit), nsLabel, "owner")
		if err != nil {
			// TODO: don't fail if already exists
			return rserrors.NewOtherServiceError("Kube api error: create namespace: %v", err)
		}
		err = rs.Billing.Subscribe(ctx, userID, tariffID, nsID)
		if err != nil {
			return rserrors.NewOtherServiceError("Billing error: subscribe: %v", err)
		}

		if tariff.VV != nil && tariff.VV.TariffID != "" {
			err = rs.db.Transactional(func(volTx ResourceSvcDB) error {
				err := rs.CreateVolume(context.TODO(), userID, nsLabel+"-volume", tariff.VV.TariffID, adminAction)
				if err != nil {
					rs.log.Errorf("ResourceSvc: create namespace userID=%s label=%s: failed to create volume: %v <%[1]T>", userID, nsLabel, err)
					return errors.Format("create volume: %[1]v <%[1]T>", err)
				}
				var vol Volume
				vol, err = rs.GetVolume(context.TODO(), userID, nsLabel+"-volume", true)
				if err != nil {
					rs.log.Errorf("ResourceSvc: create namespace userID=%s label=%s: failed to get new volume: %v <%[1]T>", userID, nsLabel, err)
					return errors.Format("get volume: %[1]v <%[1]T>", err)
				}
				err = volTx.NamespaceVolumeAssociate(nsID, vol.ID)
				if err != nil {
					rs.log.Errorf("ResourceSvc: create namespace userID=%s label=%s: failed to associate namespace and volume: %v",
						userID, nsLabel, err)
					return errors.Format("database: associate volume: %v", err)
				}
				return nil
			})
		}
		return nil
	})
	if err != nil {
		return err
	}

	go func() {
		defer rs.keepCalmAndDontPanic("CreateNamespace/Mailer")
		if err := rs.Mailer.SendNamespaceCreated(userID, nsLabel, tariff); err != nil {
			rs.log.Warnf("Mailer error: %v", err)
		}
	}()
	go func() {
		defer rs.keepCalmAndDontPanic("CreateNamespace/Auth")
		if err := rs.Auth.UpdateUserAccess(userID); err != nil {
			rs.log.Warnf("auth svc error: %v", err)
		}
	}()

	return nil
}

func (rs *ResourceSvc) DeleteNamespace(ctx context.Context, userID, nsLabel string) error {
	var err error
	var nsID, nsTariffID string
	var avols []Volume

	{
		var nss []Namespace
		nss, err = rs.db.NamespaceList(userID)
		if err != nil {
			return errors.Format("database: %v", err)
		}
		for _, ns := range nss {
			if ns.Label == nsLabel && ns.ID != "" && ns.TariffID != "" {
				nsID = ns.ID
				nsTariffID = ns.TariffID
			}
		}
		if nsID == "" || nsTariffID == "" {
			return rserrors.ErrNoSuchResource
		}
	}

	// Delete volumes
	{
		avols, err = rs.db.NamespaceVolumeListAssoc(nsID)
		if err != nil {
			return errors.Format("database: list associated volumes: %v", err)
		}
		for _, avol := range avols {
			rs.log.Infof("ResourceSvc.DeleteNamespace: deleting volume userID=%q label=%q", userID, avol.Label)
			err = rs.DeleteVolume(context.TODO(), userID, avol.Label)
			if err != nil {
				return errors.Format("delete volume: %[1]v <%[1]T>", err)
			}
		}
	}

	err = rs.db.Transactional(func(tx ResourceSvcDB) error {
		err := tx.NamespaceDelete(userID, nsLabel)
		if err != nil {
			if err == rserrors.ErrDenied || err == rserrors.ErrNoSuchResource {
				return err
			} else if _, ok := err.(*errors.Error); ok {
				return err
			} else {
				return errors.Format("database: %v", err)
			}
		}

		err = rs.Billing.Unsubscribe(ctx, userID, nsID)
		if err != nil {
			// TODO: don't fail in the "already unsubscribed" case
			return rserrors.NewOtherServiceError("Billing error: unsubscribe %v", err)
		}

		err = rs.Auth.UpdateUserAccess(userID)
		if err != nil {
			rs.log.Warnf("auth svc error: update user access: %v", err)
		}

		err = rs.Kube.DeleteNamespace(ctx, nsID)
		if err != nil {
			rs.log.Warnf("Kube api error: delete namespace: %v", err)
		}

		return nil
	})

	go func() {
		defer rs.keepCalmAndDontPanic("DeleteNamespace/Mailer")
		tariff, err := rs.getNSTariff(ctx, nsTariffID)
		if err != nil {
			rs.log.Warnf("failed to get namespace tariff %s: %v", nsTariffID, err)
			return
		}
		if err = rs.Mailer.SendNamespaceDeleted(userID, nsLabel, tariff); err != nil {
			rs.log.Warnf("Mailer.SendNamespaceDeleted userID=%s nsLabel=%s error: %v", userID, nsLabel, err)
		}
	}()

	return nil
}

func (rs *ResourceSvc) RenameNamespace(ctx context.Context, userID, labelOld, labelNew string) error {
	err := rs.db.Transactional(func(tx ResourceSvcDB) error {
		err := tx.NamespaceRename(userID, labelOld, labelNew)
		if err != nil {
			return errors.Format("database: rename namespace: %v", err)
		}

		err = rs.Auth.UpdateUserAccess(userID)
		if err != nil {
			return rserrors.NewOtherServiceError("auth svc error: %v", err)
		}

		return nil
	})

	return err
}

func (rs *ResourceSvc) ListNamespaces(ctx context.Context, userID string, adminAction bool) ([]Namespace, error) {
	namespaces, err := rs.db.NamespaceList(userID)
	if err != nil {
		return nil, errors.Format("database: list namespaces: %v", err)
	}
	for i, ns := range namespaces {
		var avols []Volume
		avols, err = rs.db.NamespaceVolumeListAssoc(ns.ID)
		if err != nil {
			return nil, errors.Format("database: list associated volumes for ns %s: %v", ns.ID, err)
		}
		var uservols []Volume
		uservols, err = rs.ListVolumes(ctx, userID, true)
		if err != nil {
			return nil, errors.Format("list volumes: %v", err)
		}
		for i := range uservols {
			for j := range avols {
				if uservols[i].ID == avols[j].ID {
					ns.Volumes = append(ns.Volumes, uservols[i])
				}
			}
		}

		namespaces[i] = ns
	}

	if !adminAction {
		for i := range namespaces {
			namespaces[i].ID = ""
			namespaces[i].Access = namespaces[i].NewAccess
			namespaces[i].NewAccess = ""
			namespaces[i].Limited = false
			for j := range namespaces[i].Volumes {
				namespaces[i].Volumes[j].ID = ""
				namespaces[i].Volumes[j].Access = namespaces[i].Volumes[j].NewAccess
				namespaces[i].Volumes[j].NewAccess = ""
				namespaces[i].Volumes[j].Limited = false
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
	nss, err = rs.db.NamespaceList(userID)
	if err != nil {
		err = errors.Format("database: get namespace: %v", err.Error())
		return
	}
	for i := range nss {
		if nss[i].Label == nsLabel {
			ns = nss[i]
			break
		}
	}
	if ns.ID == "" {
		err = rserrors.ErrNoSuchResource
		return
	}

	var avols []Volume
	avols, err = rs.db.NamespaceVolumeListAssoc(ns.ID)
	if err != nil {
		err = errors.Format("database: list associated volumes for ns %v : %v", ns.ID, err)
		return
	}

	var uservols []Volume
	uservols, err = rs.ListVolumes(ctx, userID, true)
	if err != nil {
		err = errors.Format("list volumes: %v", err)
		return
	}

	for i := range uservols {
		for j := range avols {
			if uservols[i].ID == avols[j].ID {
				ns.Volumes = append(ns.Volumes, uservols[i])
			}
		}
	}

	if !adminAction {
		ns.ID = ""
		ns.Access = ns.NewAccess
		ns.NewAccess = ""
		ns.Limited = false
		for i := range ns.Volumes {
			ns.Volumes[i].ID = ""
			ns.Volumes[i].Access = ns.Volumes[i].NewAccess
			ns.Volumes[i].NewAccess = ""
			ns.Volumes[i].Limited = false
		}
	}
	return
}

func (rs *ResourceSvc) ChangeNamespaceAccess(ctx context.Context, ownerUserID, label string, otherUserID, access string) error {
	err := rs.db.Transactional(func(tx ResourceSvcDB) error {
		err := tx.NamespaceSetAccess(ownerUserID, label, otherUserID, access)
		if err != nil {
			return errors.Format("database, set access: %v", err)
		}

		err = rs.Auth.UpdateUserAccess(otherUserID)
		if err != nil {
			return rserrors.NewOtherServiceError("auth svc error: failed to update user access: %v", err)
		}

		return nil
	})

	return err
}

func (rs *ResourceSvc) LockNamespace(ctx context.Context, userID, label string, lockState bool) error {
	err := rs.db.Transactional(func(tx ResourceSvcDB) error {
		err := tx.NamespaceSetLimited(userID, label, lockState)
		if err != nil {
			return errors.Format("database, set limited: %v", err)
		}

		err = rs.Auth.UpdateUserAccess(userID)
		if err != nil {
			return rserrors.NewOtherServiceError("auth svc error: failed to update user access: %v", err)
		}
		return nil
	})

	return err
}

func (rs *ResourceSvc) getNSTariff(ctx context.Context, id string) (t rstypes.NamespaceTariff, err error) {
	if rs.tariffCache == nil {
		rs.tariffCache = cache.NewTimed(time.Second * 10)
	}

	if tmp, cached := rs.tariffCache.Get(id); cached && tmp != nil {
		t = tmp.(rstypes.NamespaceTariff)
	} else {
		t, err = rs.Billing.GetNamespaceTariff(ctx, id)
		if err != nil {
			return
		}
		rs.tariffCache.Set(id, t)
	}

	return
}

func (rs *ResourceSvc) getVolumeTariff(ctx context.Context, id string) (t rstypes.VolumeTariff, err error) {
	if rs.tariffCache == nil {
		rs.tariffCache = cache.NewTimed(time.Second * 10)
	}

	if tmp, cached := rs.tariffCache.Get(id); cached && tmp != nil {
		t = tmp.(rstypes.VolumeTariff)
	} else {
		t, err = rs.Billing.GetVolumeTariff(ctx, id)
		if err != nil {
			return
		}
		rs.tariffCache.Set(id, t)
	}

	return
}

func (rs *ResourceSvc) CreateVolume(ctx context.Context, userID, label, tariffID string, adminAction bool) error {
	var err error
	var tariff rstypes.VolumeTariff

	// Get supplementary info
	if tariff, err = rs.getVolumeTariff(ctx, tariffID); err != nil {
		return rserrors.NewOtherServiceError("failed to get volume tariff %s: %v", tariffID, err)
	}

	err = rs.db.Transactional(func(tx ResourceSvcDB) error {
		if _, err = tx.VolumeCreate(tariff, userID, label); err != nil {
			if err == rserrors.ErrAlreadyExists || err == rserrors.ErrDenied {
				return err
			}
			return errors.Format("database: create volume: %v", err)
		}

		// Create the volume
		if err = rs.Volume.CreateVolume(); err != nil {
			// TODO: don't fail if already exists
			return rserrors.NewOtherServiceError("volume svc error: create volume: %v", err)
		}

		// Update accesses in auth service
		if err = rs.Auth.UpdateUserAccess(userID); err != nil {
			return rserrors.NewOtherServiceError("auth svc error: add access to volume: %v", err)
		}

		return nil
	})

	// Non-critical commands to other services
	go func() {
		defer rs.keepCalmAndDontPanic("CreateVolume/Mailer")
		if err := rs.Mailer.SendVolumeCreated(userID, label, tariff); err != nil {
			rs.log.Warnf("Mailer error: send volume created: %v", err)
		}
	}()

	return nil
}

func (rs *ResourceSvc) DeleteVolume(ctx context.Context, userID, label string) (err error) {
	var vol *Volume
	{
		var vols []Volume
		vols, err = rs.db.VolumeList(userID)
		if err != nil {
			err = errors.Format("database: list volumes: %v", err)
			return
		}
		for i := range vols {
			if vols[i].Label == label {
				vol = &vols[i]
				break
			}
		}
		if vol == nil {
			err = rserrors.ErrNoSuchResource
			return
		}
	}

	err = rs.db.Transactional(func(tx ResourceSvcDB) error {
		err := tx.VolumeDelete(userID, label)
		if err != nil {
			err = errors.Format("database: delete volume: %v", err)
			return err
		}

		if err = rs.Billing.Unsubscribe(ctx, userID, vol.ID); err != nil {
			// TODO:
			//var canContinue bool
			//if errBilling, ok := err.(other.BillingError); ok {
			//	if errBilling.IsAlreadyUnsubscribed() {
			//		canContinue = true
			//	}
			//}
			//if !canContinue {
			//	return rserrors.NewOtherServiceError("Billing error: unsubscribe: %v", err)
			//}

			return rserrors.NewOtherServiceError("Billing error: unsubscribe: %v", err)
		}

		err = rs.Volume.DeleteVolume()
		if err != nil {
			return rserrors.NewOtherServiceError("volume svc error: deleting volume: %v", err)
		}

		return nil
	})

	go func() {
		defer rs.keepCalmAndDontPanic("DeleteVolume/Mailer")
		tariff, err := rs.getVolumeTariff(context.TODO(), vol.TariffID)
		if err != nil {
			rs.log.Warnf("failed to get volume tariff %s: %v", vol.TariffID, err)
			return
		}
		if err := rs.Mailer.SendVolumeDeleted(userID, label, tariff); err != nil {
			rs.log.Warnf("Mailer error: send volume deleted: %v", err)
		}
	}()

	return
}

func (rs *ResourceSvc) ListVolumes(ctx context.Context, userID string, adminAction bool) (volList []Volume, err error) {
	if volList, err = rs.db.VolumeList(userID); err != nil {
		err = errors.Format("database: list volumes: %v", err)
		return
	}
	if !adminAction {
		for i := range volList {
			volList[i].ID = "" // remove from response for non-admins
		}
	}
	if volList == nil {
		volList = []Volume{}
	}
	return
}

func (rs *ResourceSvc) GetVolume(ctx context.Context, userID, label string, adminAction bool) (vol Volume, err error) {
	var vols []Volume
	vols, err = rs.db.VolumeList(userID)
	if err != nil {
		err = errors.Format("database: list volumes: %v", err)
		return
	}

	for i := range vols {
		if vols[i].Label == label {
			vol = vols[i]
			break
		}
	}
	if vol.ID == "" {
		err = rserrors.ErrNoSuchResource
		return
	}
	return
}

func (rs *ResourceSvc) RenameVolume(ctx context.Context, userID, labelOld, labelNew string) error {
	err := rs.db.Transactional(func(tx ResourceSvcDB) error {
		err := tx.VolumeRename(userID, labelOld, labelNew)
		if err != nil {
			return errors.Format("database: rename volume: %v", err)
		}

		err = rs.Auth.UpdateUserAccess(userID)
		if err != nil {
			return rserrors.NewOtherServiceError("auth svc error: %v", err)
		}

		return nil
	})

	return err
}

func (rs *ResourceSvc) ChangeVolumeAccess(ctx context.Context, ownerUserID, label string, otherUserID, access string) error {
	err := rs.db.Transactional(func(tx ResourceSvcDB) error {
		err := tx.VolumeSetAccess(ownerUserID, label, otherUserID, access)
		if err != nil {
			return errors.Format("database, set access: %v", err)
		}

		err = rs.Auth.UpdateUserAccess(otherUserID)
		if err != nil {
			return rserrors.NewOtherServiceError("auth svc error: failed to update user access: %v", err)
		}

		return nil
	})

	return err
}

func (rs *ResourceSvc) LockVolume(ctx context.Context, userID, label string, lockState bool) error {
	err := rs.db.Transactional(func(tx ResourceSvcDB) error {
		err := tx.VolumeSetLimited(userID, label, lockState)
		if err != nil {
			return errors.Format("database, set limited: %v", err)
		}

		err = rs.Auth.UpdateUserAccess(userID)
		if err != nil {
			return rserrors.NewOtherServiceError("auth svc error: failed to update user access: %v", err)
		}

		return nil
	})

	return err
}

func (rs *ResourceSvc) keepCalmAndDontPanic(tag string) {
	if r := recover(); r != nil {
		rs.log.Errorf("%s: caught panic: %v", tag, r)
	}
}

func (rs *ResourceSvc) ResizeNamespace(ctx context.Context, userID, label, newTariffID string) (err error) {
	var user string
	var tariff rstypes.NamespaceTariff
	var ns Namespace

	tariff, err = rs.getNSTariff(ctx, newTariffID)
	if err != nil {
		err = rserrors.NewOtherServiceError("get namespace tariff: %v", err.Error())
		return
	}

	ns, err = rs.GetNamespace(ctx, userID, label, true)
	if err != nil {
		if err == rserrors.ErrNoSuchResource || err == rserrors.ErrDenied {
			return err
		}
		return errors.Format("get namespace: %v", err)
	}

	err = rs.db.Transactional(func(tx ResourceSvcDB) error {
		err := tx.NamespaceSetTariff(user, label, tariff)
		if err != nil {
			err = errors.Format("database, set namespace tariff: %v", err)
			return err
		}

		if err = rs.Billing.Subscribe(ctx, userID, newTariffID, ns.ID); err != nil {
			// TODO: don't fail if already subscribed
			err = rserrors.NewOtherServiceError("Billing error: %v", err)
			return err
		}

		if err = rs.Kube.SetNamespaceQuota(ctx, ns.ID, uint(tariff.CpuLimit), uint(tariff.MemoryLimit), ns.Label, string(ns.Access)); err != nil {
			err = rserrors.NewOtherServiceError("Kube api error: %v", err)
			return err
		}

		return nil
	})

	return
}

func (rs *ResourceSvc) ResizeVolume(ctx context.Context, userID, label, newTariffID string) (err error) {
	var user string
	var tariff rstypes.VolumeTariff
	var vol Volume

	tariff, err = rs.getVolumeTariff(ctx, newTariffID)
	if err != nil {
		err = rserrors.NewOtherServiceError("get volume tariff: %v", err)
		return
	}

	vol, err = rs.GetVolume(ctx, userID, label, true)
	if err != nil {
		if err == rserrors.ErrNoSuchResource || err == rserrors.ErrDenied {
			return err
		}
		return errors.Format("get volume: %v", err)
	}

	err = rs.db.Transactional(func(tx ResourceSvcDB) error {
		err := tx.VolumeSetTariff(user, label, tariff)
		if err != nil {
			err = errors.Format("database, set volume tariff: %v", err)
			return err
		}

		if tariff.IsPersistent {
			if err = rs.Billing.Subscribe(ctx, userID, newTariffID, vol.ID); err != nil {
				// TODO: don't fail if already subscribed
				err = errors.Format("Billing error: %v", err)
				return err
			}
		}

		return nil
	})
	return
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
			if del && ns.Deleted {
				output <- ns
			} else if !del && !ns.Deleted {
				output <- ns
			}
		}
	}
	var err error
	var ok bool
	var CS <-chan Namespace
	var C1, C2 chan Namespace //last 2 links in the chain of post-processing goroutines
	var sortDir string
	var afterTime time.Time
	var count uint
	var x interface{}

	C1 = make(chan Namespace)
	C1save := C1
	C2 = make(chan Namespace)

	if x = ctx.Value("sort-direction"); x == nil {
		ctx = context.WithValue(ctx, "sort-direction", "ASC")
	} else if sortDir, ok = x.(string); !ok {
		return nil, rserrors.NewBadInputError(`context value "sort-direction" was not string`)
	} else {
		sortDir = strings.ToUpper(sortDir)
		switch sortDir {
		case "ASC", "DESC":
		default:
			return nil, rserrors.NewBadInputError(`context value "sort-direction" was neither of: ASC, DESC`)
		}
	}

	if x = ctx.Value("after-time"); x != nil {
		if _, ok = x.(time.Time); !ok {
			return nil, rserrors.NewBadInputError(`context value "after-time" was not time.Time`)
		}
		afterTime = x.(time.Time)
	}

	if x = ctx.Value("count"); x == nil {
		count = 50
	} else if count, ok = x.(uint); !ok {
		return nil, rserrors.NewBadInputError(`context value "count" was not uint`)
	}

	var ctxCancel context.CancelFunc
	ctx, ctxCancel = context.WithCancel(ctx)
	go filterCount(count, ctxCancel, C2, C1)
	C1 = C2
	C2 = make(chan Namespace)

	if x = ctx.Value("limited"); x != nil {
		var b bool
		if b, ok = x.(bool); !ok {
			return nil, rserrors.NewBadInputError(`context value "limited" was not bool`)
		}
		go filterLimited(b, C2, C1)
		C1 = C2
		C2 = make(chan Namespace)
	}

	if x = ctx.Value("deleted"); x != nil {
		var b bool
		if b, ok = x.(bool); !ok {
			return nil, rserrors.NewBadInputError(`context value "deleted" was not bool`)
		}
		go filterDeleted(b, C2, C1)
		C1 = C2
		C2 = make(chan Namespace)
	}

	CS, err = rs.db.NamespaceListAllByTime(ctx, afterTime, count)
	if err != nil {
		switch err.(type) {
		case *rserrors.BadInputError, *rserrors.PermissionError, *errors.Error:
			return nil, err
		default:
			return nil, errors.Format("database: %v", err)
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
	var filterCount = func(count uint, cancel context.CancelFunc, output chan<- Volume, input <-chan Volume) {
		defer cancel()
		defer close(output)
		for count >= 0 {
			v, ok := <-input
			if ok {
				output <- v
			} else {
				return
			}
			count--
		}
	}
	var filterLimited = func(lim bool, output chan<- Volume, input <-chan Volume) {
		defer close(output)
		for v := range input {
			// TODO
			output <- v
		}
	}
	var filterDeleted = func(del bool, output chan<- Volume, input <-chan Volume) {
		defer close(output)
		for v := range input {
			if del && v.Deleted {
				output <- v
			} else if !del && !v.Deleted {
				output <- v
			}
		}
	}
	var err error
	var ok bool
	var CS <-chan Volume
	var C1, C2 chan Volume //last 2 links in the chain of post-processing goroutines
	var sortDir string
	var afterTime time.Time
	var count uint
	var x interface{}

	C1 = make(chan Volume)
	C1save := C1
	C2 = make(chan Volume)

	if x = ctx.Value("sort-direction"); x == nil {
		ctx = context.WithValue(ctx, "sort-direction", "ASC")
	} else if sortDir, ok = x.(string); !ok {
		return nil, rserrors.NewBadInputError(`context value "sort-direction" was not string`)
	} else {
		sortDir = strings.ToUpper(sortDir)
		switch sortDir {
		case "ASC", "DESC":
		default:
			return nil, rserrors.NewBadInputError(`context value "sort-direction" was neither of: ASC, DESC`)
		}
	}

	if x = ctx.Value("after-time"); x != nil {
		if _, ok = x.(time.Time); !ok {
			return nil, rserrors.NewBadInputError(`context value "after-time" was not time.Time`)
		}
		afterTime = x.(time.Time)
	}

	if x = ctx.Value("count"); x != nil {
		count = 50
	} else if count, ok = x.(uint); !ok {
		return nil, rserrors.NewBadInputError(`context value "count" was not uint`)
	}

	var ctxCancel context.CancelFunc
	ctx, ctxCancel = context.WithCancel(ctx)
	go filterCount(count, ctxCancel, C2, C1)
	C1 = C2
	C2 = make(chan Volume)

	if x = ctx.Value("limited"); x != nil {
		var b bool
		if b, ok = x.(bool); !ok {
			return nil, rserrors.NewBadInputError(`context value "limited" was not bool`)
		}
		go filterLimited(b, C2, C1)
		C1 = C2
		C2 = make(chan Volume)
	}

	if x = ctx.Value("deleted"); x != nil {
		var b bool
		if b, ok = x.(bool); !ok {
			return nil, rserrors.NewBadInputError(`context value "deleted" was not bool`)
		}
		go filterDeleted(b, C2, C1)
		C1 = C2
		C2 = make(chan Volume)
	}

	CS, err = rs.db.VolumeListAllByTime(ctx, afterTime, count)
	if err != nil {
		switch err.(type) {
		case *rserrors.BadInputError, *rserrors.PermissionError, *errors.Error:
			return nil, err
		default:
			return nil, errors.Format("database: %v", err)
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

func (rs *ResourceSvc) GetNamespaceAccesses(ctx context.Context, userID, label string) (ns Namespace, err error) {
	ns, err = rs.db.NamespaceAccesses(userID, label)
	if err != nil {
		err = errors.Format("database: %v", err)
		ns = Namespace{}
		return
	}

	return
}

func (rs *ResourceSvc) GetVolumeAccesses(ctx context.Context, userID, label string) (vol Volume, err error) {
	vol, err = rs.db.VolumeAccesses(userID, label)
	if err != nil {
		err = errors.Format("database: %v", err)
		vol = Volume{}
		return
	}

	return
}

func (rs *ResourceSvc) Close() error {
	// close all closable resources
	v := reflect.ValueOf(rs.ResourceSvcClients)
	closer := reflect.TypeOf((*io.Closer)(nil))
	for i := 0; i < v.Type().NumField(); i++ {
		f := v.Field(i)
		if f.Type().Implements(closer) || f.Type().ConvertibleTo(closer) {
			if err := f.Convert(closer).Interface().(io.Closer).Close(); err != nil {
				return err
			}
		}
	}
	return nil
}
