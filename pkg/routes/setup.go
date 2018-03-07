package routes

import (
	"net/http"
	"sync"

	"reflect"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	umtypes "git.containerum.net/ch/json-types/user-manager"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
	"git.containerum.net/ch/resource-service/pkg/server"
	"git.containerum.net/ch/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gopkg.in/go-playground/validator.v8"
)

var srv server.ResourceService

type defaultValidator struct {
	once     sync.Once
	validate *validator.Validate
}

var _ binding.StructValidator = &defaultValidator{}

func (v *defaultValidator) ValidateStruct(obj interface{}) error {
	if kindOfData(obj) == reflect.Struct {
		v.lazyinit()
		if err := v.validate.Struct(obj); err != nil {
			return error(err)
		}
	}

	return nil
}

func (v *defaultValidator) lazyinit() {
	v.once.Do(func() {
		v.validate = validator.New(&validator.Config{TagName: "binding"})

		rstypes.RegisterCustomTags(v.validate)

		v.validate.RegisterStructValidation(createIngressRequestValidate, rstypes.CreateIngressRequest{})
	})
}

func kindOfData(data interface{}) reflect.Kind {

	value := reflect.ValueOf(data)
	valueType := value.Kind()

	if valueType == reflect.Ptr {
		valueType = value.Elem().Kind()
	}
	return valueType
}

// SetupRoutes sets up a router
func SetupRoutes(app *gin.Engine, server server.ResourceService) {
	srv = server

	binding.Validator = &defaultValidator{}

	app.Use(utils.SaveHeaders)
	app.Use(utils.PrepareContext)
	app.Use(utils.RequireHeaders(rserrors.ErrValidation(), umtypes.UserIDHeader, umtypes.UserRoleHeader))
	app.Use(utils.SubstituteUserMiddleware)

	ns := app.Group("/namespace")
	{
		ns.POST("", createNamespaceHandler)

		ns.GET("", getUserNamespacesHandler)
		ns.GET("/:ns_label", getUserNamespaceHandler)
		ns.GET("/:ns_label/access", getUserNamespaceAccessesHandler)
		ns.GET("/:ns_label/volumes", getVolumesLinkedWithUserNamespaceHandler)

		ns.DELETE("/:ns_label", deleteUserNamespaceHandler)
		ns.DELETE("/:ns_label/access", deleteUserNamespaceAccessHandler)

		ns.PUT("/:ns_label/name", renameUserNamespaceHandler)
		ns.PUT("/:ns_label/access", setUserNamespaceAccessHandler)
		ns.PUT("/:ns_label", resizeUserNamespaceHandler)

		deployment := ns.Group("/:ns_label/deployment")
		{
			deployment.POST("", createDeploymentHandler)

			deployment.GET("", getDeploymentsHandler)
			deployment.GET("/:deploy_label", getDeploymentByLabelHandler)

			deployment.DELETE("/:deploy_label", deleteDeploymentByLabelHandler)

			deployment.PUT("/:deploy_label/image", setContainerImageHandler)
			deployment.PUT("/:deploy_label", replaceDeploymentHandler)
			deployment.PUT("/:deploy_label/replicas", setReplicasHandler)
		}

		ingress := ns.Group("/:ns_label/ingress")
		{
			ingress.POST("", createIngressHandler)

			ingress.GET("", getUserIngressesHandler)

			ingress.DELETE("/:domain", deleteIngressHandler)
		}

		service := ns.Group("/:ns_label/service")
		{
			service.POST("", createServiceHandler)

			service.GET("", getServicesHandler)
			service.GET("/:service_label", getServiceHandler)

			service.PUT("/:service_label", updateServiceHandler)

			service.DELETE("/:service_label", deleteServiceHandler)
		}
	}

	nss := app.Group("/namespaces")
	{
		nss.GET("", utils.RequireAdminRole(rserrors.ErrPermissionDenied()), getAllNamespacesHandler)

		nss.DELETE("", utils.RequireAdminRole(rserrors.ErrPermissionDenied()), deleteAllUserNamespacesHandler)
	}

	vol := app.Group("/volume")
	{
		vol.POST("", createVolumeHandler)

		vol.GET("", getUserVolumesHandler)
		vol.GET("/:vol_label", getUserVolumeHandler)
		vol.GET("/:vol_label/access", getUserVolumeAccessesHandler)

		vol.DELETE("/:vol_label", deleteUserVolumeHandler)
		vol.DELETE("/:vol_label/access", deleteUserVolumeAccessHandler)

		vol.PUT("/:vol_label/name", renameUserVolumeHandler)
		vol.PUT("/:vol_label/access", setUserVolumeAccessHandler)
		vol.PUT("/:vol_label", resizeUserVolumeHandler)
	}

	vols := app.Group("/volumes")
	{
		vols.GET("", utils.RequireAdminRole(rserrors.ErrPermissionDenied()), getAllVolumesHandler)

		vols.DELETE("", utils.RequireAdminRole(rserrors.ErrPermissionDenied()), deleteAllUserVolumesHandler)
	}

	app.GET("/access", getUserResourceAccessesHandler)

	app.GET("/ingresses", utils.RequireAdminRole(rserrors.ErrPermissionDenied()), getAllIngressesHandler)

	domain := app.Group("/domain", utils.RequireAdminRole(rserrors.ErrPermissionDenied()))
	{
		domain.POST("", addDomainHandler)

		domain.GET("", getAllDomainsHandler)
		domain.GET("/:domain", getDomainHandler)

		domain.DELETE("/:domain", deleteDomainHandler)
	}

	storage := app.Group("/storage", utils.RequireAdminRole(rserrors.ErrPermissionDenied()))
	{
		storage.POST("", createStorageHandler)

		storage.GET("", getStoragesHandler)

		storage.PUT("/:storage_name", updateStorageHandler)

		storage.DELETE("/:storage_name", deleteStorageHandler)
	}

	adm := app.Group("/adm", utils.RequireAdminRole(rserrors.ErrPermissionDenied()))
	{
		adm.PUT("/access", setUserResourceAccessesHandler)
	}

	app.GET("/resources", getResourcesCountHandler)
}

func getResourcesCountHandler(ctx *gin.Context) {
	resp, err := srv.GetResourcesCount(ctx.Request.Context())
	if err != nil {
		ctx.AbortWithStatusJSON(handleError(err))
		return
	}

	ctx.JSON(http.StatusOK, resp)
}
