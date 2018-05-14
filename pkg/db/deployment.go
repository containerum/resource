package db

import (
	"git.containerum.net/ch/resource-service/pkg/models/deployment"
)

func (mongo *mongoStorage) CreateDeployment(deployment deployment.Deployment) (deployment.Deployment, error) {
	var collection = mongo.db.C(CollectionDeployment)
	if err := collection.Insert(deployment); err != nil {
		return deployment, err
	}
	return deployment, nil
}
