package db

import (
	"git.containerum.net/ch/resource-service/pkg/models/deployment"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"github.com/blang/semver"
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
			return depl, rserrors.ErrResourceNotExists().AddDetails(deploymentName)
		}
		return depl, PipErr{err}.ToMongerr().Extract()
	}
	return depl, err
}

func (mongo *MongoStorage) GetDeploymentVersion(namespaceID, deploymentName string, version semver.Version) (deployment.DeploymentResource, error) {
	mongo.logger.Debugf("getting deployment version by name")
	var collection = mongo.db.C(CollectionDeployment)
	var depl deployment.DeploymentResource
	var err error
	if err = collection.Find(bson.M{
		"namespaceid":        namespaceID,
		"deleted":            false,
		"deployment.name":    deploymentName,
		"deployment.version": version,
	}).One(&depl); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get deployment version by name")
		if err == mgo.ErrNotFound {
			return depl, rserrors.ErrResourceNotExists().AddDetailF("%v %v", deploymentName, version.String())
		}
		return depl, PipErr{err}.ToMongerr().Extract()
	}
	return depl, err
}

func (mongo *MongoStorage) GetDeploymentLatestVersion(namespaceID, deploymentName string) (deployment.DeploymentResource, error) {
	mongo.logger.Debugf("getting deployment latest version by name")
	var collection = mongo.db.C(CollectionDeployment)
	var depl deployment.DeploymentResource
	var err error
	if err = collection.Find(bson.M{
		"namespaceid":     namespaceID,
		"deleted":         false,
		"deployment.name": deploymentName,
	}).Sort("-deployment.version").One(&depl); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get deployment by name")
		if err == mgo.ErrNotFound {
			return depl, rserrors.ErrResourceNotExists().AddDetails(deploymentName)
		}
		return depl, PipErr{err}.ToMongerr().Extract()
	}
	return depl, err
}

func (mongo *MongoStorage) GetDeploymentVersionsList(namespaceID string, deploymentName string) (deployment.DeploymentList, error) {
	mongo.logger.Debugf("getting deployment versions list")
	var collection = mongo.db.C(CollectionDeployment)
	var depl deployment.DeploymentList
	var err error
	if err = collection.Find(bson.M{
		"namespaceid":     namespaceID,
		"deleted":         false,
		"deployment.name": deploymentName,
	}).Sort("-deployment.version").All(&depl); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get deployment %v", deploymentName)
	}
	return depl, PipErr{err}.ToMongerr().Extract()
}

func (mongo *MongoStorage) GetDeploymentList(namespaceID string) (deployment.DeploymentList, error) {
	mongo.logger.Debugf("getting deployment list")
	var collection = mongo.db.C(CollectionDeployment)
	var depl deployment.DeploymentList
	var err error
	if err = collection.Find(bson.M{
		"namespaceid":       namespaceID,
		"deleted":           false,
		"deployment.active": true,
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

func (mongo *MongoStorage) UpdateActiveDeployment(upd deployment.DeploymentResource) error {
	mongo.logger.Debugf("updating deployment")
	var collection = mongo.db.C(CollectionDeployment)
	err := collection.Update(upd.OneSelectQuery(), upd.UpdateQuery())
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to update deployment")
	}
	return PipErr{err}.ToMongerr().Extract()
}

func (mongo *MongoStorage) UpdateDeploymentVersion(namespace, name string, oldversion, newversion semver.Version) error {
	mongo.logger.Debugf("updating deployment version")
	var collection = mongo.db.C(CollectionDeployment)
	err := collection.Update(bson.M{
		"namespaceid":        namespace,
		"deleted":            false,
		"deployment.name":    name,
		"deployment.version": oldversion,
	}, bson.M{
		"$set": bson.M{"deployment.version": newversion},
	})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to update deployment version")
		if err == mgo.ErrNotFound {
			return rserrors.ErrResourceNotExists().AddDetailF("%v %v", name, oldversion.String())
		}
	}
	return PipErr{err}.ToMongerr().Extract()
}

func (mongo *MongoStorage) DeleteDeployment(namespace, name string) error {
	mongo.logger.Debugf("deleting deployment")
	var collection = mongo.db.C(CollectionDeployment)
	_, err := collection.UpdateAll(deployment.DeploymentResource{
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
			return rserrors.ErrResourceNotExists().AddDetails(name)
		}
		return PipErr{err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) ActivateDeployment(namespace, name string, version semver.Version) error {
	mongo.logger.Debugf("activating deployment")
	var collection = mongo.db.C(CollectionDeployment)
	err := collection.Update(deployment.DeploymentResource{
		Deployment: model.Deployment{
			Name:    name,
			Version: version,
		},
		NamespaceID: namespace,
	}.OneAnyVersionSelectQuery(),
		bson.M{
			"$set": bson.M{"deployment.active": true},
		})

	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to deactivate deployment")
		if err == mgo.ErrNotFound {
			return rserrors.ErrResourceNotExists().AddDetailF("%v %v", name, version.String())
		}
		return PipErr{err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) DeactivateDeployment(namespace, name string) error {
	mongo.logger.Debugf("deactivating deployment")
	var collection = mongo.db.C(CollectionDeployment)
	_, err := collection.UpdateAll(deployment.DeploymentResource{
		Deployment: model.Deployment{
			Name: name,
		},
		NamespaceID: namespace,
	}.OneSelectQuery(),
		bson.M{
			"$set": bson.M{"deployment.active": false},
		})

	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to deactivate deployment")
		if err == mgo.ErrNotFound {
			return rserrors.ErrResourceNotExists().AddDetails(name)
		}
		return PipErr{err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) DeleteDeploymentVersion(namespace, name string, version semver.Version) error {
	mongo.logger.Debugf("deleting deployment version")
	var collection = mongo.db.C(CollectionDeployment)
	err := collection.Update(deployment.DeploymentResource{
		Deployment: model.Deployment{
			Name:    name,
			Version: version,
			Active:  false,
		},
		NamespaceID: namespace,
	}.OneInactiveSelectQuery(),
		bson.M{
			"$set": bson.M{"deleted": true},
		})

	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete deployment version")
		if err == mgo.ErrNotFound {
			return rserrors.ErrResourceNotExists().AddDetailF("%v %v", name, version.String())
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
			return rserrors.ErrResourceNotExists().AddDetails(name)
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
		Deployment: model.Deployment{Owner: owner},
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
	if n, err := collection.Find(bson.M{
		"deployment.owner":  owner,
		"deployment.active": true,
		"deleted":           false,
	}).Count(); err != nil {
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
				"deployment.owner":  owner,
				"deleted":           false,
				"deployment.active": true,
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
