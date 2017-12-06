package main

import (
	"net/url"
	"os"
	"time"

	"bitbucket.org/exonch/resource-service/httpapi"
	"bitbucket.org/exonch/resource-service/server"
	"bitbucket.org/exonch/resource-service/server/other"

	"github.com/sirupsen/logrus"
)

func main() {
	var authSvc other.AuthSvc
	var billingSvc other.Billing
	var kubeSvc other.Kube
	var mailerSvc other.Mailer
	var volumeSvc other.VolumeSvc

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
		err = gin.Run(":1213")
		if err != nil {
			logrus.Errorf("gin error: %v", err)
		}
		time.Sleep(time.Second)
	}
}
