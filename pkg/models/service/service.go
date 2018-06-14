package service

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

// ServiceResource --  model for service for resource-service db
//
// swagger:model
type ServiceResource struct {
	model.Service
	ID          string      `json:"_id" bson:"_id,omitempty"`
	Deleted     bool        `json:"deleted"`
	NamespaceID string      `json:"namespaceid"`
	Type        ServiceType `json:"type" bson:"type"`
}

// ServiceList -- services list
//
// swagger:model
type ServiceList []ServiceResource

//  ServicesResponse -- ingresses response
//
// swagger:model
type ServicesResponse struct {
	Services ServiceList `json:"services"`
}

type ServiceType string

const (
	ServiceInternal ServiceType = "internal"
	ServiceExternal ServiceType = "external"
)

func ServiceFromKube(nsID, owner string, stype ServiceType, service model.Service) ServiceResource {
	service.Owner = owner
	return ServiceResource{
		Service:     service,
		NamespaceID: nsID,
		ID:          uuid.New().String(),
		Type:        stype,
	}
}

func (serv ServiceResource) Copy() ServiceResource {
	var cp = serv
	cp.IPs = append(make([]string, 0, len(cp.IPs)), cp.IPs...)
	cp.Ports = append(make([]model.ServicePort, 0, len(cp.Ports)), cp.Ports...)
	return cp
}

func (serv ServiceResource) OneSelectQuery() interface{} {
	return bson.M{
		"namespaceid":  serv.NamespaceID,
		"deleted":      false,
		"service.name": serv.Name,
	}
}

func (serv ServiceResource) OneSelectDeletedQuery() interface{} {
	return bson.M{
		"namespaceid":  serv.NamespaceID,
		"deleted":      true,
		"service.name": serv.Name,
	}
}

func (serv ServiceResource) AllSelectQuery() interface{} {
	return bson.M{
		"namespaceid": serv.NamespaceID,
		"deleted":     false,
	}
}

func (serv ServiceResource) AllSelectOwnerQuery() interface{} {
	return bson.M{
		"service.owner": serv.Owner,
		"deleted":       false,
	}
}

func (serv ServiceResource) UpdateQuery() interface{} {
	return bson.M{
		"$set": bson.M{
			"service": serv.Service,
		},
	}
}

func OneSelectQuery(namespaceID, name string) interface{} {
	return ServiceResource{
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

func (list ServiceList) Filter(pred func(ServiceResource) bool) ServiceList {
	var filtered = make(ServiceList, 0, list.Len())
	for _, serv := range list {
		if pred(serv.Copy()) {
			filtered = append(filtered, serv.Copy())
		}
	}
	return filtered
}
