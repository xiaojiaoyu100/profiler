package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/xiaojiaoyu100/profiler/collector/env"

	"github.com/xiaojiaoyu100/profiler/collector/server/engine"

	"github.com/xiaojiaoyu100/profiler/collector/config/serverconfig"
	"github.com/xiaojiaoyu100/profiler/collector/server"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/xiaojiaoyu100/aliyun-acm/v2/config"
	"github.com/xiaojiaoyu100/aliyun-acm/v2/info"
	"github.com/xiaojiaoyu100/aliyun-acm/v2/observer"
	"github.com/xiaojiaoyu100/profiler/collector/config/influxdbconfig"
	"github.com/xiaojiaoyu100/profiler/collector/config/ossconfig"
	"go.uber.org/zap"
)

func initHttpServer(a *App) observer.Handler {
	return func(coll map[info.Info]*config.Config) {
		a.Logger().Debug("about to start server ")

		dataID := serverconfig.DataID

		cc, ok := coll[info.Info{
			Group:  a.ACMGroup(),
			DataID: dataID,
		}]
		if !ok {
			a.Logger().Debug(fmt.Sprintf("fail to load, group = %s, dataID = %s", a.ACMGroup(), dataID))
			return
		}
		c := &serverconfig.Config{}
		if err := json.Unmarshal(cc.Content, c); err != nil {
			a.Logger().Warn(fmt.Sprintf("fail to unmarshal, group = %s, dataID = %s", a.ACMGroup(), dataID), zap.Error(err))
			return
		}

		en := engine.Routes(engine.Engine(env.Instance()))

		httpServer, err := server.New(
			server.WithLogger(a.Logger()),
			server.WithOption(&server.Option{
				Addr:            c.Addr,
				ShutdownTimeout: time.Duration(c.ShutdownTimeout) * time.Second,
			}),
			server.WithEngine(en),
		)
		if err != nil {
			a.Logger().Warn(fmt.Sprintf("fail to create a http server, group = %s, dataID = %s", a.ACMGroup(), dataID), zap.Error(err))
			return
		}

		a.guardHttpServer.Lock()
		if a.httpServer.Running() {
			a.httpServer.Close()
		}
		httpServer.Run()
		a.httpServer = httpServer
		a.guardHttpServer.Unlock()
	}
}

func initInfluxDBClient(a *App) observer.Handler {
	return func(coll map[info.Info]*config.Config) {
		dataID := influxdbconfig.DataID

		cc, ok := coll[info.Info{
			Group:  a.ACMGroup(),
			DataID: dataID,
		}]
		if !ok {
			a.Logger().Debug(fmt.Sprintf("fail to load, group = %s, dataID = %s", a.ACMGroup(), dataID))
			return
		}
		c := &influxdbconfig.InfluxDBConfig{}
		if err := json.Unmarshal(cc.Content, c); err != nil {
			a.Logger().Warn(fmt.Sprintf("fail to unmarshal, group = %s, dataID = %s", a.ACMGroup(), dataID), zap.Error(err))
			return
		}
		client := influxdb2.NewClient(c.ServerURL, c.AuthToken)

		env.Instance().SetInfluxDBClient(&client)
	}
}

func initOSSClient(a *App) observer.Handler {
	return func(coll map[info.Info]*config.Config) {
		dataID := ossconfig.DataID
		cc, ok := coll[info.Info{
			Group:  a.ACMGroup(),
			DataID: dataID,
		}]
		if !ok {
			a.Logger().Debug(fmt.Sprintf("fail to load, group = %s, dataID = %s", a.ACMGroup(), dataID))
			return
		}
		c := &ossconfig.Config{}
		if err := json.Unmarshal(cc.Content, c); err != nil {
			a.Logger().Warn(fmt.Sprintf("fail to unmarshal, group = %s, dataID = %s", a.ACMGroup(), dataID), zap.Error(err))
			return
		}
		client, err := oss.New(c.Endpoint, c.AccessKeyID, c.AccessKeySecret)
		if err != nil {
			a.Logger().Warn(fmt.Sprintf("fail to create a oss client, group = %s, dataID = %s", a.ACMGroup(), dataID), zap.Error(err))
			return
		}
		env.Instance().SetOSSClient(client)
	}
}
