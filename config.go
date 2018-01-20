package main

import (
	"errors"
	"os"
	"strconv"

	"net/url"

	"fmt"
	"reflect"

	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/resource-service/server/other"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type operationMode int

const (
	modeDebug operationMode = iota
	modeRelease
)

var opMode operationMode

func setupLogger() error {
	mode := os.Getenv("MODE")
	switch mode {
	case "debug":
		opMode = modeDebug
		gin.SetMode(gin.DebugMode)
		logrus.SetLevel(logrus.DebugLevel)
	case "release", "":
		opMode = modeRelease
		gin.SetMode(gin.ReleaseMode)
		logrus.SetFormatter(&logrus.JSONFormatter{})

		logLevelString := os.Getenv("LOG_LEVEL")
		var level logrus.Level
		if logLevelString == "" {
			level = logrus.InfoLevel
		} else {
			levelI, err := strconv.Atoi(logLevelString)
			if err != nil {
				return err
			}
			level = logrus.Level(levelI)
			if level > logrus.DebugLevel || level < logrus.PanicLevel {
				return errors.New("invalid log level")
			}
		}
		logrus.SetLevel(level)
	default:
		return errors.New("invalid operation mode (must be 'debug' or 'release')")
	}
	return nil
}

func setupAuthClient(addr string) (other.AuthSvc, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return other.NewAuthSvcStub(), nil
	case addr != "":
		return other.NewAuthSvcGRPC(addr)
	default:
		return nil, errors.New("missing configuration for auth service")
	}
}

func setupBillingClient(addr string) (other.Billing, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return other.NewBillingStub(), nil
	case addr != "":
		return other.NewBillingHTTP(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for billing service")
	}
}

func setupKubeClient(addr string) (other.Kube, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return other.NewKubeStub(), nil
	case addr != "":
		return other.NewKubeHTTP(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for kube service")
	}
}

func setupMailerClient(addr string) (other.Mailer, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return other.NewMailerStub(), nil
	case addr != "":
		return other.NewMailerHTTP(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for mailer service")
	}
}

func setupVolumesClient(addr string) (other.VolumeSvc, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return other.NewVolumeSvcStub(), nil
	case addr != "":
		return other.NewVolumeSvcHTTP(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for volume service")
	}
}

func setupServer() (server.ResourceSvcInterface, error) {
	var clients server.ResourceSvcClients

	var err error
	if clients.Auth, err = setupAuthClient(os.Getenv("AUTH_ADDR")); err != nil {
		return nil, err
	}
	if clients.Billing, err = setupBillingClient(os.Getenv("BILLING_ADDR")); err != nil {
		return nil, err
	}
	if clients.Kube, err = setupKubeClient(os.Getenv("KUBE_ADDR")); err != nil {
		return nil, err
	}
	if clients.Mailer, err = setupMailerClient(os.Getenv("MAILER_ADDR")); err != nil {
		return nil, err
	}
	if clients.Volume, err = setupVolumesClient(os.Getenv("VOLUMES_ADDR")); err != nil {
		return nil, err
	}

	// print info about clients which implements Stringer
	v := reflect.ValueOf(clients)
	// trick to take type of interface, because of reflect.TypeOf((fmt.Stringer)(nil)) throws panic
	stringer := reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	for i := 0; i < reflect.TypeOf(clients).NumField(); i++ {
		f := v.Field(i)
		if f.Type().ConvertibleTo(stringer) {
			logrus.Infof("%s", f.Interface())
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
