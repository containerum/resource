package db

import (
	"time"

	"git.containerum.net/ch/resource-service/pkg/models/service"
	"git.containerum.net/ch/resource-service/pkg/models/stats"
	"git.containerum.net/ch/resource-service/pkg/rserrors"
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

func (mongo *MongoStorage) GetService(namespaceID, serviceName string) (service.ResourceService, error) {
	mongo.logger.Debugf("getting service")
	var collection = mongo.db.C(CollectionService)
	var result service.ResourceService
	var err error
	if err = collection.Find(service.OneSelectQuery(namespaceID, serviceName)).One(&result); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get service")
		if err == mgo.ErrNotFound {
			return result, rserrors.ErrResourceNotExists().AddDetails(serviceName)
		}
		return result, PipErr{error: err}.ToMongerr().Extract()
	}
	return result, nil
}

func (mongo *MongoStorage) GetServiceList(namespaceID string) (service.ListService, error) {
	mongo.logger.Debugf("getting services list")
	var collection = mongo.db.C(CollectionService)
	result := make(service.ListService, 0)
	if err := collection.Find(bson.M{
		"namespaceid": namespaceID,
		"deleted":     false,
	}).All(&result); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get service list")
		return result, PipErr{error: err}.ToMongerr().NotFoundToNil().Extract()
	}
	return result, nil
}

// If ID is empty, then generates UUID4 and uses it
func (mongo *MongoStorage) CreateService(service service.ResourceService) (service.ResourceService, error) {
	mongo.logger.Debugf("creating service")
	var collection = mongo.db.C(CollectionService)
	if service.ID == "" {
		service.ID = uuid.New().String()
	}
	service.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := collection.Insert(service); err != nil {
		mongo.logger.WithError(err).Errorf("unable to create service")
		if mgo.IsDup(err) {
			return service, rserrors.ErrResourceAlreadyExists()
		}
		return service, PipErr{error: err}.ToMongerr().Extract()
	}
	return service, nil
}

func (mongo *MongoStorage) UpdateService(upd service.ResourceService) (service.ResourceService, error) {
	mongo.logger.Debugf("updating service")
	var collection = mongo.db.C(CollectionService)
	if err := collection.Update(upd.OneSelectQuery(), upd.UpdateQuery()); err != nil {
		mongo.logger.WithError(err).Errorf("unable to update service")
		return upd, PipErr{error: err}.ToMongerr().Extract()
	}
	return upd, nil
}

func (mongo *MongoStorage) DeleteService(namespaceID, name string) error {
	mongo.logger.Debugf("deleting service")
	var collection = mongo.db.C(CollectionService)
	err := collection.Update(service.ResourceService{
		Service: model.Service{
			Name: name,
		},
		NamespaceID: namespaceID,
	}.OneSelectQuery(),
		bson.M{
			"$set": bson.M{"deleted": true,
				"service.deletedat": time.Now().UTC().Format(time.RFC3339)},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete service")
		if err == mgo.ErrNotFound {
			return rserrors.ErrResourceNotExists().AddDetails(name)
		}
		return PipErr{error: err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) RestoreService(namespaceID, name string) error {
	mongo.logger.Debugf("restoring service")
	var collection = mongo.db.C(CollectionService)
	err := collection.Update(service.ResourceService{
		Service: model.Service{
			Name: name,
		},
		NamespaceID: namespaceID,
	}.OneSelectDeletedQuery(),
		bson.M{
			"$set": bson.M{"deleted": true,
				"service.deletedat": ""},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to restore service")
		if err == mgo.ErrNotFound {
			return rserrors.ErrResourceNotExists().AddDetails(name)
		}
		return PipErr{error: err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) DeleteAllServicesInNamespace(namespaceID string) error {
	mongo.logger.Debugf("deleting all services in namespace")
	var collection = mongo.db.C(CollectionService)
	_, err := collection.UpdateAll(service.ResourceService{
		NamespaceID: namespaceID,
	}.AllSelectQuery(),
		bson.M{
			"$set": bson.M{"deleted": true,
				"service.deletedat": time.Now().UTC().Format(time.RFC3339)},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete service")
		return PipErr{error: err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) DeleteAllServicesByOwner(owner string) error {
	mongo.logger.Debugf("deleting all services in namespace")
	var collection = mongo.db.C(CollectionService)
	_, err := collection.UpdateAll(service.ResourceService{
		Service: model.Service{Owner: owner},
	}.AllSelectOwnerQuery(),
		bson.M{
			"$set": bson.M{"deleted": true,
				"service.deletedat": time.Now().UTC().Format(time.RFC3339)},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete services")
		return PipErr{error: err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) DeleteAllServicesBySolutionName(nsID, solution string) error {
	mongo.logger.Debugf("deleting all solutions services")
	var collection = mongo.db.C(CollectionService)
	_, err := collection.UpdateAll(bson.M{
		"namespaceid":        nsID,
		"service.solutionid": solution,
	},
		bson.M{
			"$set": bson.M{"deleted": true,
				"service.deletedat": time.Now().UTC().Format(time.RFC3339)},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete solution services")
	}
	return PipErr{error: err}.ToMongerr().Extract()
}

func (mongo *MongoStorage) CountServices(owner string) (stats.Service, error) {
	mongo.logger.Debugf("counting services")
	var collection = mongo.db.C(CollectionService)
	var statData []struct {
		NoDomain bool `bson:"_id"`
		Count    int  `bson:"count"`
	}
	if err := collection.Pipe([]bson.M{
		{"$match": bson.M{
			"service.owner": owner,
			"deleted":       false,
		}},
		{"$project": bson.M{
			"domain": "$service.domain",
		}},
		{"$group": bson.M{
			"_id":   bson.M{"$eq": []interface{}{"$domain", ""}},
			"count": bson.M{"$sum": 1},
		}},
	}).All(&statData); err != nil {
		return stats.Service{}, PipErr{error: err}.ToMongerr().NotFoundToNil().Extract()
	}
	var serviceStats stats.Service
	for _, serv := range statData {
		if serv.NoDomain {
			serviceStats.External += serv.Count
		} else {
			serviceStats.Internal += serv.Count
		}
	}
	return serviceStats, nil
}

func (mongo *MongoStorage) CountAllServices() (stats.Service, error) {
	mongo.logger.Debugf("counting services")
	var collection = mongo.db.C(CollectionService)
	var statData []struct {
		NoDomain bool `bson:"_id"`
		Count    int  `bson:"count"`
	}
	if err := collection.Pipe([]bson.M{
		{"$match": bson.M{
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
		return stats.Service{}, PipErr{error: err}.ToMongerr().NotFoundToNil().Extract()
	}
	var serviceStats stats.Service
	for _, serv := range statData {
		if serv.NoDomain {
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
		return stats.Service{}, PipErr{error: err}.ToMongerr().NotFoundToNil().Extract()
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
