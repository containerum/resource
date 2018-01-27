package server

import (
	"net/http"

	"context"
	"io"

	"git.containerum.net/ch/json-types/errors"
	rstypes "git.containerum.net/ch/json-types/resource-service"
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
}

// ResourceService is an interface for resource-service operations.
type ResourceService interface {
	CreateNamespace(ctx context.Context, req *rstypes.CreateNamespaceRequest) (err error)
	GetUserNamespaces(ctx context.Context, params *rstypes.GetAllResourcesQueryParams) (rstypes.GetAllNamespacesResponse, error)
	GetUserNamespace(ctx context.Context, label string) (rstypes.GetUserNamespaceResponse, error)
	GetAllNamespaces(ctx context.Context, params *rstypes.GetAllResourcesQueryParams) (rstypes.GetAllNamespacesResponse, error)
	GetUserNamespaceAccesses(ctx context.Context, label string) (rstypes.GetUserNamespaceAccessesResponse, error)
	DeleteUserNamespace(ctx context.Context, label string) error
	DeleteAllUserNamespaces(ctx context.Context) error

	io.Closer
}

// "Business-logic" errors
var (
	ErrPermission = errors.NewWithCode("permission denied", http.StatusForbidden)
)
