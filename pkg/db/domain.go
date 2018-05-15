package db

import (
	"git.containerum.net/ch/resource-service/pkg/models/domain"
	"github.com/globalsign/mgo/bson"
	"github.com/satori/go.uuid"
)

func (mongo *MongoStorage) GetDomain(domainName string) (*domain.Domain, error) {
	var collection = mongo.db.C(CollectionDomain)
	colQuerier := bson.M{"domain": domainName}
	result := domain.Domain{}
	if err := collection.Find(colQuerier).One(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (mongo *MongoStorage) GetDomainsList() ([]domain.Domain, error) {
	var collection = mongo.db.C(CollectionDomain)
	var result []domain.Domain
	if err := collection.Find(nil).All(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (mongo *MongoStorage) CreateDomain(domain domain.Domain) (*domain.Domain, error) {
	domain.ID = uuid.NewV4().String()
	var collection = mongo.db.C(CollectionDomain)
	if err := collection.Insert(domain); err != nil {
		return nil, err
	}
	return &domain, nil
}

func (mongo *MongoStorage) UpdateDomain(domain domain.Domain) (*domain.Domain, error) {
	var collection = mongo.db.C(CollectionDomain)
	colQuerier := bson.M{"domain": domain.Domain}
	if err := collection.Update(colQuerier, domain); err != nil {
		return nil, err
	}
	return &domain, nil
}

func (mongo *MongoStorage) DeleteDomain(domainName string) error {
	var collection = mongo.db.C(CollectionDomain)
	colQuerier := bson.M{"domain": domainName}
	if err := collection.Remove(colQuerier); err != nil {
		return err
	}
	return nil
}
