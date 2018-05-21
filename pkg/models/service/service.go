package service

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

// Service --  model for service for resource-service db
//
// swagger:model
type Service struct {
	model.Service
	Owner       string `json:"owner"`
	ID          string `json:"_id" bson:"_id,omitempty"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespaceid"`
}

// ServiceList -- services list
//
// swagger:model
type ServiceList []Service

type ServiceType string

const (
	ServiceInternal ServiceType = "internal"
	ServiceExternal ServiceType = "external"
)

func ServiceFromKube(nsID, owner string, service model.Service) Service {
	return Service{
		Service:     service,
		Owner:       owner,
		NamespaceID: nsID,
		ID:          uuid.New().String(),
	}
}

func (serv Service) Copy() Service {
	var cp = serv
	cp.IPs = append(make([]string, 0, len(cp.IPs)), cp.IPs...)
	cp.Ports = append(make([]model.ServicePort, 0, len(cp.Ports)), cp.Ports...)
	return cp
}

func (serv Service) OneSelectQuery() interface{} {
	return bson.M{
		"namespaceid":  serv.NamespaceID,
		"deleted":      false,
		"service.name": serv.Name,
	}
}

func (serv Service) OneSelectDeletedQuery() interface{} {
	return bson.M{
		"namespaceid":  serv.NamespaceID,
		"deleted":      true,
		"service.name": serv.Name,
	}
}

func (serv Service) AllSelectQuery() interface{} {
	return bson.M{
		"namespaceid": serv.NamespaceID,
		"deleted":     false,
	}
}

func (serv Service) AllSelectOwnerQuery() interface{} {
	return bson.M{
		"owner":   serv.Owner,
		"deleted": false,
	}
}

func (serv Service) UpdateQuery() interface{} {
	return bson.M{
		"$set": bson.M{
			"service": serv.Service,
		},
	}
}

func OneSelectQuery(namespaceID, name string) interface{} {
	return Service{
		NamespaceID: namespaceID,
		Service: model.Service{
			Name: name,
		},
	}.OneSelectQuery()
}

func (list ServiceList) Len() int {
	return len(list)
}

func (list ServiceList) Names() []string {
	var names = make([]string, 0, len(list))
	for _, serv := range list {
		names = append(names, serv.Name)
	}
	return names
}

func (list ServiceList) Domains() []string {
	var domains = make([]string, 0, len(list))
	for _, serv := range list {
		if serv.Domain != "" {
			domains = append(domains, serv.Domain)
		}
	}
	return domains
}

func (list ServiceList) Copy() ServiceList {
	var cp = make(ServiceList, 0, list.Len())
	for _, serv := range list {
		cp = append(cp, serv.Copy())
	}
	return cp
}

func (list ServiceList) Filter(pred func(Service) bool) ServiceList {
	var filtered = make(ServiceList, 0, list.Len())
	for _, serv := range list {
		if pred(serv.Copy()) {
			filtered = append(filtered, serv.Copy())
		}
	}
	return filtered
}
