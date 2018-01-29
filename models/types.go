package models

import (
	"context"
	"io"

	"git.containerum.net/ch/json-types/errors"

	"git.containerum.net/ch/grpc-proto-files/auth"
	rstypes "git.containerum.net/ch/json-types/resource-service"
)

// DB is an interface to resource-service database
type DB interface {
	CreateNamespace(ctx context.Context, userID, label string, namespace *rstypes.Namespace) error
	GetUserNamespaces(ctx context.Context, userID string, filters *NamespaceFilterParams) ([]rstypes.NamespaceWithVolumes, error)
	GetAllNamespaces(ctx context.Context, page, perPage int, filters *NamespaceFilterParams) ([]rstypes.NamespaceWithVolumes, error)
	GetUserNamespaceByLabel(ctx context.Context, userID string, label string) (rstypes.NamespaceWithVolumes, error)
	GetNamespaceWithUserPermissions(ctx context.Context, userID, label string) (rstypes.NamespaceWithUserPermissions, error)
	DeleteUserNamespaceByLabel(ctx context.Context, userID, label string) error
	DeleteAllUserNamespaces(ctx context.Context, userID string) error
	RenameNamespace(ctx context.Context, userID, oldLabel, newLabel string) error
	ResizeNamespace(ctx context.Context, userID, label string, namespace *rstypes.Namespace) error

	CreateVolume(ctx context.Context, userID, label string, volume *rstypes.Volume) error
	GetUserVolumes(ctx context.Context, userID string, filters *VolumeFilterParams) ([]rstypes.VolumeWithPermission, error)
	GetAllVolumes(ctx context.Context, page, perPage int, filters *VolumeFilterParams) ([]rstypes.VolumeWithPermission, error)
	GetUserVolumeByLabel(ctx context.Context, userID, label string) (rstypes.VolumeWithPermission, error)
	GetVolumeWithUserPermissions(ctx context.Context, userID, label string) (rstypes.VolumeWithUserPermissions, error)
	DeleteUserVolumeByLabel(ctx context.Context, userID, label string) error
	DeleteAllUserVolumes(ctx context.Context, userID string, deletePersistent bool) error
	RenameVolume(ctx context.Context, userID, oldLabel, newLabel string) error
	ResizeVolume(ctx context.Context, userID, label string, volume *rstypes.Volume) error
	SetVolumeActiveByID(ctx context.Context, id string, active bool) error
	SetUserVolumeActive(ctx context.Context, userID, label string, active bool) error

	GetUserResourceAccesses(ctx context.Context, userID string) (*auth.ResourcesAccess, error)
	SetNamespaceAccess(ctx context.Context, userID, label string, access rstypes.PermissionStatus) error
	SetVolumeAccess(ctx context.Context, userID, label string, access rstypes.PermissionStatus) error

	UnlinkNamespaceVolumes(ctx context.Context, userID, namespaceLabel string) ([]rstypes.Volume, error)
	UnlinkAllNamespaceVolumes(ctx context.Context, userID string) ([]rstypes.Volume, error)

	// Perform operations inside transaction
	// Transaction commits if `f` returns nil error, rollbacks and forwards error otherwise
	// May return ErrTransactionBegin if transaction start failed,
	// ErrTransactionCommit if commit failed, ErrTransactionRollback if rollback failed
	Transactional(ctx context.Context, f func(ctx context.Context, tx DB) error) error

	io.Closer
}

// DBError describes error from database
type DBError struct {
	Err *errors.Error
}

func (e *DBError) Error() string {
	return e.Err.Error()
}

// Errors which may occur in transactional operations
var (
	ErrTransactionBegin    = &DBError{Err: errors.New("transaction begin error")}
	ErrTransactionRollback = &DBError{Err: errors.New("transaction rollback error")}
	ErrTransactionCommit   = &DBError{Err: errors.New("transaction commit error")}
)

// Generic resource errors
var (
	ErrLabeledResourceExists    = errors.New("resource with this label already exists")
	ErrLabeledResourceNotExists = errors.New("resource with this label not exists")

	ErrResourceExists    = errors.New("resource already exists")
	ErrResourceNotExists = errors.New("resource not exists")
)
