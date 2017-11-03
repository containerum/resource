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
	var (
		dbuser     = os.Getenv("DB_USER")
		dbpassword = os.Getenv("DB_PASSWORD")
		dbaddress  = os.Getenv("DB_ADDRESS")
	)
	srv := &server.ResourceManager{}
	err := srv.Initialize(
		&url.URL{
			Scheme: "http",
			Host:   "localhost:1007",
		},
		&url.URL{
			Scheme: "http",
			Host:   "localhost:1212",
		},
		nil,
		nil,
		"postgres://"+dbuser+":"+dbpassword+"@"+dbaddress+"/resource_service?sslmode=disable",
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
