package db

import (
	"git.containerum.net/ch/resource-service/pkg/models/deployment"
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

// If ID is empty when use UUID4 to generate one
func (mongo *MongoStorage) CreateDeployment(deployment deployment.Deployment) (deployment.Deployment, error) {
	mongo.logger.Debugf("creating deployment")
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
	mongo.logger.Debugf("getting deployment by name")
	var collection = mongo.db.C(CollectionDeployment)
	var depl deployment.Deployment
	var err error
	if err = collection.Find(deployment.Deployment{
		NamespaceID: namespaceID,
		Deployment: model.Deployment{
			Name: deploymentName,
		},
	}.OneSelectQuery()).One(&depl); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get deployment by name")
	}
	return depl, err
}

func (mongo *MongoStorage) GetDeploymentByID(ID string) (deployment.Deployment, error) {
	mongo.logger.Debugf("getting deployment by ID")
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
	mongo.logger.Debugf("getting deployment list")
	var collection = mongo.db.C(CollectionDeployment)
	var depl deployment.DeploymentList
	var err error
	if err = collection.Find(bson.M{
		"namespaceid": namespaceID,
		"deleted":     false,
	}).All(&depl); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get deployment")
	}
	return depl, err
}

func (mongo *MongoStorage) UpdateDeployment(upd deployment.Deployment) error {
	mongo.logger.Debugf("updating deployment")
	var collection = mongo.db.C(CollectionDeployment)
	err := collection.Update(upd.OneSelectQuery(), upd.UpdateQuery())
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to update deployment")
	}
	return err
}

func (mongo *MongoStorage) DeleteDeployment(namespace, name string) error {
	mongo.logger.Debugf("deleting deployment")
	var collection = mongo.db.C(CollectionDeployment)
	err := collection.Update(deployment.Deployment{
		Deployment: model.Deployment{
			Name: name,
		},
		NamespaceID: namespace,
	}.OneSelectQuery(),
		bson.M{
			"$set": bson.M{"deleted": true},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete deployment")
	}
	return err
}

func (mongo *MongoStorage) CountDeployments(owner string) (int, error) {
	mongo.logger.Debugf("counting deployment")
	var collection = mongo.db.C(CollectionDeployment)
	if n, err := collection.Find(bson.M{"owner": owner}).Count(); err != nil {
		return 0, err
	} else {
		return n, nil
	}
}

func (mongo *MongoStorage) CountReplicas(owner string) (int, error) {
	mongo.logger.Debugf("counting deployment replicas")
	var collection = mongo.db.C(CollectionDeployment)
	var count struct {
		Count int `json:"count"`
	}
	if err := collection.Pipe([]bson.M{
		{
			"$match": bson.M{
				"owner": owner,
			},
		},
		{
			"$group": bson.M{
				"_id": "",
				"count": bson.M{
					"$sum": "$deployment.replicas"},
			},
		},
	}).One(&count); err != nil {
		return 0, err
	} else {
		return count.Count, nil
	}
}
