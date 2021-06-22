package profile

import (
	"github.com/gin-gonic/gin"
)

func Index(engine *gin.Engine) {
	engine.POST("/v1/profile", ReceiveProfile)
	engine.POST("/v1/profile/merge", MergeProfile)
}
