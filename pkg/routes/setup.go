package routes

import (
	"net/http"

	umtypes "git.containerum.net/ch/json-types/user-manager"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
	"git.containerum.net/ch/resource-service/pkg/server"
	"git.containerum.net/ch/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/universal-translator"
	"gopkg.in/go-playground/validator.v9"
)

type TranslateValidate struct {
	*ut.UniversalTranslator
	*validator.Validate
}

func mainMiddlewareSetup(router gin.IRouter, tv *TranslateValidate) {
	router.Use(utils.SaveHeaders)
	router.Use(utils.PrepareContext)
	router.Use(utils.RequireHeaders(rserrors.ErrValidation, umtypes.UserIDHeader, umtypes.UserRoleHeader))
	router.Use(tv.ValidateHeaders(map[string]string{
		umtypes.UserIDHeader:   "uuid",
		umtypes.UserRoleHeader: "eq=admin|eq=user",
	}))
	router.Use(utils.SubstituteUserMiddleware(tv.Validate, tv.UniversalTranslator, rserrors.ErrValidation))
}

// SetupRoutes sets up a router
func SetupRoutes(router gin.IRouter, tv *TranslateValidate, backend server.ResourceService) {
	mainMiddlewareSetup(router, tv)

	nsHandlers := NamespaceHandlers{NamespaceActions: backend, TranslateValidate: tv}
	accessHandlers := AccessHandlers{AccessActions: backend, TranslateValidate: tv}
	deployHandlers := DeployHandlers{DeployActions: backend, TranslateValidate: tv}
	domainHandlers := DomainHandlers{DomainActions: backend, TranslateValidate: tv}
	ingressHandlers := IngressHandlers{IngressActions: backend, TranslateValidate: tv}
	serviceHandlers := ServiceHandlers{ServiceActions: backend, TranslateValidate: tv}
	storageHandlers := StorageHandlers{StorageActions: backend, TranslateValidate: tv}
	volumeHandlers := VolumeHandlers{VolumeActions: backend, TranslateValidate: tv}

	ns := router.Group("/namespace")
	{
		ns.POST("", nsHandlers.CreateNamespaceHandler)

		ns.GET("", nsHandlers.GetUserNamespacesHandler)
		ns.GET("/:ns_label", nsHandlers.GetUserNamespaceHandler)
		ns.GET("/:ns_label/access", accessHandlers.GetUserNamespaceAccessesHandler)
		ns.GET("/:ns_label/volumes", volumeHandlers.GetVolumesLinkedWithUserNamespaceHandler)

		ns.DELETE("/:ns_label", nsHandlers.DeleteUserNamespaceHandler)
		ns.DELETE("/:ns_label/access", accessHandlers.DeleteUserNamespaceAccessHandler)

		ns.PUT("/:ns_label/name", nsHandlers.RenameUserNamespaceHandler)
		ns.PUT("/:ns_label/access", accessHandlers.SetUserNamespaceAccessHandler)
		ns.PUT("/:ns_label", nsHandlers.ResizeUserNamespaceHandler)

		deployment := ns.Group("/:ns_label/deployment")
		{
			deployment.POST("", deployHandlers.CreateDeploymentHandler)

			deployment.GET("", deployHandlers.GetDeploymentsHandler)
			deployment.GET("/:deploy_label", deployHandlers.GetDeploymentByLabelHandler)

			deployment.DELETE("/:deploy_label", deployHandlers.DeleteDeploymentByLabelHandler)

			deployment.PUT("/:deploy_label/image", deployHandlers.SetContainerImageHandler)
			deployment.PUT("/:deploy_label", deployHandlers.ReplaceDeploymentHandler)
			deployment.PUT("/:deploy_label/replicas", deployHandlers.SetReplicasHandler)
		}

		ingress := ns.Group("/:ns_label/ingress")
		{
			ingress.POST("", ingressHandlers.CreateIngressHandler)

			ingress.GET("", ingressHandlers.GetUserIngressesHandler)

			ingress.DELETE("/:domain", ingressHandlers.DeleteIngressHandler)
		}

		service := ns.Group("/:ns_label/service")
		{
			service.POST("", serviceHandlers.CreateServiceHandler)

			service.GET("", serviceHandlers.GetServicesHandler)
			service.GET("/:service_label", serviceHandlers.GetServiceHandler)

			service.PUT("/:service_label", serviceHandlers.UpdateServiceHandler)

			service.DELETE("/:service_label", serviceHandlers.DeleteServiceHandler)
		}
	}

	nss := router.Group("/namespaces")
	{
		nss.GET("", utils.RequireAdminRole(rserrors.ErrPermissionDenied), nsHandlers.GetAllNamespacesHandler)

		nss.DELETE("", utils.RequireAdminRole(rserrors.ErrPermissionDenied), nsHandlers.DeleteAllUserNamespacesHandler)
	}

	vol := router.Group("/volume")
	{
		vol.POST("", volumeHandlers.CreateVolumeHandler)

		vol.GET("", volumeHandlers.GetUserVolumesHandler)
		vol.GET("/:vol_label", volumeHandlers.GetUserVolumeHandler)
		vol.GET("/:vol_label/access", accessHandlers.GetUserVolumeAccessesHandler)

		vol.DELETE("/:vol_label", volumeHandlers.DeleteUserVolumeHandler)
		vol.DELETE("/:vol_label/access", accessHandlers.DeleteUserVolumeAccessHandler)

		vol.PUT("/:vol_label/name", volumeHandlers.RenameUserVolumeHandler)
		vol.PUT("/:vol_label/access", accessHandlers.SetUserVolumeAccessHandler)
		vol.PUT("/:vol_label", volumeHandlers.ResizeUserVolumeHandler)
	}

	vols := router.Group("/volumes")
	{
		vols.GET("", utils.RequireAdminRole(rserrors.ErrPermissionDenied), volumeHandlers.GetAllVolumesHandler)

		vols.DELETE("", utils.RequireAdminRole(rserrors.ErrPermissionDenied), volumeHandlers.DeleteAllUserVolumesHandler)
	}

	router.GET("/access", accessHandlers.GetUserResourceAccessesHandler)

	router.GET("/ingresses", utils.RequireAdminRole(rserrors.ErrPermissionDenied), ingressHandlers.GetAllIngressesHandler)

	domain := router.Group("/domain", utils.RequireAdminRole(rserrors.ErrPermissionDenied))
	{
		domain.POST("", domainHandlers.AddDomainHandler)

		domain.GET("", domainHandlers.GetAllDomainsHandler)
		domain.GET("/:domain", domainHandlers.GetDomainHandler)

		domain.DELETE("/:domain", domainHandlers.DeleteDomainHandler)
	}

	storage := router.Group("/storage", utils.RequireAdminRole(rserrors.ErrPermissionDenied))
	{
		storage.POST("", storageHandlers.CreateStorageHandler)

		storage.GET("", storageHandlers.GetStoragesHandler)

		storage.PUT("/:storage_name", storageHandlers.UpdateStorageHandler)

		storage.DELETE("/:storage_name", storageHandlers.DeleteStorageHandler)
	}

	adm := router.Group("/adm", utils.RequireAdminRole(rserrors.ErrPermissionDenied))
	{
		adm.PUT("/access", accessHandlers.SetUserResourceAccessesHandler)
	}

	router.GET("/resources", func(ctx *gin.Context) {
		resp, err := backend.GetResourcesCount(ctx.Request.Context())
		if err != nil {
			ctx.AbortWithStatusJSON(tv.HandleError(err))
			return
		}

		ctx.JSON(http.StatusOK, resp)
	})
}
