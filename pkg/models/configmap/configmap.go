package configmap

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

// ResourceConfigMap --  model for ConfigMap for resource-service db
//
// swagger:model
type ResourceConfigMap struct {
	model.ConfigMap
	ID          string `json:"_id" bson:"_id,omitempty"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespaceid"`
}

// ListConfigMaps -- ConfigMaps list
//
// swagger:model
type ListConfigMaps []ResourceConfigMap

//  ConfigMapsResponse -- configmap response
//
// swagger:model
type ConfigMapsResponse struct {
	ConfigMaps ListConfigMaps `json:"config_maps"`
}

func FromKube(nsID, owner string, ConfigMap model.ConfigMap) ResourceConfigMap {
	if owner == "" {
		owner = "00000000-0000-0000-0000-000000000000"
	}

	ConfigMap.Owner = owner
	return ResourceConfigMap{
		ConfigMap:   ConfigMap,
		NamespaceID: nsID,
		ID:          uuid.New().String(),
	}
}

func (cm ResourceConfigMap) Copy() ResourceConfigMap {
	var cp = cm
	return cp
}

func (cm ResourceConfigMap) OneSelectQuery() interface{} {
	return bson.M{
		"namespaceid":    cm.NamespaceID,
		"deleted":        false,
		"configmap.name": cm.Name,
	}
}

func (cm ResourceConfigMap) OneSelectDeletedQuery() interface{} {
	return bson.M{
		"namespaceid":    cm.NamespaceID,
		"deleted":        true,
		"configmap.name": cm.Name,
	}
}

func (cm ResourceConfigMap) AllSelectQuery() interface{} {
	return bson.M{
		"namespaceid": cm.NamespaceID,
		"deleted":     false,
	}
}

func (cm ResourceConfigMap) AllSelectOwnerQuery() interface{} {
	return bson.M{
		"ConfigMap.owner": cm.Owner,
		"deleted":         false,
	}
}

func (cm ResourceConfigMap) UpdateQuery() interface{} {
	return bson.M{
		"$set": bson.M{
			"configmap": cm.ConfigMap,
		},
	}
}

func OneSelectQuery(namespaceID, name string) interface{} {
	return ResourceConfigMap{
		NamespaceID: namespaceID,
		ConfigMap: model.ConfigMap{
			Name: name,
		},
	}.OneSelectQuery()
}

func (list ListConfigMaps) Len() int {
	return len(list)
}

func (list ListConfigMaps) Names() []string {
	var names = make([]string, 0, len(list))
	for _, cm := range list {
		names = append(names, cm.Name)
	}
	return names
}

func (list ListConfigMaps) Copy() ListConfigMaps {
	var cp = make(ListConfigMaps, 0, list.Len())
	for _, serv := range list {
		cp = append(cp, serv.Copy())
	}
	return cp
}

func (list ListConfigMaps) Filter(pred func(ResourceConfigMap) bool) ListConfigMaps {
	var filtered = make(ListConfigMaps, 0, list.Len())
	for _, serv := range list {
		if pred(serv.Copy()) {
			filtered = append(filtered, serv.Copy())
		}
	}
	return filtered
}
