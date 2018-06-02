package deployment

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

// DeploymentResource -- model for deployments for resource-service db
//
// swagger:model
type DeploymentResource struct {
	model.Deployment
	ID          string `json:"_id,omitempty" bson:"_id,omitempty"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespaceid"`
}

// Deployment -- deployments list
//
// swagger:model
type DeploymentList []DeploymentResource

func (depl DeploymentResource) UpdateQuery() interface{} {
	return bson.M{
		"$set": bson.M{
			"deployment": depl.Deployment,
		},
	}
}

func (depl DeploymentResource) OneSelectQuery() interface{} {
	return bson.M{
		"namespaceid":       depl.NamespaceID,
		"deleted":           false,
		"deployment.active": true,
		"deployment.name":   depl.Name,
	}
}

func (depl DeploymentResource) OneInactiveVersionSelectQuery() interface{} {
	return bson.M{
		"namespaceid":        depl.NamespaceID,
		"deleted":            false,
		"deployment.name":    depl.Name,
		"deployment.version": depl.Version,
	}
}

func (depl DeploymentResource) OneSelectDeletedQuery() interface{} {
	return bson.M{
		"namespaceid":     depl.NamespaceID,
		"deleted":         true,
		"deployment.name": depl.Name,
	}
}

func (depl DeploymentResource) AllSelectQuery() interface{} {
	return bson.M{
		"namespaceid": depl.NamespaceID,
		"deleted":     false,
	}
}

func (depl DeploymentResource) AllSelectOwnerQuery() interface{} {
	return bson.M{
		"deployment.owner": depl.Owner,
		"deleted":          false,
	}
}

func DeploymentFromKube(nsID, owner string, deployment model.Deployment) DeploymentResource {
	deployment.Owner = owner
	return DeploymentResource{
		Deployment:  deployment,
		NamespaceID: nsID,
		ID:          uuid.New().String(),
	}
}

func (depl DeploymentResource) Copy() DeploymentResource {
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
	return DeploymentResource{
		NamespaceID: namespaceID,
		Deployment: model.Deployment{
			Name: name,
		},
	}.OneSelectQuery()
}

func (list DeploymentList) Copy() DeploymentList {
	var cp = make(DeploymentList, 0, list.Len())
	for _, depl := range list {
		cp = append(cp, depl.Copy())
	}
	return cp
}

func (list DeploymentList) Len() int {
	return len(list)
}

func (list DeploymentList) Names() []string {
	var names = make([]string, 0, len(list))
	for _, depl := range list {
		names = append(names, depl.Name)
	}
	return names
}

func (list DeploymentList) IDs() []string {
	var IDs = make([]string, 0, len(list))
	for _, depl := range list {
		IDs = append(IDs, depl.ID)
	}
	return IDs
}

func (list DeploymentList) Filter(pred func(deployment DeploymentResource) bool) DeploymentList {
	var filtered = make(DeploymentList, 0, list.Len())
	for _, depl := range list {
		if pred(depl.Copy()) {
			filtered = append(filtered, depl.Copy())
		}
	}
	return filtered
}

func copyContainer(container model.Container) model.Container {
	var cp = container
	for i, env := range cp.Env {
		cp.Env[i] = env
	}
	for i, command := range cp.Commands {
		cp.Commands[i] = command
	}
	for i, port := range cp.Ports {
		cp.Ports[i] = port
	}
	for i, volume := range cp.VolumeMounts {
		cp.VolumeMounts[i] = volume
	}
	for i, config := range cp.ConfigMaps {
		cp.ConfigMaps[i] = config
	}
	return cp
}
