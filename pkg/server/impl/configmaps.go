package impl

import (
	"context"

	"git.containerum.net/ch/resource-service/pkg/clients"
	"git.containerum.net/ch/resource-service/pkg/db"
	"git.containerum.net/ch/resource-service/pkg/models/configmap"
	"git.containerum.net/ch/resource-service/pkg/util/coblog"
	"github.com/containerum/cherry/adaptors/cherrylog"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type ConfigMapsActionsImpl struct {
	kube  clients.Kube
	mongo *db.MongoStorage
	log   *cherrylog.LogrusAdapter
}

func NewConfigMapsActionsImpl(mongo *db.MongoStorage, kube *clients.Kube) *ConfigMapsActionsImpl {
	return &ConfigMapsActionsImpl{
		kube:  *kube,
		mongo: mongo,
		log:   cherrylog.NewLogrusAdapter(logrus.WithField("component", "configmaps_actions")),
	}
}

func (ia *ConfigMapsActionsImpl) GetConfigMapsList(ctx context.Context, nsID string) (*configmap.ConfigMapsResponse, error) {
	userID := httputil.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"namespace": nsID,
	}).Info("get user configmaps")

	cms, err := ia.mongo.GetConfigMapList(nsID)
	if err != nil {
		return nil, err
	}

	return &configmap.ConfigMapsResponse{ConfigMaps: cms}, nil
}

func (ia *ConfigMapsActionsImpl) GetConfigMap(ctx context.Context, nsID, cmName string) (*configmap.Resource, error) {
	ia.log.Info("get configmap")

	resp, err := ia.mongo.GetConfigMap(nsID, cmName)

	return &resp, err
}

func (ia *ConfigMapsActionsImpl) CreateConfigMap(ctx context.Context, nsID string, req kubtypes.ConfigMap) (*configmap.Resource, error) {
	userID := httputil.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id": userID,
		"ns_id":   nsID,
	}).Info("create configmap")
	coblog.Std.Struct(req)

	createdCM, err := ia.mongo.CreateConfigMap(configmap.FromKube(nsID, userID, req))
	if err != nil {
		return nil, err
	}

	if err := ia.kube.CreateConfigMap(ctx, nsID, req); err != nil {
		ia.log.Debug("Kube-API error! Deleting configmap from DB.")
		if err := ia.mongo.DeleteConfigMap(nsID, req.Name); err != nil {
			return nil, err
		}
		return nil, err
	}

	return &createdCM, nil
}

func (ia *ConfigMapsActionsImpl) ImportConfigMap(ctx context.Context, nsID string, req kubtypes.ConfigMap) error {
	userID := httputil.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id": userID,
		"ns_id":   nsID,
	}).Info("import configmap")
	coblog.Std.Struct(req)

	_, err := ia.mongo.CreateConfigMap(configmap.FromKube(nsID, userID, req))
	if err != nil {
		return err
	}

	return nil
}

func (ia *ConfigMapsActionsImpl) DeleteConfigMap(ctx context.Context, nsID, cmName string) error {
	userID := httputil.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id": userID,
		"ns_id":   nsID,
		"cm":      cmName,
	}).Info("delete configmap")

	if err := ia.mongo.DeleteConfigMap(nsID, cmName); err != nil {
		return err
	}

	if err := ia.kube.DeleteConfigMap(ctx, nsID, cmName); err != nil {
		ia.log.Debug("Kube-API error! Reverting changes.")
		if err := ia.mongo.RestoreConfigMap(nsID, cmName); err != nil {
			return err
		}
		return err
	}

	return nil
}

func (ia *ConfigMapsActionsImpl) DeleteAllConfigMaps(ctx context.Context, nsID string) error {
	ia.log.WithFields(logrus.Fields{
		"ns_id": nsID,
	}).Info("delete all configmaps")

	if err := ia.mongo.DeleteAllConfigMapsInNamespace(nsID); err != nil {
		return err
	}

	return nil
}
