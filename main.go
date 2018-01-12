package main

import (
	"net/http"
	"os"
	"time"

	"os/signal"

	"context"

	"git.containerum.net/ch/resource-service/httpapi"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func exitOnError(err error) {
	if err != nil {
		logrus.WithError(err).Fatalf("can`t setup resource-service")
		os.Exit(1)
	}
}

func main() {
	logrus.Infof("starting resource-service version %s")
	exitOnError(setupLogger())

	listenAddr, err := getListenAddr()
	exitOnError(err)

	srv, err := setupServer()
	exitOnError(err)
	defer srv.Close()

	g := gin.New()
	g.Use(gin.RecoveryWithWriter(logrus.WithField("component", "gin_recovery").WriterLevel(logrus.ErrorLevel)))
	g.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true))

	exitOnError(httpapi.SetupGinEngine(srv, g))

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
