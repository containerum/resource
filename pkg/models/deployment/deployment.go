package deployment

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

// ResourceDeploy -- model for deployments for resource-service db
//
// swagger:model
type ResourceDeploy struct {
	model.Deployment
	ID          string `json:"_id,omitempty" bson:"_id,omitempty"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespaceid"`
}

// Deployment -- deployments list
//
// swagger:model
type ListDeploy []ResourceDeploy

// DeploymentsResponse -- deployments response
//
// swagger:model
type DeploymentsResponse struct {
	Deployments ListDeploy `json:"deployments"`
}

func (depl ResourceDeploy) UpdateQuery() interface{} {
	return bson.M{
		"$set": bson.M{
			"deployment": depl.Deployment,
		},
	}
}

func (depl ResourceDeploy) OneSelectQuery() interface{} {
	return bson.M{
		"namespaceid":       depl.NamespaceID,
		"deleted":           false,
		"deployment.active": true,
		"deployment.name":   depl.Name,
	}
}

func (depl ResourceDeploy) OneInactiveSelectQuery() interface{} {
	return bson.M{
		"namespaceid":        depl.NamespaceID,
		"deleted":            false,
		"deployment.active":  false,
		"deployment.name":    depl.Name,
		"deployment.version": depl.Version,
	}
}

func (depl ResourceDeploy) OneAnyVersionSelectQuery() interface{} {
	return bson.M{
		"namespaceid":        depl.NamespaceID,
		"deleted":            false,
		"deployment.name":    depl.Name,
		"deployment.version": depl.Version,
	}
}

func (depl ResourceDeploy) OneSelectDeletedQuery() interface{} {
	return bson.M{
		"namespaceid":     depl.NamespaceID,
		"deleted":         true,
		"deployment.name": depl.Name,
	}
}

func (depl ResourceDeploy) AllSelectQuery() interface{} {
	return bson.M{
		"namespaceid": depl.NamespaceID,
		"deleted":     false,
	}
}

func (depl ResourceDeploy) AllSelectOwnerQuery() interface{} {
	return bson.M{
		"deployment.owner": depl.Owner,
		"deleted":          false,
	}
}

func FromKube(nsID, owner string, deployment model.Deployment) ResourceDeploy {
	if owner == "" {
		owner = "00000000-0000-0000-0000-000000000000"
	}
	deployment.Owner = owner
	return ResourceDeploy{
		Deployment:  deployment,
		NamespaceID: nsID,
		ID:          uuid.New().String(),
	}
}

func (depl ResourceDeploy) Copy() ResourceDeploy {
	var cp = depl
	if cp.Status != nil {
		var status = *cp.Status
		cp.Status = &status
	}
	for i, container := range depl.Containers {
		depl.Containers[i] = copyContainer(container)
	}
	return cp
}

func OneSelectQuery(namespaceID, name string) interface{} {
	return ResourceDeploy{
		NamespaceID: namespaceID,
		Deployment: model.Deployment{
			Name: name,
		},
	}.OneSelectQuery()
}

func (list ListDeploy) Copy() ListDeploy {
	var cp = make(ListDeploy, 0, list.Len())
	for _, depl := range list {
		cp = append(cp, depl.Copy())
	}
	return cp
}

func (list ListDeploy) Len() int {
	return len(list)
}

func (list ListDeploy) Names() []string {
	var names = make([]string, 0, len(list))
	for _, depl := range list {
		names = append(names, depl.Name)
	}
	return names
}

func (list ListDeploy) IDs() []string {
	var IDs = make([]string, 0, len(list))
	for _, depl := range list {
		IDs = append(IDs, depl.ID)
	}
	return IDs
}

func (list ListDeploy) Filter(pred func(deployment ResourceDeploy) bool) ListDeploy {
	var filtered = make(ListDeploy, 0, list.Len())
	for _, depl := range list {
		if pred(depl.Copy()) {
			filtered = append(filtered, depl.Copy())
		}
	}
	return filtered
}

func copyContainer(container model.Container) model.Container {
	var cp = container
	copy(cp.Env, cp.Env)
	copy(cp.Commands, cp.Commands)
	copy(cp.Ports, cp.Ports)
	copy(cp.VolumeMounts, cp.VolumeMounts)
	copy(cp.ConfigMaps, cp.ConfigMaps)
	return cp
}
