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
	"git.containerum.net/ch/resource-service/pkg/util/validation"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
)

//go:generate noice -t ../Errors.toml -o ../pkg/

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

	srv, err := setupServer()
	exitOnError(err)
	defer srv.Close()

	translate := setupTranslator()
	validate := validation.StandardResourceValidator(translate)

	g := gin.New()
	g.Use(gonic.Recovery(rserrors.ErrInternal, cherrylog.NewLogrusAdapter(logrus.WithField("component", "gin_recovery"))))
	g.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true))
	binding.Validator = &validation.GinValidatorV9{Validate: validate} // gin has no local validator

	routes.SetupRoutes(g, &routes.TranslateValidate{UniversalTranslator: translate, Validate: validate}, srv)

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
