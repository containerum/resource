package middleware

import (
	"git.containerum.net/ch/resource-service/pkg/models/headers"
	"git.containerum.net/ch/resource-service/pkg/rserrors"
	"github.com/containerum/cherry/adaptors/gonic"
	"github.com/containerum/utils/httputil"
	"github.com/gin-gonic/gin"
)

type AccessLevel string

const (
	levelOwner      AccessLevel = "owner"
	levelWrite      AccessLevel = "write"
	levelReadDelete AccessLevel = "read-delete"
	levelRead       AccessLevel = "read"
)

var (
	readLevels = []AccessLevel{
		levelOwner,
		levelWrite,
		levelReadDelete,
		levelRead,
	}
)

var (
	writeLevels = []AccessLevel{
		levelOwner,
		levelWrite,
	}
)

func ReadAccess(c *gin.Context) {
	ns := c.Param("namespace")
	if c.GetHeader(httputil.UserRoleXHeader) == RoleUser {
		var userNsData *headers.UserHeaderData
		nsList := c.MustGet(UserNamespaces).(*UserHeaderDataMap)
		for _, n := range *nsList {
			if ns == n.ID {
				userNsData = &n
				break
			}
		}
		if userNsData != nil {
			if ok := containsAccess(userNsData.Access, readLevels...); ok {
				return
			}
			gonic.Gonic(rserrors.ErrAccessError(), c)
			return
		}
		gonic.Gonic(rserrors.ErrResourceNotExists(), c)
		return
	}
}

func WriteAccess(c *gin.Context) {
	ns := c.Param("namespace")
	if c.GetHeader(httputil.UserRoleXHeader) == RoleUser {
		var userNsData *headers.UserHeaderData
		nsList := c.MustGet(UserNamespaces).(*UserHeaderDataMap)
		for _, n := range *nsList {
			if ns == n.ID {
				userNsData = &n
				break
			}
		}
		if userNsData != nil {
			if ok := containsAccess(userNsData.Access, writeLevels...); ok {
				return
			}
			gonic.Gonic(rserrors.ErrAccessError(), c)
			return
		}
		gonic.Gonic(rserrors.ErrResourceNotExists(), c)
		return
	}
}

func containsAccess(access string, in ...AccessLevel) bool {
	contains := false
	userAccess := AccessLevel(access)
	for _, acc := range in {
		if acc == userAccess {
			return true
		}
	}
	return contains
}
