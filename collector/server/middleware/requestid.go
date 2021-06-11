package middleware

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	RequestIdKey = "X-Request-Id"
)

func SetRequestId() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestId := ctx.Request.Header.Get(RequestIdKey)
		if requestId == "" {
			requestId = primitive.NewObjectID().Hex()
		}
		ctx.Set(RequestIdKey, requestId)
		ctx.Header(RequestIdKey, requestId)
		ctx.Next()
	}
}
