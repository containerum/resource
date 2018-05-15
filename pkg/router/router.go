package router

import (
	"net/http"

	"time"

	"git.containerum.net/ch/resource-service/pkg/db"
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

func CreateRouter(mongo *db.MongoStorage, tv *m.TranslateValidate, enableCORS bool) http.Handler {
	e := gin.New()
	initMiddlewares(e, tv, enableCORS)

	//TODO
	deployHandlersSetup(e, tv, impl.NewDeployActionsImpl(mongo))
	domainHandlersSetup(e, tv, impl.NewDomainActionsImpl(mongo))
	ingressHandlersSetup(e, tv, impl.NewIngressActionsImpl(mongo))
	serviceHandlersSetup(e, tv, impl.NewServiceActionsImpl(mongo))
	resourceCountHandlersSetup(e, tv, impl.NewResourceCountActionsImpl(mongo))

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
		deployment.GET("", deployHandlers.GetDeploymentsListHandler)
		deployment.GET("/:deploy_label", deployHandlers.GetDeploymentHandler)

		deployment.POST("", deployHandlers.CreateDeploymentHandler)

		deployment.PUT("/:deploy_label", deployHandlers.UpdateDeploymentHandler)
		deployment.PUT("/:deploy_label/image", deployHandlers.SetContainerImageHandler)
		deployment.PUT("/:deploy_label/replicas", deployHandlers.SetReplicasHandler)

		deployment.DELETE("/:deploy_label", deployHandlers.DeleteDeploymentByLabelHandler)
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

	ingress := router.Group("/namespaces/:ns_label/ingresses")
	{
		ingress.GET("", ingressHandlers.GetIngressesListHandler)
		ingress.GET("/:ingress", ingressHandlers.GetIngressHandler)

		ingress.POST("", ingressHandlers.CreateIngressHandler)

		ingress.PUT("/:ingress", ingressHandlers.UpdateIngressHandler)

		ingress.DELETE("/:ingress", ingressHandlers.DeleteIngressHandler)
	}
}

func serviceHandlersSetup(router gin.IRouter, tv *m.TranslateValidate, backend server.ServiceActions) {
	serviceHandlers := h.ServiceHandlers{ServiceActions: backend, TranslateValidate: tv}

	service := router.Group("/namespaces/:ns_label/services")
	{
		service.GET("", serviceHandlers.GetServicesListHandler)
		service.GET("/:service_label", serviceHandlers.GetServiceHandler)

		service.POST("", serviceHandlers.CreateServiceHandler)

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
