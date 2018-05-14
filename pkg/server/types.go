package server

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/pkg/clients"
	"git.containerum.net/ch/resource-service/pkg/models"
	kubtypes "github.com/containerum/kube-client/pkg/model"
)

// ResourceServiceClients is a structure with all client interfaces needed for resource-service functioning
type ResourceServiceClients struct {
	DB   models.RelationalDB
	Kube clients.Kube
}

type ResourceServiceConstructors struct {
	NamespaceDB     models.NamespaceDBConstructor
	VolumeDB        models.VolumeDBConstructor
	StorageDB       models.StorageDBConstructor
	DeployDB        models.DeployDBConstructor
	IngressDB       models.IngressDBConstructor
	DomainDB        models.DomainDBConstructor
	AccessDB        models.AccessDBConstructor
	ServiceDB       models.ServiceDBConstructor
	ResourceCountDB models.ResourceCountDBConstructor
	EndpointsDB     models.GlusterEndpointsDBConstructor
}

type UpdateServiceRequest kubtypes.Service

type DeployActions interface {
	CreateDeployment(ctx context.Context, nsLabel string, deploy kubtypes.Deployment) error
	GetDeployments(ctx context.Context, nsLabel string) ([]kubtypes.Deployment, error)
	GetDeploymentByLabel(ctx context.Context, nsLabel, deplName string) (kubtypes.Deployment, error)
	DeleteDeployment(ctx context.Context, nsLabel, deplName string) error
	ReplaceDeployment(ctx context.Context, nsLabel string, deploy kubtypes.Deployment) error
	SetDeploymentReplicas(ctx context.Context, nsLabel, deplName string, req kubtypes.UpdateReplicas) error
	SetContainerImage(ctx context.Context, nsLabel, deplName string, req kubtypes.UpdateImage) error
}

type DomainActions interface {
	AddDomain(ctx context.Context, req rstypes.AddDomainRequest) error
	GetAllDomains(ctx context.Context, params rstypes.GetAllDomainsQueryParams) (rstypes.GetAllDomainsResponse, error)
	GetDomain(ctx context.Context, domain string) (rstypes.GetDomainResponse, error)
	DeleteDomain(ctx context.Context, domain string) error
}

type IngressActions interface {
	CreateIngress(ctx context.Context, nsLabel string, req kubtypes.Ingress) error
	GetUserIngresses(ctx context.Context, nsLabel string, params rstypes.GetIngressesQueryParams) (rstypes.GetIngressesResponse, error)
	GetAllIngresses(ctx context.Context, params rstypes.GetIngressesQueryParams) (rstypes.GetIngressesResponse, error)
	DeleteIngress(ctx context.Context, nsLabel, domain string) error
}

type ServiceActions interface {
	CreateService(ctx context.Context, nsLabel string, req kubtypes.Service) error
	GetServices(ctx context.Context, nsLabel string) ([]kubtypes.Service, error)
	GetService(ctx context.Context, nsLabel, serviceName string) (kubtypes.Service, error)
	UpdateService(ctx context.Context, nsLabel string, req UpdateServiceRequest) error
	DeleteService(ctx context.Context, nsLabel, serviceName string) error
}

type ResourceCountActions interface {
	GetResourcesCount(ctx context.Context) (rstypes.GetResourcesCountResponse, error)
}
