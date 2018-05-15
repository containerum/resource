package service

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

type Service struct {
	model.Service
	Owner       string `json:"owner"`
	ID          string `json:"_id"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespaceid"`
}

func ServiceFromKube(nsID, owner string, service model.Service) Service {
	return Service{
		Service:     service,
		Owner:       owner,
		NamespaceID: nsID,
		ID:          uuid.New().String(),
	}
}

func (serv Service) SelectQuery() interface{} {
	return bson.M{
		"namespaceid":  serv.NamespaceID,
		"service.name": serv.Name,
		"deleted":      false,
	}
}
