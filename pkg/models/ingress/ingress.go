package service

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

type Ingress struct {
	model.Ingress
	Owner       string `json:"owner"`
	ID          string `json:"_id"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespaceid"`
}

func IngressFromKube(nsID, owner string, ingress model.Ingress) Ingress {
	return Ingress{
		Ingress:     ingress,
		Owner:       owner,
		NamespaceID: nsID,
		ID:          uuid.New().String(),
	}
}

func (ingress Ingress) Copy() Ingress {
	var cp = ingress
	cp.Rules = append(make([]model.Rule, 0, len(cp.Rules)), cp.Rules...)
	return cp
}

func (ingress Ingress) SelectQuery() interface{} {
	return bson.M{
		"namespaceid":  ingress.NamespaceID,
		"service.name": ingress.Name,
		"deleted":      false,
	}
}

func (serv Ingress) UpdateQuery() interface{} {
	return bson.M{
		"$set": bson.M{
			"ingress": serv.Ingress,
		},
	}
}

type IngressList []Ingress

func (list IngressList) Len() int {
	return len(list)
}

func (list IngressList) Names() []string {
	var names = make([]string, 0, len(list))
	for _, serv := range list {
		names = append(names, serv.Name)
	}
	return names
}

func (list IngressList) Copy() IngressList {
	var cp = make(IngressList, 0, list.Len())
	for _, serv := range list {
		cp = append(cp, serv.Copy())
	}
	return cp
}

func (list IngressList) Filter(pred func(Ingress) bool) IngressList {
	var filtered = make(IngressList, 0, list.Len())
	for _, serv := range list {
		if pred(serv.Copy()) {
			filtered = append(filtered, serv.Copy())
		}
	}
	return filtered
}
