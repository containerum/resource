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
	"github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
)

func CreateRouter(mongo *db.MongoStorage, permissions *clients.Permissions, kube *clients.Kube, tv *m.TranslateValidate, access httputil.AccessChecker, enableCORS bool) http.Handler {
	e := gin.New()
	initMiddlewares(e, tv, enableCORS)
	deployHandlersSetup(e, tv, access, impl.NewDeployActionsImpl(mongo, permissions, kube))
	domainHandlersSetup(e, tv, impl.NewDomainActionsImpl(mongo))
	ingressHandlersSetup(e, tv, access, impl.NewIngressActionsImpl(mongo, kube))
	serviceHandlersSetup(e, tv, access, impl.NewServiceActionsImpl(mongo, permissions, kube))
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
}

func deployHandlersSetup(router gin.IRouter, tv *m.TranslateValidate, access httputil.AccessChecker, backend server.DeployActions) {
	deployHandlers := h.DeployHandlers{DeployActions: backend, TranslateValidate: tv}

	deployment := router.Group("/projects/:project/namespaces/:namespace/deployments")
	{
		deployment.GET("", access.CheckAccess(model.AccessGuest), deployHandlers.GetDeploymentsListHandler)
		deployment.GET("/:deployment", access.CheckAccess(model.AccessGuest), deployHandlers.GetActiveDeploymentHandler)
		deployment.GET("/:deployment/versions", access.CheckAccess(model.AccessGuest), deployHandlers.GetDeploymentVersionsListHandler)
		deployment.GET("/:deployment/versions/:version", access.CheckAccess(model.AccessGuest), deployHandlers.GetDeploymentVersionHandler)
		deployment.GET("/:deployment/versions/:version/diff", access.CheckAccess(model.AccessGuest), deployHandlers.DiffDeploymentPreviousVersionsHandler)
		deployment.GET("/:deployment/versions/:version/diff/:version2", access.CheckAccess(model.AccessGuest), deployHandlers.DiffDeploymentVersionsHandler)

		deployment.POST("", access.CheckAccess(model.AccessMaster), deployHandlers.CreateDeploymentHandler)
		deployment.POST("/:deployment/versions/:version", access.CheckAccess(model.AccessMaster), deployHandlers.ChangeActiveDeploymentHandler)

		deployment.PUT("/:deployment", access.CheckAccess(model.AccessMaster), deployHandlers.UpdateDeploymentHandler)
		deployment.PUT("/:deployment/image", access.CheckAccess(model.AccessMaster), deployHandlers.SetContainerImageHandler)
		deployment.PUT("/:deployment/replicas", access.CheckAccess(model.AccessMaster), deployHandlers.SetReplicasHandler)
		deployment.PUT("/:deployment/versions/:version", access.CheckAccess(model.AccessMaster), deployHandlers.RenameVersionHandler)

		deployment.DELETE("/:deployment", access.CheckAccess(model.AccessMaster), deployHandlers.DeleteDeploymentHandler)
		deployment.DELETE("/:deployment/versions/:version", access.CheckAccess(model.AccessMaster), deployHandlers.DeleteDeploymentVersionHandler)
		deployment.DELETE("", deployHandlers.DeleteAllDeploymentsHandler)
	}
	router.DELETE("/projects/:project/namespaces/:namespace/solutions/:solution/deployments", access.CheckAccess(model.AccessMaster), deployHandlers.DeleteAllSolutionDeploymentsHandler)
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

func ingressHandlersSetup(router gin.IRouter, tv *m.TranslateValidate, access httputil.AccessChecker, backend server.IngressActions) {
	ingressHandlers := h.IngressHandlers{IngressActions: backend, TranslateValidate: tv}

	ingress := router.Group("/projects/:project/namespaces/:namespace/ingresses")
	{
		ingress.GET("", access.CheckAccess(model.AccessGuest), ingressHandlers.GetIngressesListHandler)
		ingress.GET("/:ingress", access.CheckAccess(model.AccessGuest), ingressHandlers.GetIngressHandler)

		ingress.POST("", access.CheckAccess(model.AccessMaster), ingressHandlers.CreateIngressHandler)

		ingress.PUT("/:ingress", access.CheckAccess(model.AccessMaster), ingressHandlers.UpdateIngressHandler)

		ingress.DELETE("/:ingress", access.CheckAccess(model.AccessMaster), ingressHandlers.DeleteIngressHandler)
		ingress.DELETE("", ingressHandlers.DeleteAllIngressesHandler)
	}
}

func serviceHandlersSetup(router gin.IRouter, tv *m.TranslateValidate, access httputil.AccessChecker, backend server.ServiceActions) {
	serviceHandlers := h.ServiceHandlers{ServiceActions: backend, TranslateValidate: tv}

	service := router.Group("/projects/:project/namespaces/:namespace/services")
	{
		service.GET("", access.CheckAccess(model.AccessGuest), serviceHandlers.GetServicesListHandler)
		service.GET("/:service", access.CheckAccess(model.AccessGuest), serviceHandlers.GetServiceHandler)

		service.POST("", access.CheckAccess(model.AccessMaster), serviceHandlers.CreateServiceHandler)

		service.PUT("/:service", access.CheckAccess(model.AccessMaster), serviceHandlers.UpdateServiceHandler)

		service.DELETE("/:service", access.CheckAccess(model.AccessMaster), serviceHandlers.DeleteServiceHandler)
		service.DELETE("", serviceHandlers.DeleteAllServicesHandler)
	}
	router.DELETE("/projects/:project/namespaces/:namespace/solutions/:solution/services", access.CheckAccess(model.AccessMaster), serviceHandlers.DeleteAllSolutionServicesHandler)
}

func resourceCountHandlersSetup(router gin.IRouter, tv *m.TranslateValidate, backend server.ResourcesActions) {
	resourceHandlers := h.ResourceHandlers{ResourcesActions: backend, TranslateValidate: tv}
	router.DELETE("/projects/:project/namespaces/:namespace", resourceHandlers.DeleteAllResourcesInNamespaceHandler)
	router.DELETE("/projects/:project/namespaces", resourceHandlers.DeleteAllResourcesHandler)
	router.GET("/resources", resourceHandlers.GetResourcesCountHandler)
}
