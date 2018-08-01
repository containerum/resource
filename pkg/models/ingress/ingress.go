package ingress

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

// ResourceIngress --  model for ingress for resource-service db
//
// swagger:model
type ResourceIngress struct {
	model.Ingress
	ID          string `json:"_id" bson:"_id,omitempty"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespaceid"`
}

// ListIngress -- ingresses list
//
// swagger:model
type ListIngress []ResourceIngress

//  IngressesResponse -- ingresses response
//
// swagger:model
type IngressesResponse struct {
	Ingresses ListIngress `json:"ingresses"`
}

func (ingr ResourceIngress) Copy() ResourceIngress {
	var cp = ingr
	cp.Rules = append(make([]model.Rule, 0, len(cp.Rules)), cp.Rules...)
	for i, rule := range cp.Rules {
		rule.Path = append(make([]model.Path, 0, len(rule.Path)), rule.Path...)
		cp.Rules[i] = rule
	}
	return cp
}

func (ingr ResourceIngress) Paths() []model.Path {
	var paths = make([]model.Path, 0, len(ingr.Rules))
	for _, rule := range ingr.Rules {
		paths = append(paths, rule.Path...)
	}
	return paths
}

func (ingr ResourceIngress) OneSelectQuery() interface{} {
	return bson.M{
		"namespaceid":  ingr.NamespaceID,
		"deleted":      false,
		"ingress.name": ingr.Name,
	}
}

func (ingr ResourceIngress) OneSelectDeletedQuery() interface{} {
	return bson.M{
		"namespaceid":  ingr.NamespaceID,
		"deleted":      true,
		"ingress.name": ingr.Name,
	}
}

func (ingr ResourceIngress) AllSelectQuery() interface{} {
	return bson.M{
		"namespaceid": ingr.NamespaceID,
		"deleted":     false,
	}
}

func (ingr ResourceIngress) AllSelectOwnerQuery() interface{} {
	return bson.M{
		"ingress.owner": ingr.Owner,
		"deleted":       false,
	}
}

func FromKube(nsID, owner string, ingress model.Ingress) ResourceIngress {
	ingress.Owner = owner
	return ResourceIngress{
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
	return ResourceIngress{
		NamespaceID: namespaceID,
		Ingress: model.Ingress{
			Name: name,
		},
	}.OneSelectQuery()
}

func (ingr ResourceIngress) UpdateQuery() interface{} {
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
