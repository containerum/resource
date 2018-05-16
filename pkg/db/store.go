package db

import (
	"io"

	"git.containerum.net/ch/resource-service/pkg/models/deployment"
	"github.com/containerum/kube-client/pkg/model"
)

type Store interface {
	io.Closer
	Init() error
}

type DeploymentStorage interface {
	Create(deployment deployment.Deployment) (model.Deployment, error)
	Update(deployment deployment.Deployment) (model.Deployment, error)
	Delete(ID string) error
}

type ServiceStorage interface {
	Create(service model.Service) (model.Service, error)
	Update(service model.Service) (model.Service, error)
	Delete(ID string) error
}

type IngressStorage interface {
	Create(ingress model.Ingress) (model.Ingress, error)
	Update(ingress model.Ingress) (model.Ingress, error)
	Delete(ID string) error
}
