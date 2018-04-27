package server

import (
	"context"

	"github.com/containerum/utils/httputil"
)

// IsAdminRole checks that request came from user with admin permissions.
func IsAdminRole(ctx context.Context) bool {
	if v, ok := ctx.Value(httputil.UserRoleContextKey).(string); ok {
		return v == "admin"
	}
	return false
}
