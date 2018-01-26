package impl

import (
	"io"
	"reflect"

	"git.containerum.net/ch/json-types/errors"
	"git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/server"
	"github.com/sirupsen/logrus"
)

type resourceServiceImpl struct {
	server.ResourceServiceClients
	log *logrus.Entry
}

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
	}
}

func (rs *resourceServiceImpl) filterVolume(isAdmin bool, vol *resource.VolumeWithPermission) {
	if !isAdmin {
		vol.ID = ""
		vol.Limited = nil
		vol.NewAccessLevel = vol.AccessLevel
		vol.NewAccessLevel = ""
	}
}

func (rs *resourceServiceImpl) filterNamespaceWithVolume(isAdmin bool, nsvol *resource.NamespaceWithVolumes) {
	rs.filterNamespace(isAdmin, &nsvol.NamespaceWithPermission)
	for i := range nsvol.Volume {
		rs.filterVolume(isAdmin, &nsvol.Volume[i])
	}
}
