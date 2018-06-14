package server

import (
	"context"

	"git.containerum.net/ch/resource-service/pkg/models/deployment"
	"git.containerum.net/ch/resource-service/pkg/models/domain"
	"git.containerum.net/ch/resource-service/pkg/models/ingress"
	"git.containerum.net/ch/resource-service/pkg/models/resources"
	"git.containerum.net/ch/resource-service/pkg/models/service"
	kubtypes "github.com/containerum/kube-client/pkg/model"
)

type DeployActions interface {
	GetDeploymentsList(ctx context.Context, nsID string) (*deployment.DeploymentsResponse, error)
	GetDeployment(ctx context.Context, nsID, deplName string) (*deployment.DeploymentResource, error)
	GetDeploymentVersionsList(ctx context.Context, nsID, deployName string) (*deployment.DeploymentsResponse, error)
	GetDeploymentVersion(ctx context.Context, nsID, deplName, version string) (*deployment.DeploymentResource, error)
	DiffDeployments(ctx context.Context, nsID, deplName, version1, version2 string) (*string, error)
	DiffDeploymentsPrevious(ctx context.Context, nsID, deplName, version string) (*string, error)
	CreateDeployment(ctx context.Context, nsID string, deploy kubtypes.Deployment) (*deployment.DeploymentResource, error)
	ChangeActiveDeployment(ctx context.Context, nsID, deplName, version string) (*deployment.DeploymentResource, error)
	UpdateDeployment(ctx context.Context, nsID string, deploy kubtypes.Deployment) (*deployment.DeploymentResource, error)
	SetDeploymentReplicas(ctx context.Context, nsID, deplName string, req kubtypes.UpdateReplicas) (*deployment.DeploymentResource, error)
	SetDeploymentContainerImage(ctx context.Context, nsID, deplName string, req kubtypes.UpdateImage) (*deployment.DeploymentResource, error)
	RenameDeploymentVersion(ctx context.Context, nsID, deplName, oldversion, newversion string) (*deployment.DeploymentResource, error)
	DeleteDeployment(ctx context.Context, nsID, deplName string) error
	DeleteDeploymentVersion(ctx context.Context, nsID, deplName, version string) error
	DeleteAllDeployments(ctx context.Context, nsID string) error
	DeleteAllSolutionDeployments(ctx context.Context, nsID, solutionName string) error
}

type DomainActions interface {
	GetDomainsList(ctx context.Context, page, per_page string) (*domain.DomainsResponse, error)
	GetDomain(ctx context.Context, domain string) (*domain.Domain, error)
	AddDomain(ctx context.Context, req domain.Domain) (*domain.Domain, error)
	DeleteDomain(ctx context.Context, domain string) error
}

type IngressActions interface {
	CreateIngress(ctx context.Context, nsID string, req kubtypes.Ingress) (*ingress.IngressResource, error)
	GetIngressesList(ctx context.Context, nsID string) (*ingress.IngressesResponse, error)
	GetIngress(ctx context.Context, nsID, ingressName string) (*ingress.IngressResource, error)
	UpdateIngress(ctx context.Context, nsID string, req kubtypes.Ingress) (*ingress.IngressResource, error)
	DeleteIngress(ctx context.Context, nsID, ingressName string) error
	DeleteAllIngresses(ctx context.Context, nsID string) error
}

type ServiceActions interface {
	CreateService(ctx context.Context, nsID string, req kubtypes.Service) (*service.ServiceResource, error)
	GetServicesList(ctx context.Context, nsID string) (*service.ServicesResponse, error)
	GetService(ctx context.Context, nsID, serviceName string) (*service.ServiceResource, error)
	UpdateService(ctx context.Context, nsID string, req kubtypes.Service) (*service.ServiceResource, error)
	DeleteService(ctx context.Context, nsID, serviceName string) error
	DeleteAllServices(ctx context.Context, nsID string) error
	DeleteAllSolutionServices(ctx context.Context, nsID, solutionName string) error
}

type ResourcesActions interface {
	GetResourcesCount(ctx context.Context) (*resources.GetResourcesCountResponse, error)
	DeleteAllResourcesInNamespace(ctx context.Context, nsID string) error
	DeleteAllUserResources(ctx context.Context) error
}
