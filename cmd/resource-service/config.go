package main

import (
	"errors"
	"os"
	"strconv"

	"net/url"

	"fmt"
	"reflect"

	"git.containerum.net/ch/resource-service/pkg/clients"
	"git.containerum.net/ch/resource-service/pkg/models"
	"git.containerum.net/ch/resource-service/pkg/models/postgres"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/en_US"
	"github.com/go-playground/universal-translator"
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

func setupDB(connStr, migrationAddr string) (models.RelationalDB, *server.ResourceServiceConstructors, error) {
	if connStr == "" {
		return nil, nil, errors.New("db connection string was not specified")
	}
	if migrationAddr == "" {
		return nil, nil, errors.New("migrations address was not specified")
	}

	db, err := postgres.DBConnect(connStr, migrationAddr)
	constructors := &server.ResourceServiceConstructors{
		NamespaceDB:     postgres.NewNamespacePG,
		VolumeDB:        postgres.NewVolumePG,
		StorageDB:       postgres.NewStoragePG,
		DeployDB:        postgres.NewDeployPG,
		IngressDB:       postgres.NewIngressPG,
		DomainDB:        postgres.NewDomainPG,
		AccessDB:        postgres.NewAccessPG,
		ServiceDB:       postgres.NewServicePG,
		ResourceCountDB: postgres.NewResourceCountPG,
		EndpointsDB:     postgres.NewGlusterPG,
	}

	return db, constructors, err
}

func setupAuthClient(addr string) (clients.AuthSvc, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return clients.NewDummyAuthSvc(), nil
	case addr != "":
		return clients.NewAuthSvcGRPC(addr)
	default:
		return nil, errors.New("missing configuration for auth service")
	}
}

func setupBillingClient(addr string) (clients.Billing, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return clients.NewDummyBillingClient(), nil
	case addr != "":
		return clients.NewHTTPBillingClient(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for billing service")
	}
}

func setupKubeClient(addr string) (clients.Kube, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return clients.NewDummyKube(), nil
	case addr != "":
		return clients.NewKubeHTTP(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for kube service")
	}
}

func setupMailerClient(addr string) (clients.Mailer, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return clients.NewDummyMailer(), nil
	case addr != "":
		return clients.NewMailerHTTP(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for mailer service")
	}
}

// TODO: implement it
/*func setupVolumesClient(addr string) (clients.VolumeSvc, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return clients.NewVolumeSvcStub(), nil
	case addr != "":
		return clients.NewVolumeSvcHTTP(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for volume service")
	}
}*/

func setupUserClient(addr string) (clients.UserManagerClient, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return clients.NewUserManagerStub(), nil
	case addr != "":
		return clients.NewHTTPUserManagerClient(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for user-manager service")
	}
}

func setupServerClients() (*server.ResourceServiceClients, *server.ResourceServiceConstructors, error) {
	var ret server.ResourceServiceClients
	var constructors *server.ResourceServiceConstructors

	var err error
	if ret.DB, constructors, err = setupDB(os.Getenv("DB_URL"), os.Getenv("MIGRATION_URL")); err != nil {
		return nil, nil, err
	}
	if ret.Auth, err = setupAuthClient(os.Getenv("AUTH_ADDR")); err != nil {
		return nil, nil, err
	}
	if ret.Billing, err = setupBillingClient(os.Getenv("BILLING_ADDR")); err != nil {
		return nil, nil, err
	}
	if ret.Kube, err = setupKubeClient(os.Getenv("KUBE_ADDR")); err != nil {
		return nil, nil, err
	}
	if ret.Mail, err = setupMailerClient(os.Getenv("MAILER_ADDR")); err != nil {
		return nil, nil, err
	}
	/*	if ret.Volume, err = setupVolumesClient(os.Getenv("VOLUMES_ADDR")); err != nil {
		return nil, err
	}*/
	if ret.User, err = setupUserClient(os.Getenv("USER_ADDR")); err != nil {
		return nil, nil, err
	}

	// print info about ret which implements Stringer
	v := reflect.ValueOf(ret)
	for i := 0; i < reflect.TypeOf(ret).NumField(); i++ {
		f := v.Field(i)
		if str, ok := f.Interface().(fmt.Stringer); ok {
			logrus.Infof("%s", str)
		}
	}

	return &ret, constructors, nil
}

func getListenAddr() (la string, err error) {
	if la = os.Getenv("LISTEN_ADDR"); la == "" {
		return "", errors.New("environment LISTEN_ADDR is not specified")
	}
	return la, nil
}

func setupTranslator() *ut.UniversalTranslator {
	return ut.New(en.New(), en.New(), en_US.New())
}
