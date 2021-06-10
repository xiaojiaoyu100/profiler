package engine

import (
	"net/http"

	"github.com/xiaojiaoyu100/profiler/collector/server/middleware"

	"github.com/gin-gonic/gin"
	"github.com/xiaojiaoyu100/profiler/collector/env"
	"github.com/xiaojiaoyu100/profiler/collector/server/controller/profile"
)

func Engine(env *env.Env) *gin.Engine {
	engine := gin.New()
	engine.HandleMethodNotAllowed = true
	engine.NoRoute(NoRoutesHandler)
	engine.Use(gin.Recovery())
	engine.Use(middleware.InjectEnv(env))
	return engine
}

func NoRoutesHandler(c *gin.Context) {
	c.String(http.StatusNotFound, "Please double check your HTTP method, url, etc")
}

func Routes(engine *gin.Engine) *gin.Engine {
	profile.Index(engine)
	return engine
}
