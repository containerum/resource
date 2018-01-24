package routes

import (
	"net/http"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func namespaceCreateHandler(ctx *gin.Context) {
	var req rstypes.CreateNamespaceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, errors.New(err.Error()))
	}
}
