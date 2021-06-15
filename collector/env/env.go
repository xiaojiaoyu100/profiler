package env

import (
	"sync"

	"go.uber.org/zap"

	"github.com/aliyun/aliyun-tablestore-go-sdk/v5/tablestore"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

type InfluxDBClient struct {
	client *influxdb2.Client
}

type OSSClient struct {
	Bucket     string
	PathPrefix string
	Client     *oss.Client
}

type TablestoreClient struct {
	TableName string
	Client    *tablestore.TableStoreClient
}

type Logger struct {
	*zap.Logger
}

type Env struct {
	logger           *Logger
	ossClient        *OSSClient
	tableStoreClient *TablestoreClient
	influxClient     *InfluxDBClient
}

var (
	once sync.Once
	env  *Env
)

func Instance() *Env {
	once.Do(func() {
		env = &Env{
			ossClient:        &OSSClient{},
			influxClient:     &InfluxDBClient{},
			tableStoreClient: &TablestoreClient{},
		}
	})
	return env
}

func (e *Env) SetOSSClient(client *OSSClient) {
	e.ossClient = client
}

func (e *Env) SetLogger(logger *Logger) {
	e.logger = logger
}

func (e *Env) Logger() *Logger {
	return e.logger
}

func (e *Env) OSSClient() *OSSClient {
	return e.ossClient
}

func (e *Env) SetInfluxDBClient(client *influxdb2.Client) {
	e.influxClient.client = client
}

func (e *Env) InfluxDBClient() *influxdb2.Client {
	return e.influxClient.client
}

func (e *Env) SetTablestoreClient(client *TablestoreClient) {
	e.tableStoreClient = client
}

func (e *Env) TablestoreClient() *TablestoreClient {
	return e.tableStoreClient
}
