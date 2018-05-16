package db

import (
	"fmt"

	"git.containerum.net/ch/resource-service/pkg/models/service"
	"git.containerum.net/ch/resource-service/pkg/models/stats"
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

func (mongo *MongoStorage) GetService(namespaceID, serviceName string) (service.Service, error) {
	mongo.logger.Debugf("getting service")
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
	mongo.logger.Debugf("getting service list")
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

// If ID is empty, then generates UUID4 and uses it
func (mongo *MongoStorage) CreateService(service service.Service) (service.Service, error) {
	mongo.logger.Debugf("creating service")
	var collection = mongo.db.C(CollectionService)
	if service.ID == "" {
		service.ID = uuid.New().String()
	}
	if err := collection.Insert(service); err != nil {
		mongo.logger.WithError(err).Errorf("unable to create service")
		return service, err
	}
	return service, nil
}

func (mongo *MongoStorage) UpdateService(upd service.Service) (service.Service, error) {
	mongo.logger.Debugf("updating service")
	var collection = mongo.db.C(CollectionService)
	if err := collection.Update(upd.SelectQuery(), upd.UpdateQuery()); err != nil {
		mongo.logger.WithError(err).Errorf("unable to update service")
		return upd, err
	}
	return upd, nil
}

func (mongo *MongoStorage) DeleteService(namespaceID, name string) error {
	mongo.logger.Debugf("deleting service")
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

func (mongo *MongoStorage) CountService(owner string) (stats.Service, error) {
	mongo.logger.Debugf("counting deployment")
	var collection = mongo.db.C(CollectionService)
	var statData []struct {
		HasDomain bool `json:"_id"`
		Count     int  `json:"count"`
	}
	if err := collection.Pipe([]bson.M{
		{"$match": bson.M{
			"owner": owner,
		}},
		{"$project": bson.M{"domain": "$service.domain"}},
		{
			"$group": bson.M{
				"_id":   bson.M{"$ne": []interface{}{"domain", ""}},
				"count": bson.M{"$sum": 1},
			},
		},
	}).All(&statData); err != nil {
		return stats.Service{}, err
	}
	var serviceStats stats.Service
	fmt.Println("TEST", statData)
	for _, serv := range statData {
		if serv.HasDomain {
			serviceStats.External += serv.Count
		} else {
			serviceStats.Internal += serv.Count
		}
	}
	return serviceStats, nil
}
