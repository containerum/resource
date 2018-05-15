package db

import (
	"git.containerum.net/ch/resource-service/pkg/models/service"
	"github.com/globalsign/mgo/bson"
)

func (mongo *MongoStorage) GetService(owner, namespaceID, serviceName string) (service.Service, error) {
	var collection = mongo.db.C(CollectionService)
	colQuerier := bson.M{"name": serviceName, "namespaceid": namespaceID}
	result := service.Service{}
	if err := collection.Find(colQuerier).One(&result); err != nil {
		return result, err
	}
	return result, nil
}

func (mongo *MongoStorage) CreateService(service service.Service) (service.Service, error) {
	var collection = mongo.db.C(CollectionService)
	if err := collection.Insert(service); err != nil {
		return service, err
	}
	return service, nil
}

func (mongo *MongoStorage) UpdateService(service service.Service) (service.Service, error) {
	var collection = mongo.db.C(CollectionService)
	colQuerier := bson.M{"name": service.Name, "namespaceid": service.NamespaceID}
	if err := collection.Update(colQuerier, service); err != nil {
		return service, err
	}
	return service, nil
}
