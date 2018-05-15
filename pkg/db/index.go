package db

import (
	"git.containerum.net/ch/resource-service/pkg/util/strset"
	"github.com/globalsign/mgo"
)

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
			return err
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
			return false, err
		}
	}
	return true, nil
}
