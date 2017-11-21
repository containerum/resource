package server

import (
	"testing"
	"fmt"
)

func TestPermCheck(t *testing.T) {
	perms := []string{"owner", "read", "write", "readdelete"}
	levels := []string{"owner", "read", "write", "delete"}
	var permMatrix [][]bool
	for i := range perms {
		permMatrix = append(permMatrix, make([]bool, len(levels)))
		for j := range levels {
			permMatrix[i][j] = permCheck(perms[i], levels[j])
		}
	}
	for i := range perms {
		for j := range levels {
			t.Logf("%-6s %s %s", fmt.Sprintf("%t", permMatrix[i][j]),
				perms[i], levels[j])
		}
	}
}
