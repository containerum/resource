package router

import (
	"net/http"

	"time"

	h "git.containerum.net/ch/resource-service/pkg/router/handlers"
	m "git.containerum.net/ch/resource-service/pkg/router/middleware"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"git.containerum.net/ch/resource-service/pkg/server"
	"git.containerum.net/ch/resource-service/pkg/server/impl"
	"git.containerum.net/ch/resource-service/pkg/util/validation"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/containerum/cherry/adaptors/gonic"
	"github.com/containerum/utils/httputil"
	headers "github.com/containerum/utils/httputil"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
)

func CreateRouter(clients *server.ResourceServiceClients, constructors *server.ResourceServiceConstructors, tv *m.TranslateValidate, enableCORS bool) http.Handler {
	e := gin.New()
	initMiddlewares(e, tv, enableCORS)

	//TODO
	deployHandlersSetup(e, tv, impl.NewDeployActionsImpl(clients, &impl.DeployActionsDB{
		DeployDB:    constructors.DeployDB,
		NamespaceDB: constructors.NamespaceDB,
	}))
	domainHandlersSetup(e, tv, impl.NewDomainActionsImpl(clients, &impl.DomainActionsDB{
		DomainDB: constructors.DomainDB,
	}))
	ingressHandlersSetup(e, tv, impl.NewIngressActionsImpl(clients, &impl.IngressActionsDB{
		NamespaceDB: constructors.NamespaceDB,
		ServiceDB:   constructors.ServiceDB,
		IngressDB:   constructors.IngressDB,
	}))
	serviceHandlersSetup(e, tv, impl.NewServiceActionsImpl(clients, &impl.ServiceActionsDB{
		ServiceDB:   constructors.ServiceDB,
		NamespaceDB: constructors.NamespaceDB,
		DomainDB:    constructors.DomainDB,
		IngressDB:   constructors.IngressDB,
	}))
	resourceCountHandlersSetup(e, tv, impl.NewResourceCountActionsImpl(clients, &impl.ResourceCountActionsDB{
		ResourceCountDB: constructors.ResourceCountDB,
	}))

	return e
}

func initMiddlewares(e gin.IRouter, tv *m.TranslateValidate, enableCORS bool) {
	/* CORS */
	if enableCORS {
		cfg := cors.DefaultConfig()
		cfg.AllowAllOrigins = true
		cfg.AddAllowMethods(http.MethodDelete)
		cfg.AddAllowHeaders(headers.UserRoleXHeader, headers.UserIDXHeader)
		e.Use(cors.New(cfg))
	}
	e.Use(gonic.Recovery(rserrors.ErrInternal, cherrylog.NewLogrusAdapter(logrus.WithField("component", "gin_recovery"))))
	e.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true))
	binding.Validator = &validation.GinValidatorV9{Validate: tv.Validate} // gin has no local validator

	e.Use(httputil.SaveHeaders)
	e.Use(httputil.PrepareContext)
	e.Use(httputil.RequireHeaders(rserrors.ErrValidation, headers.UserIDXHeader, headers.UserRoleXHeader))
	e.Use(tv.ValidateHeaders(map[string]string{
		headers.UserIDXHeader:   "uuid",
		headers.UserRoleXHeader: "eq=admin|eq=user",
	}))
	e.Use(httputil.SubstituteUserMiddleware(tv.Validate, tv.UniversalTranslator, rserrors.ErrValidation))
}

func deployHandlersSetup(router gin.IRouter, tv *m.TranslateValidate, backend server.DeployActions) {
	deployHandlers := h.DeployHandlers{DeployActions: backend, TranslateValidate: tv}

	deployment := router.Group("/namespaces/:ns_label/deployments")
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

func domainHandlersSetup(router gin.IRouter, tv *m.TranslateValidate, backend server.DomainActions) {
	domainHandlers := h.DomainHandlers{DomainActions: backend, TranslateValidate: tv}

	domain := router.Group("/domains", httputil.RequireAdminRole(rserrors.ErrPermissionDenied))
	{
		domain.POST("", domainHandlers.AddDomainHandler)

		domain.GET("", domainHandlers.GetAllDomainsHandler)
		domain.GET("/:domain", domainHandlers.GetDomainHandler)

		domain.DELETE("/:domain", domainHandlers.DeleteDomainHandler)
	}
}

func ingressHandlersSetup(router gin.IRouter, tv *m.TranslateValidate, backend server.IngressActions) {
	ingressHandlers := h.IngressHandlers{IngressActions: backend, TranslateValidate: tv}

	ingress := router.Group("/namespaces/:ns_label/ingresses")
	{
		ingress.POST("", ingressHandlers.CreateIngressHandler)

		ingress.GET("", ingressHandlers.GetUserIngressesHandler)

		ingress.DELETE("/:domain", ingressHandlers.DeleteIngressHandler)
	}
}

func serviceHandlersSetup(router gin.IRouter, tv *m.TranslateValidate, backend server.ServiceActions) {
	serviceHandlers := h.ServiceHandlers{ServiceActions: backend, TranslateValidate: tv}

	service := router.Group("/namespaces/:ns_label/services")
	{
		service.POST("", serviceHandlers.CreateServiceHandler)

		service.GET("", serviceHandlers.GetServicesHandler)
		service.GET("/:service_label", serviceHandlers.GetServiceHandler)

		service.PUT("/:service_label", serviceHandlers.UpdateServiceHandler)

		service.DELETE("/:service_label", serviceHandlers.DeleteServiceHandler)
	}
}

func resourceCountHandlersSetup(router gin.IRouter, tv *m.TranslateValidate, backend server.ResourceCountActions) {
	router.GET("/resources", func(ctx *gin.Context) {
		resp, err := backend.GetResourcesCount(ctx.Request.Context())
		if err != nil {
			ctx.AbortWithStatusJSON(tv.HandleError(err))
			return
		}

		ctx.JSON(http.StatusOK, resp)
	})
}
