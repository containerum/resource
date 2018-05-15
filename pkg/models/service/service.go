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

func (serv Service) Copy() Service {
	var cp = serv
	cp.IPs = append(make([]string, 0, len(cp.IPs)), cp.IPs...)
	cp.Ports = append(make([]model.ServicePort, 0, len(cp.Ports)), cp.Ports...)
	return cp
}

func (serv Service) SelectQuery() interface{} {
	return bson.M{
		"namespaceid":  serv.NamespaceID,
		"service.name": serv.Name,
		"deleted":      false,
	}
}

func (serv Service) UpdateQuery() interface{} {
	return bson.M{
		"$set": bson.M{
			"service": serv.Service,
		},
	}
}

type ServiceList []Service

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
