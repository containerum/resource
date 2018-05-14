package deployment

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/google/uuid"
)

type Deployment struct {
	model.Deployment
	Owner       string `json:"owner"`
	ID          string `json:"id,omitempty"`
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

type DeploymentList Deployment
