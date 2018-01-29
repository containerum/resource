package impl

import (
	"io"
	"reflect"

	"context"

	"time"

	"git.containerum.net/ch/grpc-proto-files/auth"
	"git.containerum.net/ch/json-types/billing"
	"git.containerum.net/ch/json-types/errors"
	"git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

type resourceServiceImpl struct {
	server.ResourceServiceClients
	log *logrus.Entry
}

// NewResourceServiceImpl creates a resource-service
func NewResourceServiceImpl(clients server.ResourceServiceClients) server.ResourceService {
	return &resourceServiceImpl{
		ResourceServiceClients: clients,
		log: logrus.WithField("component", "resource_service"),
	}
}

func (rs *resourceServiceImpl) Close() error {
	var errs []string
	v := reflect.ValueOf(rs.ResourceServiceClients)
	for i := 0; i < v.NumField(); i++ {
		if closer, ok := v.Field(i).Interface().(io.Closer); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, closer.Close().Error())
			}
		}
	}
	if len(errs) > 0 {
		return errors.Format("%#v", errs)
	}
	return nil
}

func (rs *resourceServiceImpl) filterNamespace(isAdmin bool, ns *resource.NamespaceWithPermission) {
	if !isAdmin {
		ns.ID = ""
		ns.Limited = nil
		ns.NewAccessLevel = ns.AccessLevel
		ns.NewAccessLevel = ""
		ns.CreateTime = time.Time{}
		ns.Deleted = nil
		ns.DeleteTime.IsNull = true
		ns.AccessLevelChangeTime = time.Time{}
	}
}

func (rs *resourceServiceImpl) filterVolume(isAdmin bool, vol *resource.VolumeWithPermission) {
	if !isAdmin {
		vol.ID = ""
		vol.Limited = nil
		vol.NewAccessLevel = vol.AccessLevel
		vol.NewAccessLevel = ""
		vol.Deleted = nil
		vol.DeleteTime.IsNull = true
		vol.AccessLevelChangeTime = time.Time{}
		vol.CreateTime = time.Time{}
		vol.Replicas = 0
	}
}

func (rs *resourceServiceImpl) filterNamespaceWithVolume(isAdmin bool, nsvol *resource.NamespaceWithVolumes) {
	rs.filterNamespace(isAdmin, &nsvol.NamespaceWithPermission)
	for i := range nsvol.Volume {
		rs.filterVolume(isAdmin, &nsvol.Volume[i])
	}
}

func checkTariff(tariff billing.Tariff, isAdmin bool) error {
	if !tariff.Active {
		return server.ErrTariffInactive
	}
	if !isAdmin && !tariff.Public {
		return server.ErrTariffNotPublic
	}

	return nil
}

func (rs *resourceServiceImpl) filterVolumes(isAdmin bool, volumes []resource.VolumeWithPermission) {
	for i := range volumes {
		rs.filterVolume(isAdmin, &volumes[i])
	}
}

func (rs *resourceServiceImpl) GetUserAccesses(ctx context.Context) (*auth.ResourcesAccess, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithField("user_id", userID).Info("get all user accesses")

	ret, err := rs.DB.GetUserResourceAccesses(ctx, userID)
	if err != nil {
		err = server.HandleDBError(err)
		return nil, err
	}

	return ret, nil
}
