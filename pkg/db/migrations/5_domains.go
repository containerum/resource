package migrations

import (
	"fmt"

	"git.containerum.net/ch/resource-service/pkg/util/strset"
	"github.com/globalsign/mgo"
	"github.com/xakep666/mongo-migrate"
)

func init() {
	migrate.Register(func(db *mgo.Database) error {
		collections, err := db.CollectionNames()
		if err != nil {
			return err
		}
		if strset.FromSlice(collections).In("domain") {
			fmt.Println("Collection 'domain' already exists")
			return nil
		}
		if err := db.C("domain").Create(&mgo.CollectionInfo{
			ForceIdIndex: true,
		}); err != nil {
			return err
		}
		var collection = db.C("domain")
		if err := collection.EnsureIndexKey("domain"); err != nil {
			return err
		}
		if err := collection.EnsureIndexKey("domain_group"); err != nil {
			return err
		}
		if err := collection.EnsureIndexKey("domain_group", "domain"); err != nil {
			return err
		}
		return nil
	}, func(db *mgo.Database) error {
		if err := db.C("domain").DropCollection(); err != nil {
			return err
		}
		return nil
	})
}
