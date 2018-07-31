package configmap

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

// ConfigMapResource --  model for ConfigMap for resource-service db
//
// swagger:model
type ConfigMapResource struct {
	model.ConfigMap
	ID          string `json:"_id" bson:"_id,omitempty"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespaceid"`
}

// ConfigMapList -- ConfigMaps list
//
// swagger:model
type ConfigMapList []ConfigMapResource

//  ConfigMapsResponse -- configmap response
//
// swagger:model
type ConfigMapsResponse struct {
	ConfigMaps ConfigMapList `json:"ConfigMaps"`
}

func ConfigMapFromKube(nsID, owner string, ConfigMap model.ConfigMap) ConfigMapResource {
	ConfigMap.Owner = owner
	return ConfigMapResource{
		ConfigMap:   ConfigMap,
		NamespaceID: nsID,
		ID:          uuid.New().String(),
	}
}

func (cm ConfigMapResource) Copy() ConfigMapResource {
	var cp = cm
	return cp
}

func (cm ConfigMapResource) OneSelectQuery() interface{} {
	return bson.M{
		"namespaceid":    cm.NamespaceID,
		"deleted":        false,
		"configmap.name": cm.Name,
	}
}

func (cm ConfigMapResource) OneSelectDeletedQuery() interface{} {
	return bson.M{
		"namespaceid":    cm.NamespaceID,
		"deleted":        true,
		"configmap.name": cm.Name,
	}
}

func (cm ConfigMapResource) AllSelectQuery() interface{} {
	return bson.M{
		"namespaceid": cm.NamespaceID,
		"deleted":     false,
	}
}

func (cm ConfigMapResource) AllSelectOwnerQuery() interface{} {
	return bson.M{
		"ConfigMap.owner": cm.Owner,
		"deleted":         false,
	}
}

func (cm ConfigMapResource) UpdateQuery() interface{} {
	return bson.M{
		"$set": bson.M{
			"configmap": cm.ConfigMap,
		},
	}
}

func OneSelectQuery(namespaceID, name string) interface{} {
	return ConfigMapResource{
		NamespaceID: namespaceID,
		ConfigMap: model.ConfigMap{
			Name: name,
		},
	}.OneSelectQuery()
}

func (list ConfigMapList) Len() int {
	return len(list)
}

func (list ConfigMapList) Names() []string {
	var names = make([]string, 0, len(list))
	for _, cm := range list {
		names = append(names, cm.Name)
	}
	return names
}

func (list ConfigMapList) Copy() ConfigMapList {
	var cp = make(ConfigMapList, 0, list.Len())
	for _, serv := range list {
		cp = append(cp, serv.Copy())
	}
	return cp
}

func (list ConfigMapList) Filter(pred func(ConfigMapResource) bool) ConfigMapList {
	var filtered = make(ConfigMapList, 0, list.Len())
	for _, serv := range list {
		if pred(serv.Copy()) {
			filtered = append(filtered, serv.Copy())
		}
	}
	return filtered
}
