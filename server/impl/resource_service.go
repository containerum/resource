package impl

import (
	"io"
	"reflect"

	"context"

	"errors"
	"fmt"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/cherrylog"
	"git.containerum.net/ch/resource-service/models"
	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

type resourceServiceImpl struct {
	server.ResourceServiceClients
	log *cherrylog.LogrusAdapter
}

// NewResourceServiceImpl creates a resource-service
func NewResourceServiceImpl(clients server.ResourceServiceClients) server.ResourceService {
	return &resourceServiceImpl{
		ResourceServiceClients: clients,
		log: cherrylog.NewLogrusAdapter(logrus.WithField("component", "resource_service")),
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
		return errors.New(fmt.Sprintf("%#v", errs))
	}
	return nil
}

func (rs *resourceServiceImpl) GetResourcesCount(ctx context.Context) (rstypes.GetResourcesCountResponse, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithField("user_id", userID).Info("get resources count")

	ret, err := rs.DB.GetResourcesCount(ctx, userID)

	return ret, err
}

func (rs *resourceServiceImpl) updateAccess(ctx context.Context, db models.DB, userID string) error {
	rs.log.WithField("user_id", userID).Info("update user accesses")

	accesses, err := db.GetUserResourceAccesses(ctx, userID)
	if err != nil {
		return err
	}
	return rs.Auth.UpdateUserAccess(ctx, userID, accesses)
}
