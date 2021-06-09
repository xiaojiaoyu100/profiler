package env

import (
	"sync"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

type InfluxDBClient struct {
	lock   sync.RWMutex
	client *influxdb2.Client
}

type OSSClient struct {
	lock   sync.RWMutex
	client *oss.Client
}

type Env struct {
	ossClient    *OSSClient
	influxClient *InfluxDBClient
}

var (
	once sync.Once
	env  *Env
)

func Instance() *Env {
	once.Do(func() {
		env = &Env{
			ossClient:    &OSSClient{},
			influxClient: &InfluxDBClient{},
		}
	})
	return env
}

func (e *Env) SetOSSClient(client *oss.Client) {
	e.ossClient.lock.Lock()
	defer e.ossClient.lock.Unlock()
	e.ossClient.client = client
}

func (e *Env) OSSClient() *oss.Client {
	e.ossClient.lock.RLock()
	defer e.ossClient.lock.RUnlock()
	return e.ossClient.client
}

func (e *Env) SetInfluxDBClient(client *influxdb2.Client) {
	e.influxClient.lock.Lock()
	defer e.influxClient.lock.Unlock()
	e.influxClient.client = client
}

func (e *Env) InfluxDBClient() *influxdb2.Client {
	e.influxClient.lock.RLock()
	defer e.influxClient.lock.RUnlock()
	return e.influxClient.client
}
