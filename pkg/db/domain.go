package db

import (
	"git.containerum.net/ch/resource-service/pkg/models/domain"
	"github.com/globalsign/mgo/bson"
	"github.com/satori/go.uuid"
)

func (mongo *MongoStorage) GetDomain(domainName string, pages ...uint) (*domain.Domain, error) {
	mongo.logger.Debugf("getting domain")
	var collection = mongo.db.C(CollectionDomain)
	var colQuerier = bson.M{"domain": domainName}
	var result = domain.Domain{}
	var query = collection.Find(colQuerier)
	if err := query.One(&result); err != nil {
		mongo.logger.WithError(err).Errorf("unable to get domain")
		return nil, err
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
		return nil, err
	}
	return &result, nil
}

// GetDomainsList supports pagination
// GetDomainsList(2) will return 2 domains
// GGetDomainsList(2, 5) will return 2 domains with offset 5
func (mongo *MongoStorage) GetDomainsList(pages ...int) ([]domain.Domain, error) {
	mongo.logger.Debugf("getting domain list")
	var collection = mongo.db.C(CollectionDomain)
	var result []domain.Domain
	var err error
	var query = collection.Find(nil)
	switch len(pages) {
	case 0:
		err = query.All(&result)
	case 1:
		var max = pages[0]
		err = query.Limit(max).All(&result)
	case 2:
		var offset = pages[1]
		var max = pages[0]
		err = query.Skip(offset).Limit(max).All(&result)
	default:
		mongo.logger.Fatalf("[resource-service/pkg/db.GetDomainList] invalid pagination config: expected at most 2 args, got %d", len(pages))
	}
	if err != nil {
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
