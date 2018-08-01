package service

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

// Resource --  model for service for resource-service db
//
// swagger:model
type Resource struct {
	model.Service
	ID          string `json:"_id" bson:"_id,omitempty"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespaceid"`
	Type        Type   `json:"type" bson:"type"`
}

// List -- services list
//
// swagger:model
type List []Resource

//  ServicesResponse -- ingresses response
//
// swagger:model
type ServicesResponse struct {
	Services List `json:"services"`
}

type Type string

const (
	Internal Type = "internal"
	External Type = "external"
)

func FromKube(nsID, owner string, stype Type, service model.Service) Resource {
	service.Owner = owner
	return Resource{
		Service:     service,
		NamespaceID: nsID,
		ID:          uuid.New().String(),
		Type:        stype,
	}
}

func (serv Resource) Copy() Resource {
	var cp = serv
	cp.IPs = append(make([]string, 0, len(cp.IPs)), cp.IPs...)
	cp.Ports = append(make([]model.ServicePort, 0, len(cp.Ports)), cp.Ports...)
	return cp
}

func (serv Resource) OneSelectQuery() interface{} {
	return bson.M{
		"namespaceid":  serv.NamespaceID,
		"deleted":      false,
		"service.name": serv.Name,
	}
}

func (serv Resource) OneSelectDeletedQuery() interface{} {
	return bson.M{
		"namespaceid":  serv.NamespaceID,
		"deleted":      true,
		"service.name": serv.Name,
	}
}

func (serv Resource) AllSelectQuery() interface{} {
	return bson.M{
		"namespaceid": serv.NamespaceID,
		"deleted":     false,
	}
}

func (serv Resource) AllSelectOwnerQuery() interface{} {
	return bson.M{
		"service.owner": serv.Owner,
		"deleted":       false,
	}
}

func (serv Resource) UpdateQuery() interface{} {
	return bson.M{
		"$set": bson.M{
			"service": serv.Service,
		},
	}
}

func OneSelectQuery(namespaceID, name string) interface{} {
	return Resource{
		NamespaceID: namespaceID,
		Service: model.Service{
			Name: name,
		},
	}.OneSelectQuery()
}

func (list List) Len() int {
	return len(list)
}

func (list List) Names() []string {
	var names = make([]string, 0, len(list))
	for _, serv := range list {
		names = append(names, serv.Name)
	}
	return names
}

func (list List) Domains() []string {
	var domains = make([]string, 0, len(list))
	for _, serv := range list {
		if serv.Domain != "" {
			domains = append(domains, serv.Domain)
		}
	}
	return domains
}

func (list List) Copy() List {
	var cp = make(List, 0, list.Len())
	for _, serv := range list {
		cp = append(cp, serv.Copy())
	}
	return cp
}

func (list List) Filter(pred func(Resource) bool) List {
	var filtered = make(List, 0, list.Len())
	for _, serv := range list {
		if pred(serv.Copy()) {
			filtered = append(filtered, serv.Copy())
		}
	}
	return filtered
}
