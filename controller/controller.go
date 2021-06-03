package controller

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
)

func Index(engine *gin.Engine) {
	engine.POST("/v1/profile",  Profile)
}


type ProfileReq struct {
	Service string `json:"service"`
	ServiceVersion string `json:"serviceVersion"`
	GoVersion string `json:"goVersion"`
	ProfileType string `json:"profileType"`
	Profile json.RawMessage `json:"profile"`
	SendTime int64 `json:"sendTime"`
}

func Profile(c *gin.Context) {
	var p ProfileReq
	if err := c.BindJSON(&p); err != nil {
		return
	}
}
