package setup

import (
	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/middleware"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

// SetupRouter initializes and returns a Gin router with middleware and basic routes
func SetupRouter(logger *service.Logger) *gin.Engine {
	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(middleware.CORS())
	err := router.SetTrustedProxies(nil)
	if err != nil {
		panic(err.Error())
	}
	if cfg.Default().Site.Environment != types.EnvironmentDevelop {
		router.Use(middleware.Recovery(logger))
	}
	router.Use(middleware.Log(logger))
	// go tool pprof only on localhost
	if cfg.Default().Site.Environment == types.EnvironmentDevelop {
		pprof.Register(router)
	}
	return router
}
