package migrations

import (
	"fmt"

	"git.containerum.net/ch/resource-service/pkg/util/strset"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/xakep666/mongo-migrate"
)

func init() {
	migrate.Register(func(db *mgo.Database) error {
		collections, err := db.CollectionNames()
		if err != nil {
			return err
		}
		if strset.FromSlice(collections).In("service") {
			fmt.Println("Collection 'service' already exists")
			return nil
		}
		if err := db.C("service").Create(&mgo.CollectionInfo{
			ForceIdIndex: true,
		}); err != nil {
			return err
		}
		var collection = db.C("service")
		if err := collection.EnsureIndex(mgo.Index{
			Key: []string{"service.owner"},
		}); err != nil {
			return err
		}
		if err := collection.EnsureIndexKey("namespaceid"); err != nil {
			return err
		}
		if err := collection.EnsureIndexKey("deleted"); err != nil {
			return err
		}
		if err := collection.EnsureIndexKey("service_deployment"); err != nil {
			return err
		}
		if err := collection.EnsureIndexKey("service_domain"); err != nil {
			return err
		}
		if err := collection.EnsureIndex(mgo.Index{
			Name: "alive_service",
			Key:  []string{"service.name", "namespaceid"},
			PartialFilter: bson.M{
				"deleted": false,
			},
			Unique: true,
		}); err != nil {
			return err
		}
		if err := collection.EnsureIndex(mgo.Index{
			Name: "alive_service_with_ports",
			Key: []string{
				"service.domain",
				"service.ports.port",
				"service.ports.protocol",
			},
			PartialFilter: bson.M{
				"deleted": false,
				"type":    "external",
				"service.ports.port": bson.M{
					"$exists": true,
				},
			},
			Unique: true,
		}); err != nil {
			return err
		}
		return nil
	}, func(db *mgo.Database) error {
		if err := db.C("service").DropCollection(); err != nil {
			return err
		}
		return nil
	})
}
