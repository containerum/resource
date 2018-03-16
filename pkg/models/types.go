package models

import (
	"context"
	"io"

	"git.containerum.net/ch/auth/proto"
	"git.containerum.net/ch/json-types/kube-api"
	rstypes "git.containerum.net/ch/json-types/resource-service"
	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
)

// DB is an interface to resource-service database
type DB interface {
	CreateNamespace(ctx context.Context, userID, label string, namespace *rstypes.Namespace) error
	GetUserNamespaces(ctx context.Context, userID string, filters *NamespaceFilterParams) ([]rstypes.NamespaceWithVolumes, error)
	GetAllNamespaces(ctx context.Context, page, perPage int, filters *NamespaceFilterParams) ([]rstypes.NamespaceWithVolumes, error)
	GetUserNamespaceByLabel(ctx context.Context, userID, label string) (rstypes.NamespaceWithPermission, error)
	GetUserNamespaceWithVolumesByLabel(ctx context.Context, userID string, label string) (rstypes.NamespaceWithVolumes, error)
	GetNamespaceWithUserPermissions(ctx context.Context, userID, label string) (rstypes.NamespaceWithUserPermissions, error)
	DeleteUserNamespaceByLabel(ctx context.Context, userID, label string) (rstypes.Namespace, error)
	DeleteAllUserNamespaces(ctx context.Context, userID string) error
	RenameNamespace(ctx context.Context, userID, oldLabel, newLabel string) error
	ResizeNamespace(ctx context.Context, namespace *rstypes.Namespace) error
	GetNamespaceID(ctx context.Context, userID, nsLabel string) (string, error)

	CreateVolume(ctx context.Context, userID, label string, volume *rstypes.Volume) error
	GetUserVolumes(ctx context.Context, userID string, filters *VolumeFilterParams) ([]rstypes.VolumeWithPermission, error)
	GetAllVolumes(ctx context.Context, page, perPage int, filters *VolumeFilterParams) ([]rstypes.VolumeWithPermission, error)
	GetUserVolumeByLabel(ctx context.Context, userID, label string) (rstypes.VolumeWithPermission, error)
	GetVolumeWithUserPermissions(ctx context.Context, userID, label string) (rstypes.VolumeWithUserPermissions, error)
	GetVolumesLinkedWithUserNamespace(ctx context.Context, userID, label string) ([]rstypes.VolumeWithPermission, error)
	DeleteUserVolumeByLabel(ctx context.Context, userID, label string) (rstypes.Volume, error)
	DeleteAllUserVolumes(ctx context.Context, userID string, nonPersistentOnly bool) ([]rstypes.Volume, error)
	RenameVolume(ctx context.Context, userID, oldLabel, newLabel string) error
	ResizeVolume(ctx context.Context, volume *rstypes.Volume) error
	SetVolumeActiveByID(ctx context.Context, id string, active bool) error
	SetUserVolumeActive(ctx context.Context, userID, label string, active bool) error

	GetUserResourceAccesses(ctx context.Context, userID string) (*authProto.ResourcesAccess, error)
	SetAllResourcesAccess(ctx context.Context, userID string, access rstypes.PermissionStatus) error
	SetResourceAccess(ctx context.Context, permRec *rstypes.PermissionRecord) error
	DeleteResourceAccess(ctx context.Context, resource rstypes.Resource, userID string) error

	CreateDeployment(ctx context.Context, userID, nsLabel string, deployment kubtypes.Deployment) (bool, error)
	GetDeployments(ctx context.Context, userID, nsLabel string) ([]kubtypes.Deployment, error)
	GetDeploymentByLabel(ctx context.Context, userID, nsLabel, deplLabel string) (kubtypes.Deployment, error)
	DeleteDeployment(ctx context.Context, userID, nsLabel, deplLabel string) (bool, error)
	ReplaceDeployment(ctx context.Context, userID, nsLabel string, deploy kubtypes.Deployment) error
	SetDeploymentReplicas(ctx context.Context, userID, nsLabel, deplLabel string, replicas int) error
	SetContainerImage(ctx context.Context, userID, nsLabel, deplLabel string, req kubtypes.UpdateImage) error

	AddDomain(ctx context.Context, req rstypes.AddDomainRequest) error
	GetAllDomains(ctx context.Context, params rstypes.GetAllDomainsQueryParams) ([]rstypes.Domain, error)
	GetDomain(ctx context.Context, domain string) (rstypes.Domain, error)
	DeleteDomain(ctx context.Context, domain string) error
	ChooseRandomDomain(ctx context.Context) (rstypes.Domain, error)

	CreateIngress(ctx context.Context, userID, nsLabel string, req rstypes.CreateIngressRequest) error
	GetUserIngresses(ctx context.Context, userID, nsLabel string, params rstypes.GetIngressesQueryParams) ([]rstypes.Ingress, error)
	GetAllIngresses(ctx context.Context, params rstypes.GetIngressesQueryParams) ([]rstypes.Ingress, error)
	DeleteIngress(ctx context.Context, userID, nsLabel, domain string) (rstypes.IngressType, error)

	CreateStorage(ctx context.Context, req rstypes.CreateStorageRequest) error
	GetStorages(ctx context.Context) ([]rstypes.Storage, error)
	UpdateStorage(ctx context.Context, name string, req rstypes.UpdateStorageRequest) error
	DeleteStorage(ctx context.Context, name string) error
	ChooseAvailableStorage(ctx context.Context, minFree int) (rstypes.Storage, error)
	ChooseDomainFreePort(ctx context.Context, domain string, protocol kubtypes.Protocol) (int, error)

	CreateGlusterEndpoints(ctx context.Context, userID, nsLabel string) ([]kube_api.Endpoint, error)
	ConfirmGlusterEndpoints(ctx context.Context, userID, nsLabel string) error

	CreateService(ctx context.Context, userID, nsLabel string, serviceType rstypes.ServiceType, req kubtypes.Service) error
	GetServices(ctx context.Context, userID, nsLabel string) ([]kubtypes.Service, error)
	GetService(ctx context.Context, userID, nsLabel, serviceName string) (kubtypes.Service, error)
	UpdateService(ctx context.Context, userID, nsLabel string, newServiceType rstypes.ServiceType, req kubtypes.Service) error
	DeleteService(ctx context.Context, userID, nsLabel, serviceName string) error

	GetResourcesCount(ctx context.Context, userID string) (rstypes.GetResourcesCountResponse, error)

	// Perform operations inside transaction
	// Transaction commits if `f` returns nil error, rollbacks and forwards error otherwise
	// May return ErrTransactionBegin if transaction start failed,
	// ErrTransactionCommit if commit failed, ErrTransactionRollback if rollback failed
	Transactional(ctx context.Context, f func(ctx context.Context, tx DB) error) error

	io.Closer
}
