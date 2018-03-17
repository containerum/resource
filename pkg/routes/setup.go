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

func MainMiddlewareSetup(router gin.IRouter, tv *TranslateValidate) {
	router.Use(utils.SaveHeaders)
	router.Use(utils.PrepareContext)
	router.Use(utils.RequireHeaders(rserrors.ErrValidation, umtypes.UserIDHeader, umtypes.UserRoleHeader))
	router.Use(tv.ValidateHeaders(map[string]string{
		umtypes.UserIDHeader:   "uuid",
		umtypes.UserRoleHeader: "eq=admin|eq=user",
	}))
	router.Use(utils.SubstituteUserMiddleware(tv.Validate, tv.UniversalTranslator, rserrors.ErrValidation))
}

func NamespaceHandlersSetup(router gin.IRouter, tv *TranslateValidate, backend server.NamespaceActions) {
	nsHandlers := NamespaceHandlers{NamespaceActions: backend, TranslateValidate: tv}

	ns := router.Group("/namespace")
	{
		ns.POST("", nsHandlers.CreateNamespaceHandler)

		ns.GET("", nsHandlers.GetUserNamespacesHandler)
		ns.GET("/:ns_label", nsHandlers.GetUserNamespaceHandler)

		ns.DELETE("/:ns_label", nsHandlers.DeleteUserNamespaceHandler)

		ns.PUT("/:ns_label/name", nsHandlers.RenameUserNamespaceHandler)
		ns.PUT("/:ns_label", nsHandlers.ResizeUserNamespaceHandler)
	}

	nss := router.Group("/namespaces", utils.RequireAdminRole(rserrors.ErrPermissionDenied))
	{
		nss.GET("", nsHandlers.GetAllNamespacesHandler)

		nss.DELETE("", nsHandlers.DeleteAllUserNamespacesHandler)
	}
}

func AccessHandlersSetup(router gin.IRouter, tv *TranslateValidate, backend server.AccessActions) {
	accessHandlers := AccessHandlers{AccessActions: backend, TranslateValidate: tv}

	ns := router.Group("/namespace")
	{
		ns.GET("/:ns_label/access", accessHandlers.GetUserNamespaceAccessesHandler)

		ns.DELETE("/:ns_label/access", accessHandlers.DeleteUserNamespaceAccessHandler)

		ns.PUT("/:ns_label/access", accessHandlers.SetUserNamespaceAccessHandler)
	}

	vol := router.Group("/volume")
	{
		vol.GET("/:vol_label/access", accessHandlers.GetUserVolumeAccessesHandler)

		vol.DELETE("/:vol_label/access", accessHandlers.DeleteUserVolumeAccessHandler)

		vol.PUT("/:vol_label/access", accessHandlers.SetUserVolumeAccessHandler)
	}

	adm := router.Group("/adm", utils.RequireAdminRole(rserrors.ErrPermissionDenied))
	{
		adm.PUT("/access", accessHandlers.SetUserResourceAccessesHandler)
	}

	router.GET("/access", accessHandlers.GetUserResourceAccessesHandler)
}

func DeployHandlersSetup(router gin.IRouter, tv *TranslateValidate, backend server.ResourceService) {
	deployHandlers := DeployHandlers{DeployActions: backend, TranslateValidate: tv}

	deployment := router.Group("/namespace/:ns_label/deployment")
	{
		deployment.POST("", deployHandlers.CreateDeploymentHandler)

		deployment.GET("", deployHandlers.GetDeploymentsHandler)
		deployment.GET("/:deploy_label", deployHandlers.GetDeploymentByLabelHandler)

		deployment.DELETE("/:deploy_label", deployHandlers.DeleteDeploymentByLabelHandler)

		deployment.PUT("/:deploy_label/image", deployHandlers.SetContainerImageHandler)
		deployment.PUT("/:deploy_label", deployHandlers.ReplaceDeploymentHandler)
		deployment.PUT("/:deploy_label/replicas", deployHandlers.SetReplicasHandler)
	}
}

func DomainHandlersSetup(router gin.IRouter, tv *TranslateValidate, backend server.DomainActions) {
	domainHandlers := DomainHandlers{DomainActions: backend, TranslateValidate: tv}

	domain := router.Group("/domain", utils.RequireAdminRole(rserrors.ErrPermissionDenied))
	{
		domain.POST("", domainHandlers.AddDomainHandler)

		domain.GET("", domainHandlers.GetAllDomainsHandler)
		domain.GET("/:domain", domainHandlers.GetDomainHandler)

		domain.DELETE("/:domain", domainHandlers.DeleteDomainHandler)
	}
}

func IngressHandlersSetup(router gin.IRouter, tv *TranslateValidate, backend server.ResourceService) {
	ingressHandlers := IngressHandlers{IngressActions: backend, TranslateValidate: tv}

	ingress := router.Group("/namespace/:ns_label/ingress")
	{
		ingress.POST("", ingressHandlers.CreateIngressHandler)

		ingress.GET("", ingressHandlers.GetUserIngressesHandler)

		ingress.DELETE("/:domain", ingressHandlers.DeleteIngressHandler)
	}

	router.GET("/ingresses", utils.RequireAdminRole(rserrors.ErrPermissionDenied), ingressHandlers.GetAllIngressesHandler)
}

func ServiceHandlersSetup(router gin.IRouter, tv *TranslateValidate, backend server.ResourceService) {
	serviceHandlers := ServiceHandlers{ServiceActions: backend, TranslateValidate: tv}

	service := router.Group("/namespace/:ns_label/service")
	{
		service.POST("", serviceHandlers.CreateServiceHandler)

		service.GET("", serviceHandlers.GetServicesHandler)
		service.GET("/:service_label", serviceHandlers.GetServiceHandler)

		service.PUT("/:service_label", serviceHandlers.UpdateServiceHandler)

		service.DELETE("/:service_label", serviceHandlers.DeleteServiceHandler)
	}
}

func StorageHandlersSetup(router gin.IRouter, tv *TranslateValidate, backend server.ResourceService) {
	storageHandlers := StorageHandlers{StorageActions: backend, TranslateValidate: tv}

	storage := router.Group("/storage", utils.RequireAdminRole(rserrors.ErrPermissionDenied))
	{
		storage.POST("", storageHandlers.CreateStorageHandler)

		storage.GET("", storageHandlers.GetStoragesHandler)

		storage.PUT("/:storage_name", storageHandlers.UpdateStorageHandler)

		storage.DELETE("/:storage_name", storageHandlers.DeleteStorageHandler)
	}
}

func VolumeHandlersSetup(router gin.IRouter, tv *TranslateValidate, backend server.ResourceService) {
	volumeHandlers := VolumeHandlers{VolumeActions: backend, TranslateValidate: tv}

	router.GET("/namespace/:ns_label/volumes", volumeHandlers.GetVolumesLinkedWithUserNamespaceHandler)

	vol := router.Group("/volume")
	{
		vol.POST("", volumeHandlers.CreateVolumeHandler)

		vol.GET("", volumeHandlers.GetUserVolumesHandler)
		vol.GET("/:vol_label", volumeHandlers.GetUserVolumeHandler)

		vol.DELETE("/:vol_label", volumeHandlers.DeleteUserVolumeHandler)

		vol.PUT("/:vol_label/name", volumeHandlers.RenameUserVolumeHandler)
		vol.PUT("/:vol_label", volumeHandlers.ResizeUserVolumeHandler)
	}

	vols := router.Group("/volumes")
	{
		vols.GET("", utils.RequireAdminRole(rserrors.ErrPermissionDenied), volumeHandlers.GetAllVolumesHandler)

		vols.DELETE("", utils.RequireAdminRole(rserrors.ErrPermissionDenied), volumeHandlers.DeleteAllUserVolumesHandler)
	}
}

// SetupRoutes sets up a router
func SetupRoutes(router gin.IRouter, tv *TranslateValidate, backend server.ResourceService) {
	MainMiddlewareSetup(router, tv)

	NamespaceHandlersSetup(router, tv, backend)
	AccessHandlersSetup(router, tv, backend)
	DeployHandlersSetup(router, tv, backend)
	DomainHandlersSetup(router, tv, backend)
	IngressHandlersSetup(router, tv, backend)
	ServiceHandlersSetup(router, tv, backend)
	StorageHandlersSetup(router, tv, backend)
	VolumeHandlersSetup(router, tv, backend)

	router.GET("/resources", func(ctx *gin.Context) {
		resp, err := backend.GetResourcesCount(ctx.Request.Context())
		if err != nil {
			ctx.AbortWithStatusJSON(tv.HandleError(err))
			return
		}

		ctx.JSON(http.StatusOK, resp)
	})
}
