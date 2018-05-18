package db

import (
	"git.containerum.net/ch/resource-service/pkg/models/ingress"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

func (mongo *MongoStorage) CreateIngress(ingress ingress.Ingress) (ingress.Ingress, error) {
	mongo.logger.Debugf("creating ingress")
	var collection = mongo.db.C(CollectionIngress)
	if ingress.ID == "" {
		ingress.ID = uuid.New().String()
	}
	if err := collection.Insert(ingress); err != nil {
		mongo.logger.WithError(err).Errorf("unable to create ingress")
		if mgo.IsDup(err) {
			return ingress, rserrors.ErrResourceAlreadyExists().AddDetailsErr(err)
		}
		return ingress, PipErr{err}.ToMongerr().Extract()
	}
	return ingress, nil
}

func (mongo *MongoStorage) GetIngress(namespaceID, name string) (ingress.Ingress, error) {
	mongo.logger.Debugf("getting ingress")
	var collection = mongo.db.C(CollectionIngress)
	var ingr ingress.Ingress
	if err := collection.Find(ingress.OneSelectQuery(namespaceID, name)).One(&ingr); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get ingress")
		if err == mgo.ErrNotFound {
			return ingr, rserrors.ErrResourceNotExists()
		}
		return ingr, PipErr{err}.ToMongerr().Extract()
	}
	return ingr, nil
}

func (mongo *MongoStorage) GetIngressList(namespaceID string) (ingress.IngressList, error) {
	mongo.logger.Debugf("getting ingress")
	var collection = mongo.db.C(CollectionIngress)
	var list ingress.IngressList
	if err := collection.Find(ingress.ListSelectQuery(namespaceID)).All(&list); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get ingress list")
		return list, PipErr{err}.ToMongerr().NotFoundToNil().Extract()
	}
	return list, nil
}

func (mongo *MongoStorage) UpdateIngress(upd ingress.Ingress) (ingress.Ingress, error) {
	mongo.logger.Debugf("updating ingress")
	var collection = mongo.db.C(CollectionIngress)
	if err := collection.Update(upd.OneSelectQuery(), upd.UpdateQuery()); err != nil {
		mongo.logger.WithError(err).Errorf("unable to update ingress")
		return upd, PipErr{err}.ToMongerr().Extract()
	}
	return upd, nil
}

func (mongo *MongoStorage) DeleteIngress(namespaceID, name string) error {
	mongo.logger.Debugf("deleting ingress")
	var collection = mongo.db.C(CollectionIngress)
	err := collection.Update(ingress.Ingress{
		Ingress: model.Ingress{
			Name: name,
		},
		NamespaceID: namespaceID,
	}.OneSelectQuery(),
		bson.M{
			"$set": bson.M{"deleted": true},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete ingress")
		if err == mgo.ErrNotFound {
			return rserrors.ErrResourceNotExists()
		}
		return PipErr{err}.ToMongerr().Extract()
	}
	return nil
}

func (mongo *MongoStorage) DeleteAllIngresses(namespace string) error {
	mongo.logger.Debugf("deleting all ingresses in namespace")
	var collection = mongo.db.C(CollectionIngress)
	_, err := collection.UpdateAll(ingress.Ingress{
		NamespaceID: namespace,
	}.AllSelectQuery(),
		bson.M{
			"$set": bson.M{"deleted": true},
		})
	if err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete deployment")
	}
	return PipErr{err}.ToMongerr().Extract()
}

func (mongo *MongoStorage) CountIngresses(owner string) (int, error) {
	mongo.logger.Debugf("counting ingresses")
	var collection = mongo.db.C(CollectionIngress)
	if n, err := collection.Find(bson.M{
		"owner":   owner,
		"deleted": false,
	}).Count(); err != nil {
		return 0, PipErr{err}.ToMongerr().NotFoundToNil().Extract()
	} else {
		return n, nil
	}
}
