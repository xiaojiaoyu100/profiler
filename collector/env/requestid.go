package env

import (
	"github.com/gin-gonic/gin"
	"github.com/xiaojiaoyu100/profiler/collector/server/middleware"
	"go.uber.org/zap"
)

func GetRequestId(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}
	requestId, ok := ctx.Value(middleware.RequestIdKey).(string)
	if !ok {
		return ""
	}
	return requestId
}

func (l *Logger) WithRequestId(ctx *gin.Context) *zap.Logger {
	return l.Logger.With(zap.String(middleware.RequestIdKey, GetRequestId(ctx)))
}
