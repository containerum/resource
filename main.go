package main

import (
	"net/url"
	"os"
	"time"

	"git.containerum.net/ch/resource-service/httpapi"
	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/resource-service/server/other"

	"github.com/sirupsen/logrus"
)

const Version = "0.0.1"
const BuildTime = "unspecified"

func main() {
	var authSvc other.AuthSvc
	var billingSvc other.Billing
	var kubeSvc other.Kube
	var mailerSvc other.Mailer
	var volumeSvc other.VolumeSvc

	logrus.Infof("starting resource-service version %s build time %s", Version, BuildTime)

	opmode := os.Getenv("MODE")
	switch opmode {
	case "debug":
		os.Setenv("GIN_MODE", "debug")

		if u := os.Getenv("AUTH_ADDR"); u != "" {
			authSvc = other.NewAuthSvcHTTP(&url.URL{
				Scheme: "http",
				Host:   u,
			})
		} else {
			authSvc = other.NewAuthSvcStub()
		}

		if u := os.Getenv("BILLING_ADDR"); u != "" {
			billingSvc = other.NewBillingHTTP(&url.URL{
				Scheme: "http",
				Host:   u,
			})
		} else {
			billingSvc = other.NewBillingStub()
		}

		if u := os.Getenv("KUBE_ADDR"); u != "" {
			kubeSvc = other.NewKubeHTTP(&url.URL{
				Scheme: "http",
				Host:   u,
			})
		} else {
			kubeSvc = other.NewKubeStub()
		}

		if u := os.Getenv("MAILER_ADDR"); u != "" {
			mailerSvc = other.NewMailerHTTP(&url.URL{
				Scheme: "http",
				Host:   u,
			})
		} else {
			mailerSvc = other.NewMailerStub()
		}

		if u := os.Getenv("VOLUMES_ADDR"); u != "" {
			volumeSvc = other.NewVolumeSvcHTTP(&url.URL{
				Scheme: "http",
				Host:   u,
			})
		} else {
			volumeSvc = other.NewVolumeSvcStub()
		}

	case "release", "":
		opmode = "release"
		os.Setenv("GIN_MODE", "release")

		if u := os.Getenv("AUTH_ADDR"); u != "" {
			authSvc = other.NewAuthSvcHTTP(&url.URL{
				Scheme: "http",
				Host:   u,
			})
		} else {
			logrus.Fatalf("missing configuration for auth service")
		}

		if u := os.Getenv("BILLING_ADDR"); u != "" {
			billingSvc = other.NewBillingHTTP(&url.URL{
				Scheme: "http",
				Host:   u,
			})
		} else {
			logrus.Fatalf("missing configuration for billing service")
		}

		if u := os.Getenv("KUBE_ADDR"); u != "" {
			kubeSvc = other.NewKubeHTTP(&url.URL{
				Scheme: "http",
				Host:   u,
			})
		} else {
			logrus.Fatalf("missing configuration for billing service")
		}

		if u := os.Getenv("MAILER_ADDR"); u != "" {
			mailerSvc = other.NewMailerHTTP(&url.URL{
				Scheme: "http",
				Host:   u,
			})
		} else {
			logrus.Fatalf("missing configuration for billing service")
		}

		if u := os.Getenv("VOLUMES_ADDR"); u != "" {
			volumeSvc = other.NewVolumeSvcHTTP(&url.URL{
				Scheme: "http",
				Host:   u,
			})
		} else {
			logrus.Fatalf("missing configuration for billing service")
		}

	default:
		logrus.Fatalf("environment MODE is neither debug, nor release")
	}

	if os.Getenv("LISTEN_ADDR") == "" {
		logrus.Fatalf("environment LISTEN_ADDR is not specified")
	}

	logrus.Infof("authSvc %v", authSvc)
	logrus.Infof("billingSvc %v", billingSvc)
	logrus.Infof("kubeSvc %v", kubeSvc)
	logrus.Infof("mailerSvc %v", mailerSvc)
	logrus.Infof("volumeSvc %v", volumeSvc)
	logrus.Infof("database url=%s", os.Getenv("DB_URL"))
	logrus.Infof("database migrations url=%s", os.Getenv("MIGRATION_URL"))
	logrus.Infof("listen address %s", os.Getenv("LISTEN_ADDR"))

	srv := &server.ResourceSvc{}
	err := srv.Initialize(
		authSvc,
		billingSvc,
		kubeSvc,
		mailerSvc,
		volumeSvc,
		os.Getenv("DB_URL"),
	)
	if err != nil {
		logrus.Fatalf("srv.Initialize error: %v", err)
	}

	gin := httpapi.NewGinEngine(srv)
	for {
		err = gin.Run(os.Getenv("LISTEN_ADDR"))
		if err != nil {
			logrus.Errorf("gin error: %v", err)
		}
		time.Sleep(time.Second)
	}
}
