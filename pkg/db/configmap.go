package db

import (
	"time"

	"git.containerum.net/ch/resource-service/pkg/models/configmap"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

func (mongo *MongoStorage) GetConfigMap(namespaceID, cmName string) (configmap.ResourceConfigMap, error) {
	mongo.logger.Debugf("getting configmap")
	var collection = mongo.db.C(CollectionCM)
	var result configmap.ResourceConfigMap
	var err error
	if err = collection.Find(configmap.OneSelectQuery(namespaceID, cmName)).One(&result); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get configmap")
		if err == mgo.ErrNotFound {
			return result, rserrors.ErrResourceNotExists().AddDetails(cmName)
		}
		return result, PipErr{error: err}.ToMongerr().Extract()
	}
	return result, nil
}

func (mongo *MongoStorage) GetConfigMapList(namespaceID string) (configmap.ListConfigMaps, error) {
	mongo.logger.Debugf("getting configmaps list")
	var collection = mongo.db.C(CollectionCM)
	result := make(configmap.ListConfigMaps, 0)
	if err := collection.Find(bson.M{
		"namespaceid": namespaceID,
		"deleted":     false,
	}).All(&result); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get configmaps list")
		return result, PipErr{error: err}.ToMongerr().NotFoundToNil().Extract()
	}
	return result, nil
}

func (mongo *MongoStorage) GetSelectedConfigMaps(namespaceID []string) (configmap.ListConfigMaps, error) {
	mongo.logger.Debugf("getting selected configmaps")
	var collection = mongo.db.C(CollectionCM)
	list := make(configmap.ListConfigMaps, 0)
	if err := collection.Find(bson.M{
		"namespaceid": bson.M{
			"$in": namespaceID,
		},
		"deleted": false,
	}).All(&list); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get configmaps")
		if err == mgo.ErrNotFound {
			return list, rserrors.ErrResourceNotExists()
		}
		return list, PipErr{error: err}.ToMongerr().Extract()
	}
	return list, nil
}

// If ID is empty, then generates UUID4 and uses it
func (mongo *MongoStorage) CreateConfigMap(cm configmap.ResourceConfigMap) (configmap.ResourceConfigMap, error) {
	mongo.logger.Debugf("creating configmap")
	var collection = mongo.db.C(CollectionCM)
	if cm.ID == "" {
		cm.ID = uuid.New().String()
	}
	cm.Data = nil
	cm.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := collection.Insert(cm); err != nil {
		mongo.logger.WithError(err).Errorf("unable to create configmap")
		if mgo.IsDup(err) {
			return cm, rserrors.ErrResourceAlreadyExists().AddDetailsErr(err)
		}
		return cm, PipErr{error: err}.ToMongerr().Extract()
	}
	return cm, nil
}

func (mongo *MongoStorage) DeleteConfigMap(namespaceID, name string) error {
	mongo.logger.Debugf("deleting configmap")
	var collection = mongo.db.C(CollectionCM)
	err := collection.Update(configmap.ResourceConfigMap{
		ConfigMap: model.ConfigMap{
			Name: name,
		},
		NamespaceID: namespaceID,
	}.OneSelectQuery(),
		bson.M{
			"$set": bson.M{"deleted": true,
				"configmap.deletedat": time.Now().UTC().Format(time.RFC3339)},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete configmap")
		if err == mgo.ErrNotFound {
			return rserrors.ErrResourceNotExists().AddDetails(name)
		}
		return PipErr{error: err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) DeleteAllConfigMapsInNamespace(namespaceID string) error {
	mongo.logger.Debugf("deleting all configmaps in namespace")
	var collection = mongo.db.C(CollectionCM)
	_, err := collection.UpdateAll(configmap.ResourceConfigMap{
		NamespaceID: namespaceID,
	}.AllSelectQuery(),
		bson.M{
			"$set": bson.M{"deleted": true,
				"configmap.deletedat": time.Now().UTC().Format(time.RFC3339)},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete configmap")
		return PipErr{error: err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) DeleteAllConfigMapsByOwner(owner string) error {
	mongo.logger.Debugf("deleting all configmaps in namespace")
	var collection = mongo.db.C(CollectionCM)
	_, err := collection.UpdateAll(configmap.ResourceConfigMap{
		ConfigMap: model.ConfigMap{Owner: owner},
	}.AllSelectOwnerQuery(),
		bson.M{
			"$set": bson.M{"deleted": true,
				"configmap.deletedat": time.Now().UTC().Format(time.RFC3339)},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete configmaps")
		return PipErr{error: err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) RestoreConfigMap(namespaceID, name string) error {
	mongo.logger.Debugf("restoring configmap")
	var collection = mongo.db.C(CollectionCM)
	err := collection.Update(configmap.ResourceConfigMap{
		ConfigMap: model.ConfigMap{
			Name: name,
		},
		NamespaceID: namespaceID,
	}.OneSelectDeletedQuery(),
		bson.M{
			"$set": bson.M{"deleted": false,
				"configmap.deletedat": ""},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to restore configmap")
		if err == mgo.ErrNotFound {
			return rserrors.ErrResourceNotExists().AddDetails(name)
		}
		return PipErr{error: err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) CountConfigMaps(owner string) (int, error) {
	mongo.logger.Debugf("counting configmaps")
	var collection = mongo.db.C(CollectionCM)
	n, err := collection.Find(bson.M{
		"configmap.owner": owner,
		"deleted":         false,
	}).Count()
	if err != nil {
		return 0, PipErr{error: err}.ToMongerr().NotFoundToNil().Extract()
	}
	return n, nil
}

func (mongo *MongoStorage) CountAllConfigMaps() (int, error) {
	mongo.logger.Debugf("counting all configmaps")
	var collection = mongo.db.C(CollectionCM)
	n, err := collection.Find(bson.M{
		"deleted": false,
	}).Count()
	if err != nil {
		return 0, PipErr{error: err}.ToMongerr().NotFoundToNil().Extract()
	}
	return n, nil
}
