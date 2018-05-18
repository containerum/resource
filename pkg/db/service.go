package db

import (
	"git.containerum.net/ch/resource-service/pkg/models/service"
	"git.containerum.net/ch/resource-service/pkg/models/stats"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

func (mongo *MongoStorage) GetService(namespaceID, serviceName string) (service.Service, error) {
	mongo.logger.Debugf("getting service")
	var collection = mongo.db.C(CollectionService)
	var result service.Service
	var err error
	if err = collection.Find(service.OneSelectQuery(namespaceID, serviceName)).One(&result); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get service")
		if err == mgo.ErrNotFound {
			return result, rserrors.ErrResourceNotExists().AddDetailsErr(err)
		}
		return result, PipErr{err}.ToMongerr().Extract()
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
		return result, PipErr{err}.ToMongerr().NotFoundToNil().Extract()
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
		if mgo.IsDup(err) {
			return service, rserrors.ErrResourceAlreadyExists()
		}
		return service, PipErr{err}.ToMongerr().Extract()
	}
	return service, nil
}

func (mongo *MongoStorage) UpdateService(upd service.Service) (service.Service, error) {
	mongo.logger.Debugf("updating service")
	var collection = mongo.db.C(CollectionService)
	if err := collection.Update(upd.OneSelectQuery(), upd.UpdateQuery()); err != nil {
		mongo.logger.WithError(err).Errorf("unable to update service")
		return upd, PipErr{err}.ToMongerr().Extract()
	}
	return upd, nil
}

func (mongo *MongoStorage) DeleteService(namespaceID, name string) error {
	mongo.logger.Debugf("deleting service")
	var collection = mongo.db.C(CollectionService)
	err := collection.Update(service.Service{
		Service: model.Service{
			Name: name,
		},
		NamespaceID: namespaceID,
	}.OneSelectQuery(),
		bson.M{
			"$set": bson.M{"deleted": true},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete service")
		if err == mgo.ErrNotFound {
			return rserrors.ErrResourceNotExists()
		}
		return PipErr{err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) DeleteAllServices(namespaceID string) error {
	mongo.logger.Debugf("deleting all services in namespace")
	var collection = mongo.db.C(CollectionService)
	_, err := collection.UpdateAll(service.Service{
		NamespaceID: namespaceID,
	}.AllSelectQuery(),
		bson.M{
			"$set": bson.M{"deleted": true},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete service")
		return PipErr{err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) CountServices(owner string) (stats.Service, error) {
	mongo.logger.Debugf("counting services")
	var collection = mongo.db.C(CollectionService)
	var statData []struct {
		HasDomain bool `bson:"_id"`
		Count     int  `bson:"count"`
	}
	if err := collection.Pipe([]bson.M{
		{"$match": bson.M{
			"owner":   owner,
			"deleted": false,
		}},
		{"$project": bson.M{
			"domain": "$service.domain",
		}},
		{"$group": bson.M{
			"_id":   bson.M{"$eq": []interface{}{"$domain", ""}},
			"count": bson.M{"$sum": 1},
		}},
	}).All(&statData); err != nil {
		return stats.Service{}, PipErr{err}.ToMongerr().NotFoundToNil().Extract()
	}
	var serviceStats stats.Service
	for _, serv := range statData {
		if serv.HasDomain {
			serviceStats.External += serv.Count
		} else {
			serviceStats.Internal += serv.Count
		}
	}
	return serviceStats, nil
}

func (mongo *MongoStorage) CountServicesInNamespace(namespaceID string) (stats.Service, error) {
	mongo.logger.Debugf("counting services in namespace")
	var collection = mongo.db.C(CollectionService)
	var statData []struct {
		HasDomain bool `bson:"_id"`
		Count     int  `bson:"count"`
	}
	if err := collection.Pipe([]bson.M{
		{"$match": bson.M{
			"namespaceid": namespaceID,
			"deleted":     false,
		}},
		{"$project": bson.M{
			"domain": "$service.domain",
		}},
		{"$group": bson.M{
			"_id":   bson.M{"$eq": []interface{}{"$domain", ""}},
			"count": bson.M{"$sum": 1},
		}},
	}).All(&statData); err != nil {
		return stats.Service{}, PipErr{err}.ToMongerr().NotFoundToNil().Extract()
	}
	var serviceStats stats.Service
	for _, serv := range statData {
		if serv.HasDomain {
			serviceStats.External += serv.Count
		} else {
			serviceStats.Internal += serv.Count
		}
	}
	return serviceStats, nil
}
