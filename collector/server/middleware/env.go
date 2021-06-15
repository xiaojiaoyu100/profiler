package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/xiaojiaoyu100/profiler/collector/env"
)

func InjectEnv(env *env.Env) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(envKey, env)
		c.Next()
	}
}
