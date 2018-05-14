package db

import (
	"fmt"
	"time"

	"git.containerum.net/ch/resource-service/pkg/util/strset"
	"github.com/globalsign/mgo"
	"github.com/go-siris/siris/core/errors"
	"github.com/sirupsen/logrus"
)

const (
	localURL = "localhost:27017"

	DBname               = "resources"
	CollectionDeployment = "deployment"
	CollectionService    = "service"
)

func CollectionsNames() []string {
	return []string{
		CollectionDeployment,
		CollectionService,
	}
}

type mongoStorage struct {
	logger logrus.FieldLogger
	config MongoConfig
	closed bool
	db     *mgo.Database
}

func (mongo *mongoStorage) Close() (err error) {
	defer func() {
		switch rec := recover().(type) {
		case nil:
		case error:
			err = rec
		case fmt.Stringer:
			err = errors.New(rec.String())
		default:
			err = fmt.Errorf("%v", rec)
		}
	}()
	if mongo.closed {
		return fmt.Errorf("mongo stoarage already closed")
	}
	mongo.db.Session.Close()
	mongo.db = nil
	mongo.closed = true
	return nil
}

func (mongo *mongoStorage) IsClosed() bool {
	return mongo.closed
}

func (mongo *mongoStorage) Init() error {
	dbCollections, err := mongo.db.CollectionNames()
	if err != nil {
		return err
	}
	for _, collection := range strset.FromSlice(CollectionsNames()).SubSlice(dbCollections).Items() {
		if err := mongo.db.C(collection).Create(&mgo.CollectionInfo{
			ForceIdIndex: false,
		}); err != nil {
			return err
		}
	}
	if err := mongo.CreateIndex("id"); err != nil {
		return err
	}
	if err := mongo.CreateIndex("name"); err != nil {
		return err
	}
	if err := mongo.CreateIndex("owner"); err != nil {
		return err
	}
	if err := mongo.CreateIndex("namespace_id"); err != nil {
		return err
	}
	return nil
}

func NewMongo(config MongoConfig) (*mongoStorage, error) {
	if config.Logger == nil {
		var logger = logrus.StandardLogger()
		if config.Debug {
			logger.SetLevel(logrus.DebugLevel)
		}
		config.Logger = logger
	}
	if config.AppName == "" {
		config.AppName = "resource-service"
	}
	config.Logger = config.Logger.WithField("app", config.AppName)
	if config.Debug {
		config.Logger.Debugf("running in debug mode")
	}
	config.Logger.Debugf("running mongo init")

	if config.Timeout <= 0 {
		config.Timeout = 120 * time.Second
	}
	config.Logger.Debugf("config timeout %v", config.Timeout)

	if len(config.Addrs) == 0 {
		config.Addrs = append(config.Addrs, localURL)
	}
	config.Logger.Debugf("addrs %v", config.Addrs)

	session, err := mgo.DialWithInfo(&config.DialInfo)
	if err != nil {
		config.Logger.WithError(err).Errorf("unable to connect to mongo")
		return nil, err
	}
	mgo.SetDebug(config.Debug)
	if config.Debug {

	}
	var db = session.DB(DBname)
	if config.Username != "" || config.Password != "" {
		if err := db.Login(config.Username, config.Password); err != nil {
			return nil, err
		}
	}
	var storage = &mongoStorage{
		logger: config.Logger,
		config: config,
		db:     db,
	}
	return storage, nil
}
