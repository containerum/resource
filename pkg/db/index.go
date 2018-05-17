package db

import (
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"git.containerum.net/ch/resource-service/pkg/util/strset"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

func (mongo *MongoStorage) InitIndexes() error {
	var errs []error
	for _, collectionName := range []string{CollectionDeployment, CollectionService, CollectionIngress} {
		var collection = mongo.db.C(collectionName)
		if err := collection.EnsureIndex(mgo.Index{
			Key: []string{"owner"},
		}); err != nil {
			errs = append(errs, err)
		}
		if err := collection.EnsureIndexKey(collectionName + "." + "name"); err != nil {
			errs = append(errs, err)
		}
		if err := collection.EnsureIndexKey("namespaceid"); err != nil {
			errs = append(errs, err)
		}
		if err := collection.EnsureIndex(mgo.Index{
			Name: "alive_" + collectionName,
			Key:  []string{collectionName + "." + "name", "namespaceid"},
			PartialFilter: bson.M{
				"deleted": false,
			},
			Unique: true,
		}); err != nil {
			errs = append(errs, err)
		}
		if err := collection.EnsureIndex(mgo.Index{
			Name: "deleted_" + collectionName,
			Key:  []string{collectionName + "." + "name", "namespaceid"},
			PartialFilter: bson.M{
				"deleted": true,
			},
			Unique: false,
		}); err != nil {
			errs = append(errs, err)
		}
		if err := collection.EnsureIndexKey("deleted"); err != nil {
			errs = append(errs, err)
		}
	}
	{
		var collection = mongo.db.C(CollectionService)
		if err := collection.EnsureIndexKey(CollectionService + "." + "deployment"); err != nil {
			errs = append(errs, err)
		}
		if err := collection.EnsureIndexKey(CollectionService + "." + "domain"); err != nil {
			errs = append(errs, err)
		}
		var portIndex = mgo.Index{
			Key: []string{
				CollectionService + "." + "domain",
				CollectionService + "." + "ports.port",
				CollectionService + "." + "ports.protocol",
			},
			Background: true,
			Sparse:     true,
			Unique:     true,
		}
		if err := collection.EnsureIndex(portIndex); err != nil {
			errs = append(errs, err)
		}
	}
	{
		var collection = mongo.db.C(CollectionDomain)
		if err := collection.EnsureIndexKey("domain"); err != nil {
			errs = append(errs, err)
		}
		if err := collection.EnsureIndexKey("domain_group"); err != nil {
			errs = append(errs, err)
		}
		if err := collection.EnsureIndexKey("domain_group", "domain"); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return rserrors.ErrDatabase().AddDetailsErr(errs...)
	}
	return nil
}

func (mongo *MongoStorage) CreateIndex(indexName string, options ...func(mongo *MongoStorage, cName, indexName string) (bool, error)) error {
	dbCollections, err := mongo.db.CollectionNames()
	if err != nil {
		return err
	}
	for _, cName := range dbCollections {
		for _, option := range options {
			if ok, err := option(mongo, cName, indexName); !ok || err != nil {
				return err
			}
		}
		collection := mongo.db.C(cName)
		var index = mgo.Index{
			Key:    []string{indexName},
			Unique: true,
		}
		if collection.EnsureIndex(index); err != nil {
			return PipErr{err}.ToMongerr().Extract()
		}
	}
	return nil
}

func DropIndexIfExists(mongo *MongoStorage, cName, indexName string) (bool, error) {
	indexes, err := mongo.db.C(cName).Indexes()
	if err != nil {
		return false, err
	}
	var indexNames = make([]string, 0, len(indexes))
	for _, index := range indexes {
		indexNames = append(indexNames, index.Name)
	}
	var collection = mongo.db.C(cName)
	if strset.FromSlice(indexNames).In(indexName) {
		if err := collection.DropIndexName(indexName); err != nil {
			return false, PipErr{err}.ToMongerr().Extract()
		}
	}
	return true, nil
}