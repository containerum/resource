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

		if strset.FromSlice(collections).In("configmap") {
			fmt.Println("Collection 'configmap' already exists")
			return nil
		}
		if err := db.C("configmap").Create(&mgo.CollectionInfo{
			ForceIdIndex: true,
		}); err != nil {
			return err
		}
		var collection = db.C("configmap")
		if err := collection.EnsureIndex(mgo.Index{
			Key: []string{"configmap.owner"},
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
			Name: "alive_configmap",
			Key:  []string{"configmap.name"},
			PartialFilter: bson.M{
				"deleted": false,
			},
			Unique: true,
		}); err != nil {
			return err
		}
		return nil
	}, func(db *mgo.Database) error {
		if err := db.C("configmap").DropCollection(); err != nil {
			return err
		}
		return nil
	})
}
