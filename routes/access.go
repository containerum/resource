package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func getUserResourceAccessesHandler(ctx *gin.Context) {
	resp, err := srv.GetUserAccesses(ctx.Request.Context())
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}
