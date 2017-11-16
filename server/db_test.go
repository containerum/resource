package server

import (
	"testing"
	"fmt"
)

func TestPermCheck(t *testing.T) {
	perms := []string{"owner", "read", "write", "readdelete"}
	var permMatrix [][]bool
	for i := range perms {
		for j := range perms {
			permMatrix[i][j] = permCheck(perms[i], perms[j])
		}
	}
	for i := range perms {
		for j := range perms {
			t.Logf("%-5s %s %s", fmt.Sprintf("%t", permMatrix[i][j]),
				perms[i], perms[j])
		}
	}
}
