package migrations

import (
	"github.com/globalsign/mgo"
	"github.com/xakep666/mongo-migrate"
)

func init() {
	migrate.Register(func(db *mgo.Database) error {
		var collection = db.C("domain")
		if err := collection.DropAllIndexes(); err != nil {
			return err
		}
		if err := collection.EnsureIndex(mgo.Index{
			Key:    []string{"domain"},
			Unique: true,
		}); err != nil {
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
		var collection = db.C("domain")
		if err := collection.DropAllIndexes(); err != nil {
			return err
		}
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
	})
}
