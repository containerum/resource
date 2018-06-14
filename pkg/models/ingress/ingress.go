package ingress

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

// IngressResource --  model for ingress for resource-service db
//
// swagger:model
type IngressResource struct {
	model.Ingress
	ID          string `json:"_id" bson:"_id,omitempty"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespaceid"`
}

// IngressList -- ingresses list
//
// swagger:model
type IngressList []IngressResource

//  IngressesResponse -- ingresses response
//
// swagger:model
type IngressesResponse struct {
	Ingresses IngressList `json:"ingresses"`
}

func (ingr IngressResource) Copy() IngressResource {
	var cp = ingr
	cp.Rules = append(make([]model.Rule, 0, len(cp.Rules)), cp.Rules...)
	for i, rule := range cp.Rules {
		rule.Path = append(make([]model.Path, 0, len(rule.Path)), rule.Path...)
		cp.Rules[i] = rule
	}
	return cp
}

func (ingr IngressResource) Paths() []model.Path {
	var paths = make([]model.Path, 0, len(ingr.Rules))
	for _, rule := range ingr.Rules {
		for _, path := range rule.Path {
			paths = append(paths, path)
		}
	}
	return paths
}

func (ingr IngressResource) OneSelectQuery() interface{} {
	return bson.M{
		"namespaceid":  ingr.NamespaceID,
		"deleted":      false,
		"ingress.name": ingr.Name,
	}
}

func (ingr IngressResource) OneSelectDeletedQuery() interface{} {
	return bson.M{
		"namespaceid":  ingr.NamespaceID,
		"deleted":      true,
		"ingress.name": ingr.Name,
	}
}

func (ingr IngressResource) AllSelectQuery() interface{} {
	return bson.M{
		"namespaceid": ingr.NamespaceID,
		"deleted":     false,
	}
}

func (ingr IngressResource) AllSelectOwnerQuery() interface{} {
	return bson.M{
		"ingress.owner": ingr.Owner,
		"deleted":       false,
	}
}

func IngressFromKube(nsID, owner string, ingress model.Ingress) IngressResource {
	ingress.Owner = owner
	return IngressResource{
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
	return IngressResource{
		NamespaceID: namespaceID,
		Ingress: model.Ingress{
			Name: name,
		},
	}.OneSelectQuery()
}

func (ingr IngressResource) UpdateQuery() interface{} {
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
