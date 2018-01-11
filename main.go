package main

import (
	"os"
	"time"

	"git.containerum.net/ch/resource-service/httpapi"

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

	gin := httpapi.NewGinEngine(srv)
	for {
		err = gin.Run(listenAddr)
		if err != nil {
			logrus.Errorf("gin error: %v", err)
		}
		time.Sleep(time.Second)
	}
}
