package router

import (
	"net/http"

	"time"

	"git.containerum.net/ch/resource-service/pkg/clients"
	"git.containerum.net/ch/resource-service/pkg/db"
	h "git.containerum.net/ch/resource-service/pkg/router/handlers"
	m "git.containerum.net/ch/resource-service/pkg/router/middleware"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"git.containerum.net/ch/resource-service/pkg/server"
	"git.containerum.net/ch/resource-service/pkg/server/impl"
	"git.containerum.net/ch/resource-service/pkg/util/validation"
	"git.containerum.net/ch/resource-service/static"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/containerum/cherry/adaptors/gonic"
	"github.com/containerum/utils/httputil"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
)

func CreateRouter(mongo *db.MongoStorage, permissions *clients.Permissions, kube *clients.Kube, tv *m.TranslateValidate, enableCORS bool) http.Handler {
	e := gin.New()
	initMiddlewares(e, tv, enableCORS)
	deployHandlersSetup(e, tv, impl.NewDeployActionsImpl(mongo, permissions, kube))
	domainHandlersSetup(e, tv, impl.NewDomainActionsImpl(mongo))
	ingressHandlersSetup(e, tv, impl.NewIngressActionsImpl(mongo, kube))
	serviceHandlersSetup(e, tv, impl.NewServiceActionsImpl(mongo, permissions, kube))
	resourceCountHandlersSetup(e, tv, impl.NewResourcesActionsImpl(mongo))

	return e
}

func initMiddlewares(e gin.IRouter, tv *m.TranslateValidate, enableCORS bool) {
	/* CORS */
	if enableCORS {
		cfg := cors.DefaultConfig()
		cfg.AllowAllOrigins = true
		cfg.AddAllowMethods(http.MethodDelete)
		cfg.AddAllowHeaders(httputil.UserRoleXHeader, httputil.UserIDXHeader, httputil.UserNamespacesXHeader)
		e.Use(cors.New(cfg))
	}
	e.Group("/static").
		StaticFS("/", static.HTTP)
	e.Use(gonic.Recovery(rserrors.ErrInternal, cherrylog.NewLogrusAdapter(logrus.WithField("component", "gin_recovery"))))
	e.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true))
	binding.Validator = &validation.GinValidatorV9{Validate: tv.Validate} // gin has no local validator

	e.Use(httputil.SaveHeaders)
	e.Use(httputil.PrepareContext)
	e.Use(httputil.RequireHeaders(rserrors.ErrValidation, httputil.UserIDXHeader, httputil.UserRoleXHeader))
	e.Use(tv.ValidateHeaders(map[string]string{
		httputil.UserIDXHeader:   "uuid",
		httputil.UserRoleXHeader: "eq=admin|eq=user",
	}))
	e.Use(httputil.SubstituteUserMiddleware(tv.Validate, tv.UniversalTranslator, rserrors.ErrValidation))
	e.Use(m.RequiredUserHeaders())
}

func deployHandlersSetup(router gin.IRouter, tv *m.TranslateValidate, backend server.DeployActions) {
	deployHandlers := h.DeployHandlers{DeployActions: backend, TranslateValidate: tv}

	deployment := router.Group("/namespaces/:namespace/deployments")
	{
		deployment.GET("", m.ReadAccess, deployHandlers.GetDeploymentsListHandler)
		deployment.GET("/:deployment", m.ReadAccess, deployHandlers.GetDeploymentHandler)
		deployment.GET("/:deployment/versions", m.ReadAccess, deployHandlers.GetDeploymentVersionsListHandler)
		deployment.GET("/:deployment/versions/:version", m.ReadAccess, deployHandlers.GetDeploymentVersionHandler)

		deployment.GET("/:deployment/versions/:version/diff", m.ReadAccess, deployHandlers.DiffDeplooymentPreviousVersionsHandler)
		deployment.GET("/:deployment/versions/:version/diff/:version2", m.ReadAccess, deployHandlers.DiffDeplooymentVersionsHandler)

		deployment.POST("", m.WriteAccess, deployHandlers.CreateDeploymentHandler)
		deployment.POST("/:deployment/versions/:version", m.WriteAccess, deployHandlers.ChangeActiveDeploymentHandler)

		deployment.PUT("/:deployment", m.WriteAccess, deployHandlers.UpdateDeploymentHandler)
		deployment.PUT("/:deployment/image", m.WriteAccess, deployHandlers.SetContainerImageHandler)
		deployment.PUT("/:deployment/replicas", m.WriteAccess, deployHandlers.SetReplicasHandler)
		deployment.PUT("/:deployment/versions/:version", m.WriteAccess, deployHandlers.RenameVersionHandler)

		deployment.DELETE("/:deployment", m.WriteAccess, deployHandlers.DeleteDeploymentHandler)
		deployment.DELETE("/:deployment/version/:version", m.WriteAccess, deployHandlers.DeleteDeploymentVersionHandler)
		deployment.DELETE("", deployHandlers.DeleteAllDeploymentsHandler)
	}
}

func domainHandlersSetup(router gin.IRouter, tv *m.TranslateValidate, backend server.DomainActions) {
	domainHandlers := h.DomainHandlers{DomainActions: backend, TranslateValidate: tv}

	domain := router.Group("/domains", httputil.RequireAdminRole(rserrors.ErrPermissionDenied))
	{
		domain.GET("", domainHandlers.GetDomainsListHandler)
		domain.GET("/:domain", domainHandlers.GetDomainHandler)

		domain.POST("", domainHandlers.AddDomainHandler)

		domain.DELETE("/:domain", domainHandlers.DeleteDomainHandler)
	}
}

func ingressHandlersSetup(router gin.IRouter, tv *m.TranslateValidate, backend server.IngressActions) {
	ingressHandlers := h.IngressHandlers{IngressActions: backend, TranslateValidate: tv}

	ingress := router.Group("/namespaces/:namespace/ingresses")
	{
		ingress.GET("", m.ReadAccess, ingressHandlers.GetIngressesListHandler)
		ingress.GET("/:ingress", m.ReadAccess, ingressHandlers.GetIngressHandler)

		ingress.POST("", m.WriteAccess, ingressHandlers.CreateIngressHandler)

		ingress.PUT("/:ingress", m.WriteAccess, ingressHandlers.UpdateIngressHandler)

		ingress.DELETE("/:ingress", m.WriteAccess, ingressHandlers.DeleteIngressHandler)
		ingress.DELETE("", ingressHandlers.DeleteAllIngressesHandler)
	}
}

func serviceHandlersSetup(router gin.IRouter, tv *m.TranslateValidate, backend server.ServiceActions) {
	serviceHandlers := h.ServiceHandlers{ServiceActions: backend, TranslateValidate: tv}

	service := router.Group("/namespaces/:namespace/services")
	{
		service.GET("", m.ReadAccess, serviceHandlers.GetServicesListHandler)
		service.GET("/:service", m.ReadAccess, serviceHandlers.GetServiceHandler)

		service.POST("", m.WriteAccess, serviceHandlers.CreateServiceHandler)

		service.PUT("/:service", m.WriteAccess, serviceHandlers.UpdateServiceHandler)

		service.DELETE("/:service", m.WriteAccess, serviceHandlers.DeleteServiceHandler)
		service.DELETE("", serviceHandlers.DeleteAllServicesHandler)
	}
}

func resourceCountHandlersSetup(router gin.IRouter, tv *m.TranslateValidate, backend server.ResourcesActions) {
	resourceHandlers := h.ResourceHandlers{ResourcesActions: backend, TranslateValidate: tv}
	router.DELETE("/namespaces/:namespace", resourceHandlers.DeleteAllResourcesInNamespaceHandler)
	router.DELETE("/namespaces", resourceHandlers.DeleteAllResourcesHandler)
	router.GET("/resources", resourceHandlers.GetResourcesCountHandler)
}
