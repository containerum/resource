package service

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

// ResourceService --  model for service for resource-service db
//
// swagger:model
type ResourceService struct {
	model.Service
	ID          string `json:"_id" bson:"_id,omitempty"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespaceid"`
	Type        Type   `json:"type" bson:"type"`
}

// ListService -- services list
//
// swagger:model
type ListService []ResourceService

//  ServicesResponse -- ingresses response
//
// swagger:model
type ServicesResponse struct {
	Services ListService `json:"services"`
}

type Type string

const (
	Internal Type = "internal"
	External Type = "external"
)

func FromKube(nsID, owner string, stype Type, service model.Service) ResourceService {
	if owner == "" {
		owner = "00000000-0000-0000-0000-000000000000"
	}
	service.Owner = owner
	return ResourceService{
		Service:     service,
		NamespaceID: nsID,
		ID:          uuid.New().String(),
		Type:        stype,
	}
}

func (serv ResourceService) Copy() ResourceService {
	var cp = serv
	cp.IPs = append(make([]string, 0, len(cp.IPs)), cp.IPs...)
	cp.Ports = append(make([]model.ServicePort, 0, len(cp.Ports)), cp.Ports...)
	return cp
}

func (serv ResourceService) OneSelectQuery() interface{} {
	return bson.M{
		"namespaceid":  serv.NamespaceID,
		"deleted":      false,
		"service.name": serv.Name,
	}
}

func (serv ResourceService) OneSelectDeletedQuery() interface{} {
	return bson.M{
		"namespaceid":  serv.NamespaceID,
		"deleted":      true,
		"service.name": serv.Name,
	}
}

func (serv ResourceService) AllSelectQuery() interface{} {
	return bson.M{
		"namespaceid": serv.NamespaceID,
		"deleted":     false,
	}
}

func (serv ResourceService) AllSelectOwnerQuery() interface{} {
	return bson.M{
		"service.owner": serv.Owner,
		"deleted":       false,
	}
}

func (serv ResourceService) UpdateQuery() interface{} {
	return bson.M{
		"$set": bson.M{
			"service": serv.Service,
		},
	}
}

func OneSelectQuery(namespaceID, name string) interface{} {
	return ResourceService{
		NamespaceID: namespaceID,
		Service: model.Service{
			Name: name,
		},
	}.OneSelectQuery()
}

func (list ListService) Len() int {
	return len(list)
}

func (list ListService) Names() []string {
	var names = make([]string, 0, len(list))
	for _, serv := range list {
		names = append(names, serv.Name)
	}
	return names
}

func (list ListService) Domains() []string {
	var domains = make([]string, 0, len(list))
	for _, serv := range list {
		if serv.Domain != "" {
			domains = append(domains, serv.Domain)
		}
	}
	return domains
}

func (list ListService) Copy() ListService {
	var cp = make(ListService, 0, list.Len())
	for _, serv := range list {
		cp = append(cp, serv.Copy())
	}
	return cp
}

func (list ListService) Filter(pred func(ResourceService) bool) ListService {
	var filtered = make(ListService, 0, list.Len())
	for _, serv := range list {
		if pred(serv.Copy()) {
			filtered = append(filtered, serv.Copy())
		}
	}
	return filtered
}
