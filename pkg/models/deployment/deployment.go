package deployment

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/google/uuid"
)

type Deployment struct {
	model.Deployment
	Owner       string `json:"owner"`
	ID          string `json:"_id,omitempty"`
	Deleted     string `json:"deleted"`
	NamespaceID string `json:"namespace_id"`
}

func DeploymentFromKube(nsID, owner string, deployment model.Deployment) Deployment {
	return Deployment{
		Deployment:  deployment,
		Owner:       owner,
		NamespaceID: nsID,
		ID:          uuid.New().String(),
	}
}

type DeploymentList []Deployment

func (list DeploymentList) Copy() DeploymentList {
	return append(make(DeploymentList, 0, list.Len()), list...)
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

func (list DeploymentList) Filter(pred func(deployment Deployment) bool) DeploymentList {
	var filtered = make(DeploymentList, 0, list.Len())
	for _, depl := range list {
		if pred(depl) {
			filtered = append(filtered, depl)
		}
	}
	return filtered
}
