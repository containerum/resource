package ingress

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

// Resource --  model for ingress for resource-service db
//
// swagger:model
type Resource struct {
	model.Ingress
	ID          string `json:"_id" bson:"_id,omitempty"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespaceid"`
}

// List -- ingresses list
//
// swagger:model
type List []Resource

//  IngressesResponse -- ingresses response
//
// swagger:model
type IngressesResponse struct {
	Ingresses List `json:"ingresses"`
}

func (ingr Resource) Copy() Resource {
	var cp = ingr
	cp.Rules = append(make([]model.Rule, 0, len(cp.Rules)), cp.Rules...)
	for i, rule := range cp.Rules {
		rule.Path = append(make([]model.Path, 0, len(rule.Path)), rule.Path...)
		cp.Rules[i] = rule
	}
	return cp
}

func (ingr Resource) Paths() []model.Path {
	var paths = make([]model.Path, 0, len(ingr.Rules))
	for _, rule := range ingr.Rules {
		paths = append(paths, rule.Path...)
	}
	return paths
}

func (ingr Resource) OneSelectQuery() interface{} {
	return bson.M{
		"namespaceid":  ingr.NamespaceID,
		"deleted":      false,
		"ingress.name": ingr.Name,
	}
}

func (ingr Resource) OneSelectDeletedQuery() interface{} {
	return bson.M{
		"namespaceid":  ingr.NamespaceID,
		"deleted":      true,
		"ingress.name": ingr.Name,
	}
}

func (ingr Resource) AllSelectQuery() interface{} {
	return bson.M{
		"namespaceid": ingr.NamespaceID,
		"deleted":     false,
	}
}

func (ingr Resource) AllSelectOwnerQuery() interface{} {
	return bson.M{
		"ingress.owner": ingr.Owner,
		"deleted":       false,
	}
}

func FromKube(nsID, owner string, ingress model.Ingress) Resource {
	ingress.Owner = owner
	return Resource{
		Ingress:     ingress,
		NamespaceID: nsID,
		ID:          uuid.New().String(),
	}
}

func ListSelectQuery(namespaceID string) interface{} {
	return bson.M{
		"namespaceid": namespaceID,
		"deleted":     false,
	}
}

func OneSelectQuery(namespaceID, name string) interface{} {
	return Resource{
		NamespaceID: namespaceID,
		Ingress: model.Ingress{
			Name: name,
		},
	}.OneSelectQuery()
}

func (ingr Resource) UpdateQuery() interface{} {
	return bson.M{
		"$set": bson.M{
			"ingress": ingr.Ingress,
		},
	}
}

func DeleteQuery() interface{} {
	return bson.M{
		"delete": true,
	}
}
