package server

import (
	"context"

	"git.containerum.net/ch/resource-service/pkg/models/configmap"
	"git.containerum.net/ch/resource-service/pkg/models/deployment"
	"git.containerum.net/ch/resource-service/pkg/models/domain"
	"git.containerum.net/ch/resource-service/pkg/models/ingress"
	"git.containerum.net/ch/resource-service/pkg/models/resources"
	"git.containerum.net/ch/resource-service/pkg/models/service"
	kubtypes "github.com/containerum/kube-client/pkg/model"
)

type DeployActions interface {
	GetDeploymentsList(ctx context.Context, nsID string) (*deployment.DeploymentsResponse, error)
	GetDeployment(ctx context.Context, nsID, deplName string) (*deployment.ResourceDeploy, error)
	GetDeploymentVersionsList(ctx context.Context, nsID, deployName string) (*deployment.DeploymentsResponse, error)
	GetDeploymentVersion(ctx context.Context, nsID, deplName, version string) (*deployment.ResourceDeploy, error)
	DiffDeployments(ctx context.Context, nsID, deplName, version1, version2 string) (*kubtypes.DeploymentDiff, error)
	DiffDeploymentsPrevious(ctx context.Context, nsID, deplName, version string) (*kubtypes.DeploymentDiff, error)
	CreateDeployment(ctx context.Context, nsID string, deploy kubtypes.Deployment) (*deployment.ResourceDeploy, error)
	ImportDeployment(ctx context.Context, nsID string, deploy kubtypes.Deployment) error
	ChangeActiveDeployment(ctx context.Context, nsID, deplName, version string) (*deployment.ResourceDeploy, error)
	UpdateDeployment(ctx context.Context, nsID string, deploy kubtypes.Deployment) (*deployment.ResourceDeploy, error)
	SetDeploymentReplicas(ctx context.Context, nsID, deplName string, req kubtypes.UpdateReplicas) (*deployment.ResourceDeploy, error)
	SetDeploymentContainerImage(ctx context.Context, nsID, deplName string, req kubtypes.UpdateImage) (*deployment.ResourceDeploy, error)
	RenameDeploymentVersion(ctx context.Context, nsID, deplName, oldversion, newversion string) (*deployment.ResourceDeploy, error)
	DeleteDeployment(ctx context.Context, nsID, deplName string) error
	DeleteDeploymentVersion(ctx context.Context, nsID, deplName, version string) error
	DeleteAllDeployments(ctx context.Context, nsID string) error
	DeleteAllSolutionDeployments(ctx context.Context, nsID, solutionName string) error
}

type DomainActions interface {
	GetDomainsList(ctx context.Context, page, perPage string) (*domain.DomainsResponse, error)
	GetDomain(ctx context.Context, domain string) (*domain.Domain, error)
	AddDomain(ctx context.Context, req domain.Domain) (*domain.Domain, error)
	DeleteDomain(ctx context.Context, domain string) error
}

type IngressActions interface {
	GetIngressesList(ctx context.Context, nsID string) (*ingress.IngressesResponse, error)
	GetSelectedIngressesList(ctx context.Context, namespaces []string) (*ingress.IngressesResponse, error)
	GetIngress(ctx context.Context, nsID, ingressName string) (*ingress.ResourceIngress, error)
	CreateIngress(ctx context.Context, nsID string, ingr kubtypes.Ingress) (*ingress.ResourceIngress, error)
	ImportIngress(ctx context.Context, nsID string, ingr kubtypes.Ingress) error
	UpdateIngress(ctx context.Context, nsID string, ingr kubtypes.Ingress) (*ingress.ResourceIngress, error)
	DeleteIngress(ctx context.Context, nsID, ingressName string) error
	DeleteAllIngresses(ctx context.Context, nsID string) error
}

type ServiceActions interface {
	GetServicesList(ctx context.Context, nsID string) (*service.ServicesResponse, error)
	GetService(ctx context.Context, nsID, serviceName string) (*service.ResourceService, error)
	CreateService(ctx context.Context, nsID string, svc kubtypes.Service) (*service.ResourceService, error)
	ImportService(ctx context.Context, nsID string, svc kubtypes.Service) error
	UpdateService(ctx context.Context, nsID string, svc kubtypes.Service) (*service.ResourceService, error)
	DeleteService(ctx context.Context, nsID, serviceName string) error
	DeleteAllServices(ctx context.Context, nsID string) error
	DeleteAllSolutionServices(ctx context.Context, nsID, solutionName string) error
}

type ConfigMapActions interface {
	GetConfigMapsList(ctx context.Context, nsID string) (*configmap.ConfigMapsResponse, error)
	GetSelectedConfigMapsList(ctx context.Context, namespaces []string) (*configmap.ConfigMapsResponse, error)
	GetConfigMap(ctx context.Context, nsID, ingressName string) (*configmap.ResourceConfigMap, error)
	CreateConfigMap(ctx context.Context, nsID string, cm kubtypes.ConfigMap) (*configmap.ResourceConfigMap, error)
	ImportConfigMap(ctx context.Context, nsID string, cm kubtypes.ConfigMap) error
	DeleteConfigMap(ctx context.Context, nsID, cmName string) error
	DeleteAllConfigMaps(ctx context.Context, nsID string) error
}

type ResourcesActions interface {
	GetResourcesCount(ctx context.Context) (*resources.GetResourcesCountResponse, error)
	GetAllResourcesCount(ctx context.Context) (*resources.GetResourcesCountResponse, error)
	DeleteAllResourcesInNamespace(ctx context.Context, nsID string) error
	DeleteAllUserResources(ctx context.Context) error
}
