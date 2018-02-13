package routes

import (
	rstypes "git.containerum.net/ch/json-types/resource-service"
	umtypes "git.containerum.net/ch/json-types/user-manager"
	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gopkg.in/go-playground/validator.v8"
)

var srv server.ResourceService

var customValidator = validator.New(&validator.Config{TagName: "binding"})

func setupValidator() {
	rstypes.RegisterCustomTags(customValidator)

	// gin`s binding can not perform struct-level validations
	customValidator.RegisterStructValidation(createIngressRequestValidate, rstypes.CreateIngressRequest{})
}

// SetupRoutes sets up a router
func SetupRoutes(app *gin.Engine, server server.ResourceService) {
	srv = server

	setupValidator()

	app.Use(utils.SaveHeaders)
	app.Use(utils.PrepareContext)
	app.Use(utils.RequireHeaders(umtypes.UserIDHeader, umtypes.UserRoleHeader))
	app.Use(utils.SubstituteUserMiddleware)

	rstypes.RegisterCustomTagsGin(binding.Validator)

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
	}

	nss := app.Group("/namespaces")
	{
		nss.GET("", utils.RequireAdminRole, getAllNamespacesHandler)

		nss.DELETE("", utils.RequireAdminRole, deleteAllUserNamespacesHandler)
	}

	vol := app.Group("/volume")
	{
		vol.POST("", createVolumeHandler)

		vol.GET("", getUserVolumesHandler)
		vol.GET("/:vol_label", getUserVolumeHandler)
		vol.GET("/:vol_label/access", getUserVolumeAccessesHandler)

		vol.DELETE("/:vol_label", deleteUserVolumeHandler)
		vol.DELETE("/:vol_label", deleteUserVolumeAccessHandler)

		vol.PUT("/:vol_label/name", renameUserVolumeHandler)
		vol.PUT("/:vol_label/access", setUserVolumeAccessHandler)
		vol.PUT("/:vol_label", resizeUserVolumeHandler)
	}

	vols := app.Group("/volumes")
	{
		vols.GET("", utils.RequireAdminRole, getAllVolumesHandler)

		vols.DELETE("", utils.RequireAdminRole, deleteAllUserVolumesHandler)
	}

	app.GET("/access", getUserResourceAccessesHandler)

	app.GET("/ingresses", utils.RequireAdminRole, getAllIngressesHandler)

	domain := app.Group("/domain", utils.RequireAdminRole)
	{
		domain.POST("", addDomainHandler)

		domain.GET("", getAllDomainsHandler)
		domain.GET("/:domain", getDomainHandler)

		domain.DELETE("/:domain", deleteDomainHandler)
	}

	adm := app.Group("/adm", utils.RequireAdminRole)
	{
		adm.PUT("/access", setUserResourceAccessesHandler)
	}
}
