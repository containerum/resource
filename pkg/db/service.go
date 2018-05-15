package db

import (
	"git.containerum.net/ch/resource-service/pkg/models/service"
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
)

func (mongo *MongoStorage) GetService(namespaceID, serviceName string) (service.Service, error) {
	var collection = mongo.db.C(CollectionService)
	var result service.Service
	if err := collection.Find(service.Service{
		NamespaceID: namespaceID,
		Service: model.Service{
			Name: serviceName,
		},
	}.SelectQuery()).One(&result); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get service")
		return result, err
	}
	return result, nil
}

func (mongo *MongoStorage) GetServiceList(namespaceID string) (service.ServiceList, error) {
	var collection = mongo.db.C(CollectionService)
	var result service.ServiceList
	if err := collection.Find(bson.M{
		"namespaceid": namespaceID,
		"deleted":     false,
	}).All(&result); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get service list")
		return result, err
	}
	return result, nil
}

func (mongo *MongoStorage) CreateService(service service.Service) (service.Service, error) {
	var collection = mongo.db.C(CollectionService)
	if err := collection.Insert(service); err != nil {
		mongo.logger.WithError(err).Errorf("unable to create service")
		return service, err
	}
	return service, nil
}

func (mongo *MongoStorage) UpdateService(upd service.Service) (service.Service, error) {
	var collection = mongo.db.C(CollectionService)
	if err := collection.Update(upd.SelectQuery(), upd.UpdateQuery()); err != nil {
		mongo.logger.WithError(err).Errorf("unable to update service")
		return upd, err
	}
	return upd, nil
}

func (mongo *MongoStorage) DeleteService(namespaceID, name string) error {
	var collection = mongo.db.C(CollectionService)
	if err := collection.Update(service.Service{
		NamespaceID: namespaceID,
		Service: model.Service{
			Name: name,
		},
	}.SelectQuery(), bson.M{"deleted": true}); err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete service")
		return err
	}
	return nil
}
