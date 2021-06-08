package app

import (
	"fmt"

	"github.com/xiaojiaoyu100/profiler/collector/config/serverconfig"

	"github.com/xiaojiaoyu100/profiler/collector/config/ossconfig"

	"errors"
	"sync"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	log "github.com/sirupsen/logrus"
	aliacm "github.com/xiaojiaoyu100/aliyun-acm/v2"
	"github.com/xiaojiaoyu100/aliyun-acm/v2/info"
	"github.com/xiaojiaoyu100/aliyun-acm/v2/observer"
	"github.com/xiaojiaoyu100/profiler/collector/config/influxdbconfig"
	"github.com/xiaojiaoyu100/profiler/collector/server"
	"go.uber.org/zap"
)

type InfluxDBClient struct {
	lock   sync.RWMutex
	client *influxdb2.Client
}

type OSSClient struct {
	lock   sync.RWMutex
	client *oss.Client
}

type App struct {
	acmOption    *ACMOption
	acmClient    *aliacm.Diamond
	buildOption  *BuildOption
	logger       *zap.Logger
	httpServer   *server.HttpServer
	ossClient    *OSSClient
	influxClient *InfluxDBClient
}

type Setter func(app *App) error

func WithACMCOption(option *ACMOption) Setter {
	return func(app *App) error {
		client, err := aliacm.New(aliacm.WithAcm(option.Addr, option.Tenant, option.AccessKey, option.SecretKey),
			aliacm.WithKms(option.KmsRegionID, option.KmsAccessKey, option.KmsSecretKey))
		if err != nil {
			return fmt.Errorf("fail to create a acm aclient: %w", err)
		}
		app.acmOption = option
		app.acmClient = client
		return nil
	}
}

func WithBuildOption(option *BuildOption) Setter {
	return func(app *App) error {
		app.buildOption = option
		return nil
	}
}

func New(setters ...Setter) (*App, func(), error) {
	a := &App{}
	for _, setter := range setters {
		if err := setter(a); err != nil {
			return nil, nil, err
		}
	}
	if a.buildOption == nil {
		return nil, nil, errors.New("no build option provided")
	}
	cleanup := func() {}
	return a, cleanup, nil
}

func (a *App) Init() error {
	if err := a.initACMClient(); err != nil {
		return err
	}
	return nil
}

func (a *App) Run() error {
	if err := a.httpServer.Run(); err != nil {
		return err
	}
	return nil
}

func (a *App) ACMGroup() string {
	return a.acmOption.Group
}

func (a *App) Logger() *zap.Logger {
	return a.logger
}

func (a *App) OSSClient() *oss.Client {
	a.ossClient.lock.RLock()
	defer a.ossClient.lock.RUnlock()
	return a.ossClient.client
}

func (a *App) SetOSSClient(c *OSSClient) {
	a.ossClient.lock.Lock()
	defer a.ossClient.lock.Unlock()
	a.ossClient = c
}

func (a *App) InfluxDBClient() *influxdb2.Client {
	a.influxClient.lock.RLock()
	defer a.influxClient.lock.RUnlock()
	return a.influxClient.client
}

func (a *App) SetInfluxDBClient(c *InfluxDBClient) {
	a.influxClient.lock.Lock()
	defer a.influxClient.lock.Unlock()
	a.influxClient = c
}

func (a *App) registerObserverList() error {
	var err error
	var create = func(h observer.Handler, ii ...info.Info) {
		if err != nil {
			return
		}
		o, err := observer.New(
			observer.WithInfo(ii...),
			observer.WithHandler(h),
		)
		if err != nil {
			err = fmt.Errorf("observer new error:info:%+v err:%s", ii, err)
			return
		}
		a.acmClient.Register(o)
	}

	create(initHttpServer(a), info.Info{Group: a.acmOption.Group, DataID: serverconfig.DataID})
	create(initOSSClient(a), info.Info{Group: a.acmOption.Group, DataID: ossconfig.DataID})
	create(initInfluxDBClient(a), info.Info{Group: a.acmOption.Group, DataID: influxdbconfig.DataID})

	if err != nil {
		log.Warning("create observer failed: %s", err)
		return err
	}
	return nil
}

func (a *App) initACMClient() error {
	if err := a.registerObserverList(); err != nil {
		return err
	}
	a.acmClient.NotifyAll()
	return nil
}
