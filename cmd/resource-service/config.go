package main

import (
	"errors"
	"net/url"

	"git.containerum.net/ch/resource-service/pkg/clients"
	"git.containerum.net/ch/resource-service/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/en_US"
	"github.com/go-playground/universal-translator"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var defaultMongoAddr = cli.StringSlice{"localhost:27017"}

var flags = []cli.Flag{
	cli.BoolFlag{
		EnvVar: "CH_RESOURCE_DEBUG",
		Name:   "debug",
		Usage:  "start the server in debug mode",
	},
	cli.BoolFlag{
		EnvVar: "CH_RESOURCE_TEXTLOG",
		Name:   "textlog",
		Usage:  "output log in text format",
	},
	cli.StringFlag{
		EnvVar: "CH_RESOURCE_PORT",
		Name:   "port",
		Value:  "1213",
		Usage:  "port for resource-service server",
	},
	cli.StringFlag{
		EnvVar: "CH_RESOURCE_KUBE_API",
		Name:   "kube",
		Value:  "http",
		Usage:  "kube-api service type",
	},
	cli.StringFlag{
		EnvVar: "CH_RESOURCE_KUBE_API_ADDR",
		Name:   "kube_addr",
		Value:  "http://kube-api:1214",
		Usage:  "kube-api service address",
	},
	cli.StringFlag{
		EnvVar: "CH_RESOURCE_PERMISSIONS_ADDR",
		Name:   "permissions_addr",
		Value:  "http://permissions:4242",
		Usage:  "permissions service address",
	},
	cli.BoolFlag{
		EnvVar: "CH_RESOURCE_CORS",
		Name:   "cors",
		Usage:  "enable CORS",
	},
	cli.StringFlag{
		EnvVar: "CH_RESOURCE_MONGO_LOGIN",
		Name:   "mongo_login",
		Usage:  "MongoDB login",
	},
	cli.StringFlag{
		EnvVar: "CH_RESOURCE_MONGO_PASSWORD",
		Name:   "mongo_password",
		Usage:  "MongoDB password",
	},
	cli.StringSliceFlag{
		EnvVar: "CH_RESOURCE_MONGO_ADDR",
		Value:  &defaultMongoAddr,
		Name:   "mongo_addr",
		Usage:  "MongoDB address",
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

func setupTranslator() *ut.UniversalTranslator {
	return ut.New(en.New(), en.New(), en_US.New())
}

func setupMongo(c *cli.Context) (*db.MongoStorage, error) {
	dialInfo := mgo.DialInfo{
		Username: c.String("mongo_login"),
		Password: c.String("mongo_password"),
		Addrs:    c.StringSlice("mongo_addr"),
	}
	cfg := db.MongoConfig{
		Logger:   logrus.WithField("component", "mongo"),
		Debug:    c.Bool("debug"),
		DialInfo: dialInfo,
	}
	return db.NewMongo(cfg)
}

func setupKube(c *cli.Context) (*clients.Kube, error) {
	switch c.String("kube") {
	case "http":
		kubeurl, err := url.Parse(c.String("kube_addr"))
		if err != nil {
			return nil, err
		}
		client := clients.NewKubeHTTP(kubeurl)
		return &client, nil
	case "dummy":
		client := clients.NewDummyKube()
		return &client, nil
	default:
		return nil, errors.New("invalid kube-api client type")
	}
}

func setupPermissions(c *cli.Context) *clients.Permissions {
	client := clients.NewPermissionsHTTP(c.String("permissions_addr"))
	return &client
}
