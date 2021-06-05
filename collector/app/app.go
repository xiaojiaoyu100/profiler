package app

import (
	"errors"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	aliacm "github.com/xiaojiaoyu100/aliyun-acm/v2"
	"github.com/xiaojiaoyu100/profiler/collector/build"
	"github.com/xiaojiaoyu100/profiler/collector/server"
	"go.uber.org/zap"
)

type App struct {
	acmClient      *aliacm.Diamond
	logger         *zap.Logger
	httpServer     *server.HttpServer
	buildOption    *build.Option
	ossClient      *oss.Client
	influxdbClient *influxdb2.Client
}

type Setter func(app *App) error

func WithHttpServer(server *server.HttpServer) Setter {
	return func(app *App) error {
		app.httpServer = server
		return nil
	}
}

func WithOSSClient(client *oss.Client) Setter {
	return func(app *App) error {
		app.ossClient = client
		return nil
	}
}

func WithLogger(logger *zap.Logger) Setter {
	return func(app *App) error {
		app.logger = logger
		return nil
	}
}

func New(setters ...Setter) (*App, error) {
	a := &App{}
	for _, setter := range setters {
		if err := setter(a); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (a *App) InitBuildOption(option *build.Option) error {
	if option == nil {
		return errors.New("no build option provided")
	}
	a.buildOption = option
	return nil
}

func (a *App) Run() error {
	if err := a.httpServer.Run(); err != nil {
		return err
	}
	return nil
}
