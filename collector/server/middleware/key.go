package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/xiaojiaoyu100/profiler/collector/env"
)

const (
	envKey string = "envKey"
)

func Env(c *gin.Context) *env.Env {
	v, ok := c.Get(envKey)
	if !ok {
		return nil
	}
	e, ok := v.(*env.Env)
	if !ok {
		return nil
	}
	return e
}
