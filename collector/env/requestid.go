package env

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	RequestIdKey = "X-Request-Id"
)

func GetRequestId(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}
	requestId, ok := ctx.Value(RequestIdKey).(string)
	if !ok {
		return ""
	}
	return requestId
}

func (l *Logger) WithRequestId(ctx *gin.Context) *zap.Logger {
	return l.Logger.With(zap.String(RequestIdKey, GetRequestId(ctx)))
}
