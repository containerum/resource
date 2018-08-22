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
		if strset.FromSlice(collections).In("ingress") {
			fmt.Println("Collection 'ingress' already exists")
			return nil
		}
		if err := db.C("ingress").Create(&mgo.CollectionInfo{
			ForceIdIndex: true,
		}); err != nil {
			return err
		}
		var collection = db.C("ingress")
		if err := collection.EnsureIndex(mgo.Index{
			Key: []string{"ingress.owner"},
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
			Name: "alive_ingress",
			Key:  []string{"ingress.name"},
			PartialFilter: bson.M{
				"deleted": false,
			},
			Unique: true,
		}); err != nil {
			return err
		}
		return nil
	}, func(db *mgo.Database) error {
		if err := db.C("ingress").DropCollection(); err != nil {
			return err
		}
		return nil
	})
}
