package main

import (
	"net/url"
	"os"
	"time"

	"bitbucket.org/exonch/resource-service/httpapi"
	"bitbucket.org/exonch/resource-service/server"

	"github.com/sirupsen/logrus"
)

func main() {
	srv := &server.ResourceSvc{}
	err := srv.Initialize(
		&url.URL{
			Scheme: "http",
			Host:   os.Getenv("AUTH_ADDR"),
		},
		&url.URL{
			Scheme: "http",
			Host:   os.Getenv("BILLING_ADDR"),
		},
		&url.URL{
			Scheme: "http",
			Host:   os.Getenv("KUBE_ADDR"),
		},
		nil,
		nil,
		os.Getenv("DB_URL"),
	)
	if err != nil {
		logrus.Fatalf("srv.Initialize error: %v", err)
	}

	gin := httpapi.NewGinEngine(srv)
	for {
		err = gin.Run(":1213")
		if err != nil {
			logrus.Errorf("gin error: %v", err)
		}
		time.Sleep(time.Second)
	}
}
