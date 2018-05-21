package db

import (
	"git.containerum.net/ch/resource-service/pkg/models/deployment"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

func (mongo *MongoStorage) GetDeployment(namespaceID, deploymentName string) (deployment.DeploymentResource, error) {
	mongo.logger.Debugf("getting deployment by name")
	var collection = mongo.db.C(CollectionDeployment)
	var depl deployment.DeploymentResource
	var err error
	if err = collection.Find(deployment.OneSelectQuery(namespaceID, deploymentName)).One(&depl); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get deployment by name")
		if err == mgo.ErrNotFound {
			return depl, rserrors.ErrResourceNotExists()
		}
		return depl, PipErr{err}.ToMongerr().Extract()
	}
	return depl, err
}

//TODO Unused method
func (mongo *MongoStorage) GetDeploymentByID(ID string) (deployment.DeploymentResource, error) {
	mongo.logger.Debugf("getting deployment by ID")
	var collection = mongo.db.C(CollectionDeployment)
	var depl deployment.DeploymentResource
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
	return depl, PipErr{err}.ToMongerr().Extract()
}

// If ID is empty when use UUID4 to generate one
func (mongo *MongoStorage) CreateDeployment(deployment deployment.DeploymentResource) (deployment.DeploymentResource, error) {
	mongo.logger.Debugf("creating deployment")
	var collection = mongo.db.C(CollectionDeployment)
	if deployment.ID == "" {
		deployment.ID = uuid.New().String()
	}
	if err := collection.Insert(deployment); err != nil {
		mongo.logger.WithError(err).Errorf("unable to create deployment")
		if mgo.IsDup(err) {
			return deployment, rserrors.ErrResourceAlreadyExists().AddDetailsErr(err)
		}
		return deployment, err
	}
	return deployment, nil
}

func (mongo *MongoStorage) UpdateDeployment(upd deployment.DeploymentResource) error {
	mongo.logger.Debugf("updating deployment")
	var collection = mongo.db.C(CollectionDeployment)
	err := collection.Update(upd.OneSelectQuery(), upd.UpdateQuery())
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to update deployment")
	}
	return PipErr{err}.ToMongerr().Extract()
}

func (mongo *MongoStorage) DeleteDeployment(namespace, name string) error {
	mongo.logger.Debugf("deleting deployment")
	var collection = mongo.db.C(CollectionDeployment)
	err := collection.Update(deployment.DeploymentResource{
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
		if err == mgo.ErrNotFound {
			return rserrors.ErrResourceNotExists()
		}
		return PipErr{err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) RestoreDeployment(namespace, name string) error {
	mongo.logger.Debugf("restoring deployment")
	var collection = mongo.db.C(CollectionDeployment)
	err := collection.Update(deployment.DeploymentResource{
		Deployment: model.Deployment{
			Name: name,
		},
		NamespaceID: namespace,
	}.OneSelectDeletedQuery(),
		bson.M{
			"$set": bson.M{"deleted": false},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to restore deployment")
		if err == mgo.ErrNotFound {
			return rserrors.ErrResourceNotExists()
		}
		return PipErr{err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) DeleteAllDeploymentsInNamespace(namespace string) error {
	mongo.logger.Debugf("deleting all deployments in namespace")
	var collection = mongo.db.C(CollectionDeployment)
	_, err := collection.UpdateAll(deployment.DeploymentResource{
		NamespaceID: namespace,
	}.AllSelectQuery(),
		bson.M{
			"$set": bson.M{"deleted": true},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete deployments")
	}
	return PipErr{err}.ToMongerr().Extract()
}

func (mongo *MongoStorage) DeleteAllDeploymentsByOwner(owner string) error {
	mongo.logger.Debugf("deleting all user deployments")
	var collection = mongo.db.C(CollectionDeployment)
	_, err := collection.UpdateAll(deployment.DeploymentResource{
		Owner: owner,
	}.AllSelectOwnerQuery(),
		bson.M{
			"$set": bson.M{"deleted": true},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete deployments")
	}
	return PipErr{err}.ToMongerr().Extract()
}

func (mongo *MongoStorage) CountDeployments(owner string) (int, error) {
	mongo.logger.Debugf("counting deployment")
	var collection = mongo.db.C(CollectionDeployment)
	if n, err := collection.Find(bson.M{"owner": owner}).Count(); err != nil {
		return 0, PipErr{err}.ToMongerr().NotFoundToNil().Extract()
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
				"owner":   owner,
				"deleted": false,
			},
		},
		{
			"$group": bson.M{
				"_id": "",
				"count": bson.M{
					"$sum": "$deployment.replicas",
				},
			},
		},
	}).One(&count); err != nil {
		return 0, PipErr{err}.NotFoundToNil().ToMongerr().Extract()
	} else {
		return count.Count, nil
	}
}
