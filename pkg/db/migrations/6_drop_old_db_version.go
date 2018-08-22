package migrations

import (
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
		if !strset.FromSlice(collections).In("db") {
			return nil
		}
		if err := db.C("db").DropCollection(); err != nil {
			return err
		}
		return nil
	}, func(db *mgo.Database) error {
		if err := db.C("db").Create(&mgo.CollectionInfo{
			ForceIdIndex: true,
		}); err != nil {
			return err
		}
		return nil
	})
}
