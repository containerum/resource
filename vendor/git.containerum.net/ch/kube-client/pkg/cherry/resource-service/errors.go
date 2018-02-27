package rserrors

import (
	"net/http"

	"git.containerum.net/ch/kube-client/pkg/cherry"
)

var buildErr = cherry.BuildErr(cherry.ResourceService)

var (
	ErrDatabase              = buildErr("Database error", http.StatusInternalServerError, 1)
	ErrResourceNotExists     = buildErr("Resource is not exists", http.StatusNotFound, 2)
	ErrResourceAlreadyExists = buildErr("Resource already exists", http.StatusConflict, 3)
	ErrPermissionDenied      = buildErr("Permission denied", http.StatusForbidden, 4)
	ErrTariffUnchanged       = buildErr("Tariff unchanged", http.StatusBadRequest, 5)
	ErrTariffNotFound        = buildErr("Tariff was not found", http.StatusNotFound, 6)
	ErrResourceNotOwned      = buildErr("Can`t set access for resource which not owned by user", http.StatusForbidden, 7)
	ErrDeleteOwnerAccess     = buildErr("Owner can`t delete has own access to resource", http.StatusConflict, 8)
	ErrAccessRecordNotExists = buildErr("Access record for user not exists", http.StatusNotFound, 9)
	ErrOther                 = buildErr("Other error", http.StatusInternalServerError, 10)
	ErrValidation            = buildErr("Validation error", http.StatusBadRequest, 11)
)
