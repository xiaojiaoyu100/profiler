package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/xiaojiaoyu100/profiler/collector/env"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SetRequestId() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestId := ctx.Request.Header.Get(env.RequestIdKey)
		if requestId == "" {
			requestId = primitive.NewObjectID().Hex()
		}
		ctx.Set(env.RequestIdKey, requestId)
		ctx.Header(env.RequestIdKey, requestId)
		ctx.Next()
	}
}
