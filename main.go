package main

import (
	"os"

	"bitbucket.org/exonch/resource-manager/server"

	"github.com/sirupsen/logrus"
)

func main() {
	var (
		dbuser     = os.Getenv("DB_USER")
		dbpassword = os.Getenv("DB_PASSWORD")
		dbaddress  = os.Getenv("DB_ADDRESS")
	)
	srv := &server.ResourceManager{}
	err := srv.Initialize(nil, nil, nil, nil,
		"postgres://"+dbuser+":"+dbpassword+"@"+dbaddress+"/resource_manager?sslmode=disable")
	if err != nil {
		logrus.Fatalf("srv.Initialize error: %v", err)
	}
	logrus.Infof("ok")
}
