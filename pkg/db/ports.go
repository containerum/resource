package db

import (
	"math/rand"
	"time"

	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
)

var (
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func (mongo *MongoStorage) GetFreePort(domain string, protocol model.Protocol, minPort, maxPort int) (int, error) {
	var collection = mongo.db.C(CollectionService)
	for {
		var port = rnd.Intn(maxPort-minPort) + minPort
		// TODO: benchmark and optimize!
		n, err := collection.Find(bson.M{
			"service.domain":         domain,
			"service.ports.port":     port,
			"deleted":                false,
			"service.ports.protocol": protocol,
		}).Count()
		if err != nil {
			return -1, err
		}
		if n == 0 {
			return port, nil
		}
	}
}
