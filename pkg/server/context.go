package server

import (
	"context"

	"git.containerum.net/ch/utils"
)

// IsAdminRole checks that request came from user with admin permissions.
func IsAdminRole(ctx context.Context) bool {
	if v, ok := ctx.Value(utils.UserRoleContextKey).(string); ok {
		return v == "admin"
	}
	return false
}
