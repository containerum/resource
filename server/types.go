package server

import (
	"net/http"

	"context"
	"io"

	"git.containerum.net/ch/grpc-proto-files/auth"
	"git.containerum.net/ch/json-types/errors"
	rstypes "git.containerum.net/ch/json-types/resource-service"
	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/resource-service/clients"
	"git.containerum.net/ch/resource-service/models"
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
	CreateNamespace(ctx context.Context, req *rstypes.CreateNamespaceRequest) (err error)
	GetUserNamespaces(ctx context.Context, filters string) (rstypes.GetAllNamespacesResponse, error)
	GetUserNamespace(ctx context.Context, label string) (rstypes.GetUserNamespaceResponse, error)
	GetAllNamespaces(ctx context.Context, params *rstypes.GetAllResourcesQueryParams) (rstypes.GetAllNamespacesResponse, error)
	GetUserNamespaceAccesses(ctx context.Context, label string) (rstypes.GetUserNamespaceAccessesResponse, error)
	DeleteUserNamespace(ctx context.Context, label string) error
	DeleteAllUserNamespaces(ctx context.Context) error
	RenameUserNamespace(ctx context.Context, oldLabel, newLabel string) error
	ResizeUserNamespace(ctx context.Context, label string, newTariffID string) error

	CreateVolume(ctx context.Context, req *rstypes.CreateVolumeRequest) error
	GetUserVolumes(ctx context.Context, filters string) (rstypes.GetUserVolumesResponse, error)
	GetUserVolume(ctx context.Context, label string) (rstypes.GetUserVolumeResponse, error)
	GetAllVolumes(ctx context.Context, params *rstypes.GetAllResourcesQueryParams) (rstypes.GetAllVolumesResponse, error)
	GetUserVolumeAccesses(ctx context.Context, label string) (rstypes.VolumeWithUserPermissions, error)
	GetVolumesLinkedWithUserNamespace(ctx context.Context, label string) (rstypes.GetUserVolumesResponse, error)
	DeleteUserVolume(ctx context.Context, label string) error
	DeleteAllUserVolumes(ctx context.Context) error
	RenameUserVolume(ctx context.Context, oldLabel, newLabel string) error
	ResizeUserVolume(ctx context.Context, label string, newTariffID string) error

	GetUserAccesses(ctx context.Context) (*auth.ResourcesAccess, error)
	SetUserNamespaceAccess(ctx context.Context, label string, req *rstypes.SetNamespaceAccessRequest) error
	SetUserVolumeAccess(ctx context.Context, label string, req *rstypes.SetVolumeAccessRequest) error

	GetDeployments(ctx context.Context, nsLabel string) ([]kubtypes.Deployment, error)

	SetUserAccesses(ctx context.Context, accessLevel rstypes.PermissionStatus) error

	io.Closer
}

// "Business-logic" errors
var (
	ErrPermission       = errors.NewWithCode("permission denied", http.StatusForbidden)
	ErrTariffIsSame     = errors.NewWithCode("provided tariff is current tariff", http.StatusConflict)
	ErrTariffInactive   = errors.NewWithCode("provided tariff is inactive", http.StatusForbidden)
	ErrTariffNotPublic  = errors.NewWithCode("provided tariff is not public", http.StatusForbidden)
	ErrResourceNotOwned = errors.NewWithCode("can`t set access for resource which not owned by user", http.StatusForbidden)
)
