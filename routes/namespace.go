package routes

import (
	"net/http"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/utils"
	"github.com/gin-gonic/gin"
)

func createNamespaceHandler(ctx *gin.Context) {
	var req rstypes.CreateNamespaceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}

	if err := srv.CreateNamespace(ctx.Request.Context(), &req); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func getUserNamespacesHandler(ctx *gin.Context) {
	resp, err := srv.GetUserNamespaces(ctx.Request.Context(), ctx.Query("filters"))
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	utils.MaskForNonAdmin(ctx, resp)

	ctx.JSON(http.StatusOK, resp)
}

func getUserNamespaceHandler(ctx *gin.Context) {
	resp, err := srv.GetUserNamespace(ctx.Request.Context(), ctx.Param("label"))
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	utils.MaskForNonAdmin(ctx, &resp)

	ctx.JSON(http.StatusOK, resp)
}

func getAllNamespacesHandler(ctx *gin.Context) {
	var params rstypes.GetAllResourcesQueryParams
	if err := ctx.ShouldBindQuery(&params); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}

	resp, err := srv.GetAllNamespaces(ctx, &params)
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func deleteUserNamespaceHandler(ctx *gin.Context) {
	if err := srv.DeleteUserNamespace(ctx.Request.Context(), ctx.Param("label")); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func deleteAllUserNamespacesHandler(ctx *gin.Context) {
	if err := srv.DeleteAllUserNamespaces(ctx.Request.Context()); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func renameUserNamespaceHandler(ctx *gin.Context) {
	var req rstypes.RenameNamespaceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}

	if err := srv.RenameUserNamespace(ctx.Request.Context(), ctx.Param("label"), req.NewLabel); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func resizeUserNamespaceHandler(ctx *gin.Context) {
	var req rstypes.ResizeNamespaceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(badRequest(err))
		return
	}

	if err := srv.ResizeUserNamespace(ctx.Request.Context(), ctx.Param("label"), req.NewTariffID); err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}
