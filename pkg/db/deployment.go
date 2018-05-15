package db

import (
	"git.containerum.net/ch/resource-service/pkg/models/deployment"
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

// If ID is empty when use UUID4 to generate one
func (mongo *MongoStorage) CreateDeployment(deployment deployment.Deployment) (deployment.Deployment, error) {
	var collection = mongo.db.C(CollectionDeployment)
	if deployment.ID == "" {
		deployment.ID = uuid.New().String()
	}
	if err := collection.Insert(deployment); err != nil {
		mongo.logger.WithError(err).Errorf("unable to create deployment")
		return deployment, err
	}
	return deployment, nil
}

func (mongo *MongoStorage) GetDeploymentByName(namespaceID, deploymentName string) (deployment.Deployment, error) {
	var collection = mongo.db.C(CollectionDeployment)
	var depl deployment.Deployment
	var err error
	if err = collection.Find(deployment.Deployment{
		NamespaceID: namespaceID,
		Deployment: model.Deployment{
			Name: deploymentName,
		},
	}.SelectByNameQuery()).One(&depl); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get deployment by name")
	}
	return depl, err
}

func (mongo *MongoStorage) GetDeploymentByID(ID string) (deployment.Deployment, error) {
	var collection = mongo.db.C(CollectionDeployment)
	var depl deployment.Deployment
	var err error
	if err = collection.FindId(ID).Select(bson.M{
		"deleted": false,
	}).One(&depl); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get deployment by id")
	}
	return depl, err
}

func (mongo *MongoStorage) GetDeploymentList(namespaceID string) (deployment.DeploymentList, error) {
	var collection = mongo.db.C(CollectionDeployment)
	var depl deployment.DeploymentList
	var err error
	if err = collection.Find(bson.M{
		"namespace_id": namespaceID,
		"deleted":      false,
	}).All(&depl); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get deployment")
	}
	return depl, err
}

func (mongo *MongoStorage) UpdateDeployment(upd deployment.Deployment) error {
	var collection = mongo.db.C(CollectionDeployment)
	err := collection.Update(
		upd.SelectByNameQuery(),
		bson.M{
			"$set": upd.UpdateQuery(),
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to update deployment")
	}
	return err
}

func (mongo *MongoStorage) DeleteDeployment(namespace, name string) error {
	var collection = mongo.db.C(CollectionDeployment)
	err := collection.Update(deployment.Deployment{
		Deployment: model.Deployment{
			Name: name,
		},
		NamespaceID: namespace,
	}.SelectByNameQuery(),
		bson.M{
			"$set": bson.M{"deleted": true},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete deployment")
	}
	return err
}
