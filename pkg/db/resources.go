package db

import (
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
)

func (mongo *MongoStorage) GetNamespaceResourcesLimits(namespaceID string) (kubtypes.Resource, error) {
	var deployments = mongo.db.C(CollectionDeployment)
	var res kubtypes.Resource
	var err = deployments.Pipe([]bson.M{
		{"$match": bson.M{
			"namespaceid": namespaceID,
			"deleted":     false,
		}},
		{"$project": bson.M{
			"replicas": "$deployment.replicas",
			"cpu":      bson.M{"$sum": "$deployment.containers.limits.cpu"},
			"memory":   bson.M{"$sum": "$deployment.containers.limits.memory"},
		}},
		{"$project": bson.M{
			"cpu":    bson.M{"$multiply": []string{"$cpu", "$replicas"}},
			"memory": bson.M{"$multiply": []string{"$memory", "$replicas"}},
		}},
		{"$group": bson.M{
			"_id":    256,
			"cpu":    bson.M{"$sum": "$cpu"},
			"memory": bson.M{"$sum": "$memory"},
		}},
	}).One(&res)
	return res, PipErr{err}.ToMongerr().NotFoundToNil()
}
