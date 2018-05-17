package ingress

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

// Ingress --  model for ingress for resource-service db
//
// swagger:model
type Ingress struct {
	model.Ingress
	Owner       string `json:"owner"`
	ID          string `json:"_id" bson:"_id,omitempty"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespaceid"`
}

// IngressList -- ingresses list
//
// swagger:model
type IngressList []Ingress

func (ingr Ingress) Copy() Ingress {
	var cp = ingr
	cp.Rules = append(make([]model.Rule, 0, len(cp.Rules)), cp.Rules...)
	for i, rule := range cp.Rules {
		rule.Path = append(make([]model.Path, 0, len(rule.Path)), rule.Path...)
		cp.Rules[i] = rule
	}
	return cp
}

func (ingr Ingress) Paths() []model.Path {
	var paths = make([]model.Path, 0, len(ingr.Rules))
	for _, rule := range ingr.Rules {
		for _, path := range rule.Path {
			paths = append(paths, path)
		}
	}
	return paths
}

func (ingr Ingress) OneSelectQuery() interface{} {
	return bson.M{
		"namespaceid":  ingr.NamespaceID,
		"deleted":      false,
		"ingress.name": ingr.Name,
	}
}

func (ingr Ingress) AllSelectQuery() interface{} {
	return bson.M{
		"namespaceid": ingr.NamespaceID,
		"deleted":     false,
	}
}

func IngressFromKube(nsID, owner string, ingress model.Ingress) Ingress {
	return Ingress{
		Ingress:     ingress,
		Owner:       owner,
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
	return Ingress{
		NamespaceID: name,
		Ingress: model.Ingress{
			Name: name,
		},
	}.OneSelectQuery()
}

func (ingr Ingress) UpdateQuery() interface{} {
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
