package main

import (
	"errors"
	"os"
	"strconv"

	"net/url"

	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/resource-service/server/other"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func setupLogger() error {
	opMode := os.Getenv("MODE")
	switch opMode {
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "release", "":
		gin.SetMode(gin.ReleaseMode)
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		return errors.New("invalid operation mode (must be 'debug' or 'release')")
	}

	logLevelString := os.Getenv("LOG_LEVEL")
	var level logrus.Level
	if logLevelString == "" {
		level = logrus.InfoLevel
	}
	levelI, err := strconv.Atoi(logLevelString)
	if err != nil {
		return err
	}
	level = logrus.Level(levelI)
	if level > logrus.DebugLevel || level < logrus.PanicLevel {
		return errors.New("invalid log level")
	}
	logrus.SetLevel(level)
	return nil
}

func setupServer() (server.ResourceSvcInterface, error) {
	var clients server.ResourceSvcClients

	if u := os.Getenv("AUTH_ADDR"); u != "" {
		clients.Auth = other.NewAuthSvcHTTP(&url.URL{
			Scheme: "http",
			Host:   u,
		})
	} else {
		if logrus.GetLevel() == logrus.DebugLevel {
			clients.Auth = other.NewAuthSvcStub()
		} else {
			return nil, errors.New("missing configuration for auth service")
		}
	}

	if u := os.Getenv("BILLING_ADDR"); u != "" {
		clients.Billing = other.NewBillingHTTP(&url.URL{
			Scheme: "http",
			Host:   u,
		})
	} else {
		clients.Billing = other.NewBillingStub()
	}

	if u := os.Getenv("KUBE_ADDR"); u != "" {
		clients.Kube = other.NewKubeHTTP(&url.URL{
			Scheme: "http",
			Host:   u,
		})
	} else {
		if logrus.GetLevel() == logrus.DebugLevel {
			clients.Kube = other.NewKubeStub()
		} else {
			return nil, errors.New("missing configuration for kube service")
		}
	}

	if u := os.Getenv("MAILER_ADDR"); u != "" {
		clients.Mailer = other.NewMailerHTTP(&url.URL{
			Scheme: "http",
			Host:   u,
		})
	} else {
		if logrus.GetLevel() == logrus.DebugLevel {
			clients.Mailer = other.NewMailerStub()
		} else {
			return nil, errors.New("missing configuration for mailer service")
		}
	}

	if u := os.Getenv("VOLUMES_ADDR"); u != "" {
		clients.Volume = other.NewVolumeSvcHTTP(&url.URL{
			Scheme: "http",
			Host:   u,
		})
	} else {
		if logrus.GetLevel() == logrus.DebugLevel {
			clients.Volume = other.NewVolumeSvcStub()
		} else {
			return nil, errors.New("missing configuration for volume service")
		}
	}

	logrus.Infof("database url=%s", os.Getenv("DB_URL"))
	logrus.Infof("database migrations url=%s", os.Getenv("MIGRATION_URL"))

	srv, err := server.NewResourceSvc(clients, os.Getenv("DB_URL"))
	if err != nil {
		return nil, errors.New("srv.Initialize error: " + err.Error())
	}
	return srv, nil
}

func getListenAddr() (la string, err error) {
	if la = os.Getenv("LISTEN_ADDR"); la == "" {
		return "", errors.New("environment LISTEN_ADDR is not specified")
	}
	return la, nil
}
