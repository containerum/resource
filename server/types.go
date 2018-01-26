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

type ResourceServiceClients struct {
	DB      models.DB
	Auth    clients.AuthSvc
	Kube    clients.Kube
	Mail    clients.Mailer
	Billing clients.Billing
}

type ResourceService interface {
	CreateNamespace(ctx context.Context, req *rstypes.CreateNamespaceRequest) (err error)
	GetUserNamespaces(ctx context.Context, params *rstypes.GetAllResourcesQueryParams) (rstypes.GetAllNamespacesResponse, error)

	io.Closer
}

var (
	ErrPermission = errors.NewWithCode("permission denied", http.StatusForbidden)
)
