package configmap

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

// Resource --  model for ConfigMap for resource-service db
//
// swagger:model
type Resource struct {
	model.ConfigMap
	ID          string `json:"_id" bson:"_id,omitempty"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespaceid"`
}

// List -- ConfigMaps list
//
// swagger:model
type List []Resource

//  ConfigMapsResponse -- configmap response
//
// swagger:model
type ConfigMapsResponse struct {
	ConfigMaps List `json:"config_maps"`
}

func FromKube(nsID, owner string, ConfigMap model.ConfigMap) Resource {
	ConfigMap.Owner = owner
	return Resource{
		ConfigMap:   ConfigMap,
		NamespaceID: nsID,
		ID:          uuid.New().String(),
	}
}

func (cm Resource) Copy() Resource {
	var cp = cm
	return cp
}

func (cm Resource) OneSelectQuery() interface{} {
	return bson.M{
		"namespaceid":    cm.NamespaceID,
		"deleted":        false,
		"configmap.name": cm.Name,
	}
}

func (cm Resource) OneSelectDeletedQuery() interface{} {
	return bson.M{
		"namespaceid":    cm.NamespaceID,
		"deleted":        true,
		"configmap.name": cm.Name,
	}
}

func (cm Resource) AllSelectQuery() interface{} {
	return bson.M{
		"namespaceid": cm.NamespaceID,
		"deleted":     false,
	}
}

func (cm Resource) AllSelectOwnerQuery() interface{} {
	return bson.M{
		"ConfigMap.owner": cm.Owner,
		"deleted":         false,
	}
}

func (cm Resource) UpdateQuery() interface{} {
	return bson.M{
		"$set": bson.M{
			"configmap": cm.ConfigMap,
		},
	}
}

func OneSelectQuery(namespaceID, name string) interface{} {
	return Resource{
		NamespaceID: namespaceID,
		ConfigMap: model.ConfigMap{
			Name: name,
		},
	}.OneSelectQuery()
}

func (list List) Len() int {
	return len(list)
}

func (list List) Names() []string {
	var names = make([]string, 0, len(list))
	for _, cm := range list {
		names = append(names, cm.Name)
	}
	return names
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
