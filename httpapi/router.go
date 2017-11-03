package httpapi

import (
	"os"

	"bitbucket.org/exonch/resource-service/server"

	"github.com/gin-gonic/gin"
)

func NewGinEngine(srv server.ResourceSvcInterface) *gin.Engine {
	g := gin.New()
	g.Use(gin.Recovery())
	g.Use(gin.LoggerWithWriter(os.Stderr))
	g.Use(initializeContext(srv))
	g.Use(parseHeaders)
	g.Use(adminAction)

	g.POST("/namespace", CreateNamespace)
	g.DELETE("/namespace/:namespace", DeleteNamespace)
	g.GET("/namespace", ListNamespaces)
	//g.GET("/namespace/:namespace", GetNamespace)

	return g
}
