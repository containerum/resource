package main

import (
	"errors"
	"net/url"

	"fmt"
	"reflect"

	"os"

	"git.containerum.net/ch/resource-service/pkg/clients"
	"git.containerum.net/ch/resource-service/pkg/models"
	"git.containerum.net/ch/resource-service/pkg/models/postgres"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/en_US"
	"github.com/go-playground/universal-translator"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var flags = []cli.Flag{
	cli.BoolFlag{
		EnvVar: "CH_RESOURCE_DEBUG",
		Name:   "debug",
		Usage:  "start the server in debug mode",
	},
	cli.StringFlag{
		EnvVar: "CH_RESOURCE_PORT",
		Name:   "port",
		Value:  "1213",
		Usage:  "port for resource-service server",
	},
	cli.StringFlag{
		EnvVar: "CH_RESOURCE_KUBE_API_ADDR",
		Name:   "kube_addr",
		Value:  "config",
		Usage:  "kube-api service address",
	},
	cli.BoolFlag{
		EnvVar: "CH_KUBE_API_TEXTLOG",
		Name:   "textlog",
		Usage:  "output log in text format",
	},
	cli.BoolFlag{
		EnvVar: "CH_KUBE_API_CORS",
		Name:   "cors",
		Usage:  "enable CORS",
	},
}

func setupLogs(c *cli.Context) {
	if c.Bool("debug") {
		gin.SetMode(gin.DebugMode)
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		gin.SetMode(gin.ReleaseMode)
		logrus.SetLevel(logrus.InfoLevel)
	}

	if c.Bool("textlog") {
		logrus.SetFormatter(&logrus.TextFormatter{})
	} else {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
}

//TODO
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
		DeployDB:        postgres.NewDeployPG,
		IngressDB:       postgres.NewIngressPG,
		DomainDB:        postgres.NewDomainPG,
		ServiceDB:       postgres.NewServicePG,
		ResourceCountDB: postgres.NewResourceCountPG,
	}

	return db, constructors, err
}

//TODO
func setupKubeClient(addr string, debug bool) (clients.Kube, error) {
	switch {
	case debug && addr == "":
		return clients.NewDummyKube(), nil
	case addr != "":
		return clients.NewKubeHTTP(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for kube service")
	}
}

func setupServerClients(c *cli.Context) (*server.ResourceServiceClients, *server.ResourceServiceConstructors, error) {
	var ret server.ResourceServiceClients
	var constructors *server.ResourceServiceConstructors

	var err error

	//TODO
	if ret.DB, constructors, err = setupDB(os.Getenv("DB_URL"), os.Getenv("MIGRATION_URL")); err != nil {
		return nil, nil, err
	}
	//

	if ret.Kube, err = setupKubeClient(c.String("kube_addr"), c.Bool("debug")); err != nil {
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

func setupTranslator() *ut.UniversalTranslator {
	return ut.New(en.New(), en.New(), en_US.New())
}
