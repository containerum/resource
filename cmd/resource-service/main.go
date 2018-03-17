package main

import (
	"net/http"
	"os"
	"time"

	"os/signal"

	"context"

	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/cherrylog"
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/gonic"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
	"git.containerum.net/ch/resource-service/pkg/routes"
	"git.containerum.net/ch/resource-service/pkg/server/impl"
	"git.containerum.net/ch/resource-service/pkg/util/validation"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
)

func exitOnError(err error) {
	if err != nil {
		logrus.WithError(err).Fatalf("can`t setup resource-service")
		os.Exit(1)
	}
}

func main() {
	exitOnError(setupLogger())

	logrus.Info("starting resource-service")

	listenAddr, err := getListenAddr()
	exitOnError(err)

	clients, constructors, err := setupServerClients()
	exitOnError(err)
	defer clients.Close()

	translate := setupTranslator()
	validate := validation.StandardResourceValidator(translate)

	g := gin.New()
	g.Use(gonic.Recovery(rserrors.ErrInternal, cherrylog.NewLogrusAdapter(logrus.WithField("component", "gin_recovery"))))
	g.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true))
	binding.Validator = &validation.GinValidatorV9{Validate: validate} // gin has no local validator

	tv := &routes.TranslateValidate{UniversalTranslator: translate, Validate: validate}
	routes.MainMiddlewareSetup(g, tv)
	routes.NamespaceHandlersSetup(g, tv, impl.NewNamespaceActionsImpl(clients, &impl.NamespaceActionsDB{
		NamespaceDB: constructors.NamespaceDB,
		StorageDB:   constructors.StorageDB,
		VolumeDB:    constructors.VolumeDB,
		AccessDB:    constructors.AccessDB,
	}))
	routes.AccessHandlersSetup(g, tv, impl.NewAccessActionsImpl(clients, &impl.AccessActionsDB{
		AccessDB:    constructors.AccessDB,
		NamespaceDB: constructors.NamespaceDB,
		VolumeDB:    constructors.VolumeDB,
	}))
	routes.DeployHandlersSetup(g, tv, impl.NewDeployActionsImpl(clients, &impl.DeployActionsDB{
		DeployDB:    constructors.DeployDB,
		NamespaceDB: constructors.NamespaceDB,
		EndpointsDB: constructors.EndpointsDB,
	}))
	routes.DomainHandlersSetup(g, tv, impl.NewDomainActionsImpl(clients, &impl.DomainActionsDB{
		DomainDB: constructors.DomainDB,
	}))
	routes.IngressHandlersSetup(g, tv, impl.NewIngressActionsImpl(clients, &impl.IngressActionsDB{
		NamespaceDB: constructors.NamespaceDB,
		ServiceDB:   constructors.ServiceDB,
		IngressDB:   constructors.IngressDB,
	}))
	routes.ServiceHandlersSetup(g, tv, impl.NewServiceActionsImpl(clients, &impl.ServiceActionsDB{
		ServiceDB:   constructors.ServiceDB,
		NamespaceDB: constructors.NamespaceDB,
		DomainDB:    constructors.DomainDB,
	}))
	routes.StorageHandlersSetup(g, tv, impl.NewStorageActionsImpl(clients, &impl.StorageActionsDB{
		StorageDB: constructors.StorageDB,
	}))
	routes.VolumeHandlersSetup(g, tv, impl.NewVolumeActionsImpl(clients, &impl.VolumeActionsDB{
		VolumeDB:  constructors.VolumeDB,
		StorageDB: constructors.StorageDB,
		AccessDB:  constructors.AccessDB,
	}))
	routes.ResourceCountHandlersSetup(g, tv, impl.NewResourceCountActionsImpl(clients, &impl.ResourceCountActionsDB{
		ResourceCountDB: constructors.ResourceCountDB,
	}))

	// for graceful shutdown
	httpsrv := &http.Server{
		Addr:    listenAddr,
		Handler: g,
	}

	// serve connections
	go exitOnError(httpsrv.ListenAndServe())

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt) // subscribe on interrupt event
	<-quit                            // wait for event
	logrus.Infoln("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	exitOnError(httpsrv.Shutdown(ctx))
}
