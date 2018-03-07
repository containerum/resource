package server

import (
	"context"
	"io"

	"git.containerum.net/ch/grpc-proto-files/auth"
	rstypes "git.containerum.net/ch/json-types/resource-service"
	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/pkg/clients"
	"git.containerum.net/ch/resource-service/pkg/models"
)

// ResourceServiceClients is a structure with all client interfaces needed for resource-service functioning
type ResourceServiceClients struct {
	DB      models.DB
	Auth    clients.AuthSvc
	Kube    clients.Kube
	Mail    clients.Mailer
	Billing clients.Billing
	User    clients.UserManagerClient
}

// ResourceService is an interface for resource-service operations.
type ResourceService interface {
	CreateNamespace(ctx context.Context, req rstypes.CreateNamespaceRequest) (err error)
	GetUserNamespaces(ctx context.Context, filters string) (rstypes.GetAllNamespacesResponse, error)
	GetUserNamespace(ctx context.Context, label string) (rstypes.GetUserNamespaceResponse, error)
	GetAllNamespaces(ctx context.Context, params rstypes.GetAllResourcesQueryParams) (rstypes.GetAllNamespacesResponse, error)
	GetUserNamespaceAccesses(ctx context.Context, label string) (rstypes.GetUserNamespaceAccessesResponse, error)
	DeleteUserNamespace(ctx context.Context, label string) error
	DeleteAllUserNamespaces(ctx context.Context) error
	RenameUserNamespace(ctx context.Context, oldLabel, newLabel string) error
	ResizeUserNamespace(ctx context.Context, label string, newTariffID string) error

	CreateVolume(ctx context.Context, req rstypes.CreateVolumeRequest) error
	GetUserVolumes(ctx context.Context, filters string) (rstypes.GetUserVolumesResponse, error)
	GetUserVolume(ctx context.Context, label string) (rstypes.GetUserVolumeResponse, error)
	GetAllVolumes(ctx context.Context, params rstypes.GetAllResourcesQueryParams) (rstypes.GetAllVolumesResponse, error)
	GetUserVolumeAccesses(ctx context.Context, label string) (rstypes.VolumeWithUserPermissions, error)
	GetVolumesLinkedWithUserNamespace(ctx context.Context, label string) (rstypes.GetUserVolumesResponse, error)
	DeleteUserVolume(ctx context.Context, label string) error
	DeleteAllUserVolumes(ctx context.Context) error
	RenameUserVolume(ctx context.Context, oldLabel, newLabel string) error
	ResizeUserVolume(ctx context.Context, label string, newTariffID string) error

	GetUserAccesses(ctx context.Context) (*auth.ResourcesAccess, error)
	SetUserAccesses(ctx context.Context, accessLevel rstypes.PermissionStatus) error
	SetUserNamespaceAccess(ctx context.Context, label string, req *rstypes.SetNamespaceAccessRequest) error
	SetUserVolumeAccess(ctx context.Context, label string, req *rstypes.SetVolumeAccessRequest) error
	DeleteUserNamespaceAccess(ctx context.Context, nsLabel string, req rstypes.DeleteNamespaceAccessRequest) error
	DeleteUserVolumeAccess(ctx context.Context, volLabel string, req rstypes.DeleteVolumeAccessRequest) error

	CreateDeployment(ctx context.Context, nsLabel string, deploy kubtypes.Deployment) error
	GetDeployments(ctx context.Context, nsLabel string) ([]kubtypes.Deployment, error)
	GetDeploymentByLabel(ctx context.Context, nsLabel, deplLabel string) (kubtypes.Deployment, error)
	DeleteDeployment(ctx context.Context, nsLabel, deplLabel string) error
	ReplaceDeployment(ctx context.Context, nsLabel, deplLabel string, deploy kubtypes.Deployment) error
	SetDeploymentReplicas(ctx context.Context, nsLabel, deplLabel string, req rstypes.SetReplicasRequest) error
	SetContainerImage(ctx context.Context, nsLabel, deplLabel string, req rstypes.SetContainerImageRequest) error

	AddDomain(ctx context.Context, req rstypes.AddDomainRequest) error
	GetAllDomains(ctx context.Context, params rstypes.GetAllDomainsQueryParams) (rstypes.GetAllDomainsResponse, error)
	GetDomain(ctx context.Context, domain string) (rstypes.GetDomainResponse, error)
	DeleteDomain(ctx context.Context, domain string) error

	CreateIngress(ctx context.Context, nsLabel string, req rstypes.CreateIngressRequest) error
	GetUserIngresses(ctx context.Context, nsLabel string, params rstypes.GetIngressesQueryParams) (rstypes.GetIngressesResponse, error)
	GetAllIngresses(ctx context.Context, params rstypes.GetIngressesQueryParams) (rstypes.GetIngressesResponse, error)
	DeleteIngress(ctx context.Context, nsLabel, domain string) error

	CreateStorage(ctx context.Context, req rstypes.CreateStorageRequest) error
	GetStorages(ctx context.Context) ([]rstypes.Storage, error)
	UpdateStorage(ctx context.Context, name string, req rstypes.UpdateStorageRequest) error
	DeleteStorage(ctx context.Context, name string) error

	CreateService(ctx context.Context, nsLabel string, req kubtypes.Service) error
	GetServices(ctx context.Context, nsLabel string) ([]kubtypes.Service, error)
	GetService(ctx context.Context, nsLabel, serviceLabel string) (kubtypes.Service, error)
	UpdateService(ctx context.Context, nsLabel, serviceLabel string, req kubtypes.Service) error
	DeleteService(ctx context.Context, nsLabel, serviceLabel string) error

	GetResourcesCount(ctx context.Context) (rstypes.GetResourcesCountResponse, error)

	io.Closer
}
