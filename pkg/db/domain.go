package db

import (
	"git.containerum.net/ch/resource-service/pkg/models/domain"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

func (mongo *MongoStorage) GetDomain(domainName string, pages ...uint) (*domain.Domain, error) {
	mongo.logger.Debugf("getting domain")
	var collection = mongo.db.C(CollectionDomain)
	var colQuerier = bson.M{"domain": domainName}
	var result = domain.Domain{}
	var query = collection.Find(colQuerier)
	if err := query.One(&result); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get domain")
		return nil, PipErr{err}.ToMongerr().Extract()
	}
	return &result, nil
}

func (mongo *MongoStorage) GetRandomDomain() (*domain.Domain, error) {
	mongo.logger.Debugf("getting random domain")
	var collection = mongo.db.C(CollectionDomain)
	colQuerier := []bson.M{{"$sample": bson.M{"size": 1}}}
	result := domain.Domain{}
	if err := collection.Pipe(colQuerier).One(&result); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get random domain")
		return nil, PipErr{err}.ToMongerr().Extract()
	}
	return &result, nil
}

// GetDomainsList supports pagination
func (mongo *MongoStorage) GetDomainsList(pages *PageInfo) ([]domain.Domain, error) {
	mongo.logger.Debugf("getting domain list")
	var collection = mongo.db.C(CollectionDomain)
	var result []domain.Domain
	if err := Paginate(collection.Find(nil), pages).All(&result); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get domain list")
		return nil, PipErr{err}.ToMongerr().NotFoundToNil().Extract()
	}
	return result, nil
}

func (mongo *MongoStorage) CreateDomain(domain domain.Domain) (*domain.Domain, error) {
	mongo.logger.Debugf("creating domain")
	if domain.ID == "" {
		domain.ID = uuid.New().String()
	}
	var collection = mongo.db.C(CollectionDomain)
	if err := collection.Insert(domain); err != nil {
		mongo.logger.WithError(err).Errorf("unable to create domain")
		return nil, PipErr{err}.ToMongerr().Extract()
	}
	return &domain, nil
}

func (mongo *MongoStorage) UpdateDomain(domain domain.Domain) (*domain.Domain, error) {
	mongo.logger.Debugf("updating domain")
	var collection = mongo.db.C(CollectionDomain)
	colQuerier := bson.M{"domain": domain.Domain}
	if err := collection.Update(colQuerier, domain); err != nil {
		mongo.logger.WithError(err).Errorf("unable to update domain")
		return nil, PipErr{err}.ToMongerr().Extract()
	}
	return &domain, nil
}

func (mongo *MongoStorage) DeleteDomain(domainName string) error {
	mongo.logger.Debugf("deleting domain")
	var collection = mongo.db.C(CollectionDomain)
	colQuerier := bson.M{"domain": domainName}
	if err := collection.Remove(colQuerier); err != nil {
		mongo.logger.WithError(err).Errorf("unable to delete domain")
		return PipErr{err}.ToMongerr().Extract()
	}
	return nil
}
