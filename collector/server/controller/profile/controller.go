package profile

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/xiaojiaoyu100/profiler/collector/env"
)

type ReceiveProfileReq struct {
	Service        string          `json:"service"`
	ServiceVersion string          `json:"serviceVersion"`
	GoVersion      string          `json:"goVersion"`
	ProfileType    string          `json:"profileType"`
	Profile        json.RawMessage `json:"profile"`
	SendTime       int64           `json:"sendTime"`
}

func ReceiveProfile(c *gin.Context) {
	var p ReceiveProfileReq
	if err := c.BindJSON(&p); err != nil {
		return
	}

	e, _ := c.Get("app")
	env := e.(env.Env)
	env.OSSClient()
	return

}
