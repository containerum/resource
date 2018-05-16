package db

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
)

func (mongo *MongoStorage) GetUserResources(namespaceID string) (model.Resource, error) {
	var deployments = mongo.db.C(CollectionDeployment)
	var res model.Resource
	return res, deployments.Pipe([]bson.M{
		{"$match": bson.M{
			"namespaceid": namespaceID,
		}},
		{"$project": bson.M{
			"replicas": "$deployment.replicas",
			"cpu":      bson.M{"$sum": "$deployment.containers.cpu"},
			"memory":   bson.M{"$sum": "$deployment.containers.cpu"},
		}},
		{"$project": bson.M{
			"cpu":    bson.M{"$multiply": []string{"$cpu", "$replicas"}},
			"memory": bson.M{"$multiply": []string{"memory", "$replicas"}},
		}},
		{"$group": bson.M{
			"_id":    256,
			"cpu":    bson.M{"$sum": "cpu"},
			"memory": bson.M{"$sum": "memory"},
		}},
	}).One(&res)
}
