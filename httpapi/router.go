package httpapi

import (
	"os"

	"github.com/gin-gonic/gin"
)

func NewGinEngine(srv server.ResourceManagerInterface) *gin.Engine {
	g := gin.New()
	g.Use(gin.Recovery())
	g.Use(gin.LoggerWithWriter(os.Stderr))
	g.Use(initializeContext(srv))
	g.Use(parseHeaders)

	g.POST("/namespace/:namespace", CreateNamespace)
	g.DELETE("/namespace/:namespace", DeleteNamespace)

	return g
}
