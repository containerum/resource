package db

import (
	"fmt"

	"os"
	"text/tabwriter"

	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"git.containerum.net/ch/resource-service/pkg/util/strset"
	"github.com/blang/semver"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

const (
	errDBVersion    = "db version (%v) is newer that application version (%v). Run application with '--force' to forcefully update db to application version"
	errOldDBVersion = "unable to parse db version. Run application with '--force' to forcefully update db to application version"
)

func (mongo *MongoStorage) InitIndexes(dbversion string, forceupdate bool) error {
	var errs []error

	var collection = mongo.db.C(CollectionDB)
	var dbinfo map[string]string
	var err error
	if err = collection.Find(nil).One(&dbinfo); err != nil {
		mongo.logger.WithError(err).Infoln("no db version set")
		if err == mgo.ErrNotFound {
			err := collection.Insert(map[string]string{"version": dbversion})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	newversionsv, err := semver.ParseTolerant(dbversion)
	if err != nil {
		return err
	}

	olddbversion := dbinfo["version"]

	if forceupdate || olddbversion == "" {
		olddbversion = "0"
	}

	oldversionsv, err := semver.ParseTolerant(olddbversion)
	if err != nil {
		return fmt.Errorf(errOldDBVersion)
	}

	switch newversionsv.Compare(oldversionsv) {
	case 0:
		mongo.logger.Infoln("no need to update db indexes")
	case -1:
		return fmt.Errorf(errDBVersion, olddbversion, dbversion)
	default:
		mongo.logger.Infof("updating db version from %v to %v", olddbversion, dbversion)

		mongo.logger.Infoln("updating db indexes")
		for _, collectionName := range CollectionsNames() {
			if err := mongo.db.C(collectionName).DropAllIndexes(); err != nil {
				return err
			}
		}
		for _, collectionName := range []string{CollectionDeployment, CollectionService, CollectionIngress} {
			var collection = mongo.db.C(collectionName)
			if err := collection.EnsureIndex(mgo.Index{
				Key: []string{collectionName + ".owner"},
			}); err != nil {
				errs = append(errs, err)
			}
			if err := collection.EnsureIndexKey(collectionName + ".name"); err != nil {
				errs = append(errs, err)
			}
			if err := collection.EnsureIndexKey("namespaceid"); err != nil {
				errs = append(errs, err)
			}
			if err := collection.EnsureIndexKey("deleted"); err != nil {
				errs = append(errs, err)
			}
		}
		{
			var collection = mongo.db.C(CollectionDeployment)
			if err := collection.EnsureIndex(mgo.Index{
				Name: "alive_" + CollectionDeployment,
				Key:  []string{CollectionDeployment + ".name", "namespaceid"},
				PartialFilter: bson.M{
					"deleted":           false,
					"deployment.active": true,
				},
				Unique: true,
			}); err != nil {
				errs = append(errs, err)
			}
			if err := collection.EnsureIndexKey("active"); err != nil {
				errs = append(errs, err)
			}
			if err := collection.EnsureIndex(mgo.Index{
				Name: "unique_version_" + CollectionDeployment,
				Key:  []string{CollectionDeployment + ".name", "namespaceid", CollectionDeployment + ".version"},
				PartialFilter: bson.M{
					"deleted": false,
				},
				Unique: true,
			}); err != nil {
				errs = append(errs, err)
			}
		}
		{
			var collection = mongo.db.C(CollectionIngress)
			if err := collection.EnsureIndex(mgo.Index{
				Name: "alive_" + CollectionIngress,
				Key:  []string{CollectionIngress + ".name"},
				PartialFilter: bson.M{
					"deleted": false,
				},
				Unique: true,
			}); err != nil {
				errs = append(errs, err)
			}
		}
		{
			var collection = mongo.db.C(CollectionService)
			if err := collection.EnsureIndexKey(CollectionService + "_deployment"); err != nil {
				errs = append(errs, err)
			}
			if err := collection.EnsureIndexKey(CollectionService + "_domain"); err != nil {
				errs = append(errs, err)
			}
			if err := collection.EnsureIndex(mgo.Index{
				Name: "alive_" + CollectionService,
				Key:  []string{CollectionService + ".name", "namespaceid"},
				PartialFilter: bson.M{
					"deleted": false,
				},
				Unique: true,
			}); err != nil {
				errs = append(errs, err)
			}
			if err := collection.EnsureIndex(mgo.Index{
				Name: "alive_" + CollectionService + "_with_ports",
				Key: []string{
					CollectionService + ".domain",
					CollectionService + ".ports.port",
					CollectionService + ".ports.protocol",
				},
				PartialFilter: bson.M{
					"deleted": false,
					"type":    "external",
					CollectionService + ".ports.port": bson.M{
						"$exists": true,
					},
				},
				Unique: true,
			}); err != nil {
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
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.TabIndent|tabwriter.Debug)
		for _, collectionName := range CollectionsNames() {
			var collection = mongo.db.C(collectionName)
			var indexes, err = collection.Indexes()
			if err != nil {
				errs = append(errs, err)
			} else {
				var width int
				for _, index := range indexes {
					if width < len(index.Name) {
						width = len(index.Name)
					}
				}
				for _, index := range indexes {
					fmt.Fprintf(w, "Index in %s: %s\t Keys: %v\n",
						collectionName,
						index.Name,
						index.Key)
				}
			}
		}
		w.Flush()
		if len(errs) > 0 {
			return rserrors.ErrDatabase().AddDetailsErr(errs...)
		}

		mongo.logger.Infoln("updating db version")
		_, err := collection.UpdateAll(nil, bson.M{
			"$set": bson.M{"version": dbversion},
		})
		if err != nil {
			return err
		}
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
