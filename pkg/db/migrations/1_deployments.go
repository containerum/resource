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
		if strset.FromSlice(collections).In("deployment") {
			fmt.Println("Collection 'deployment' already exists")
			return nil
		}
		if err := db.C("deployment").Create(&mgo.CollectionInfo{
			ForceIdIndex: true,
		}); err != nil {
			return err
		}
		var collection = db.C("deployment")
		if err := collection.EnsureIndex(mgo.Index{
			Key: []string{"deployment.owner"},
		}); err != nil {
			return err
		}
		if err := collection.EnsureIndexKey("namespaceid"); err != nil {
			return err
		}
		if err := collection.EnsureIndexKey("deleted"); err != nil {
			return err
		}
		if err := collection.EnsureIndex(mgo.Index{
			Name: "alive_deployment",
			Key:  []string{"deployment.name", "namespaceid"},
			PartialFilter: bson.M{
				"deleted":           false,
				"deployment.active": true,
			},
			Unique: true,
		}); err != nil {
			return err
		}
		if err := collection.EnsureIndexKey("active"); err != nil {
			return err
		}
		if err := collection.EnsureIndex(mgo.Index{
			Name: "unique_version_deployment",
			Key:  []string{"deployment.name", "namespaceid", "deployment.version"},
			PartialFilter: bson.M{
				"deleted": false,
			},
			Unique: true,
		}); err != nil {
			return err
		}
		return nil
	}, func(db *mgo.Database) error {
		if err := db.C("deployment").DropCollection(); err != nil {
			return err
		}
		return nil
	})
}
