package models

import (
	"fmt"
	"testing"

	rstypes "git.containerum.net/ch/json-types/resource-service"
)

func TestPermCheck(t *testing.T) {
	perms := []rstypes.PermissionStatus{rstypes.PermissionStatusOwner, rstypes.PermissionStatusRead, rstypes.PermissionStatusWrite, rstypes.PermissionStatusReadDelete}
	levels := []rstypes.PermissionStatus{rstypes.PermissionStatusOwner, rstypes.PermissionStatusRead, rstypes.PermissionStatusWrite, rstypes.PermissionStatusReadDelete}
	var permMatrix [][]bool
	for i := range perms {
		permMatrix = append(permMatrix, make([]bool, len(levels)))
		for j := range levels {
			permMatrix[i][j] = PermCheck(perms[i], levels[j])
		}
	}
	for i := range perms {
		for j := range levels {
			t.Logf("%-6s %s %s", fmt.Sprintf("%t", permMatrix[i][j]),
				perms[i], levels[j])
		}
	}
}
