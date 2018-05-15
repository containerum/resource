package db

import (
	"git.containerum.net/ch/resource-service/pkg/models/domain"
	"github.com/globalsign/mgo/bson"
	"github.com/satori/go.uuid"
)

func (mongo *MongoStorage) GetDomain(domainName string) (*domain.Domain, error) {
	mongo.logger.Debugf("getting domain")
	var collection = mongo.db.C(CollectionDomain)
	colQuerier := bson.M{"domain": domainName}
	result := domain.Domain{}
	if err := collection.Find(colQuerier).One(&result); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get domain")
		return nil, err
	}
	return &result, nil
}

func (mongo *MongoStorage) GetRandomDomain(domainName string) (*domain.Domain, error) {
	mongo.logger.Debugf("getting random domain")
	var collection = mongo.db.C(CollectionDomain)
	colQuerier := []bson.M{{"$sample": bson.M{"size": 1}}}
	result := domain.Domain{}
	if err := collection.Pipe(colQuerier).One(&result); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get random domain")
		return nil, err
	}
	return &result, nil
}

func (mongo *MongoStorage) GetDomainsList() ([]domain.Domain, error) {
	mongo.logger.Debugf("getting domain list")
	var collection = mongo.db.C(CollectionDomain)
	var result []domain.Domain
	if err := collection.Find(nil).All(&result); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get domain list")
		return nil, err
	}
	return result, nil
}

func (mongo *MongoStorage) CreateDomain(domain domain.Domain) (*domain.Domain, error) {
	mongo.logger.Debugf("creating domain")
	domain.ID = uuid.NewV4().String()
	var collection = mongo.db.C(CollectionDomain)
	if err := collection.Insert(domain); err != nil {
		mongo.logger.WithError(err).Errorf("unable to create domain")
		return nil, err
	}
	return &domain, nil
}

func (mongo *MongoStorage) UpdateDomain(domain domain.Domain) (*domain.Domain, error) {
	mongo.logger.Debugf("updating domain")
	var collection = mongo.db.C(CollectionDomain)
	colQuerier := bson.M{"domain": domain.Domain}
	if err := collection.Update(colQuerier, domain); err != nil {
		mongo.logger.WithError(err).Errorf("unable to update domain")
		return nil, err
	}
	return &domain, nil
}

func (mongo *MongoStorage) DeleteDomain(domainName string) error {
	mongo.logger.Debugf("deleting domain")
	var collection = mongo.db.C(CollectionDomain)
	colQuerier := bson.M{"domain": domainName}
	if err := collection.Remove(colQuerier); err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete domain")
		return err
	}
	return nil
}
