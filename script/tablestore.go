package script

import (
	"github.com/aliyun/aliyun-tablestore-go-sdk/v5/tablestore"
	"github.com/golang/protobuf/proto"
	"github.com/xiaojiaoyu100/profiler/collector/env"
)

func InitOTS(env *env.Env) {
	logger := env.Logger()

	logger.Info("begin to create table")

	client := env.TablestoreClient().Client
	tbl := env.TablestoreClient().TableName

	createTableRequest := new(tablestore.CreateTableRequest)

	tableMeta := new(tablestore.TableMeta)
	tableMeta.TableName = tbl
	tableMeta.AddPrimaryKeyColumn("profile_id", tablestore.PrimaryKeyType_STRING)
	tableOption := new(tablestore.TableOption)
	tableOption.TimeToAlive = -1
	tableOption.MaxVersion = 1
	reservedThroughput := new(tablestore.ReservedThroughput)
	reservedThroughput.Readcap = 0
	reservedThroughput.Writecap = 0
	createTableRequest.TableMeta = tableMeta
	createTableRequest.TableOption = tableOption
	createTableRequest.ReservedThroughput = reservedThroughput

	_, err := client.CreateTable(createTableRequest)
	if err != nil {
		logger.Info("fail to create table")
		return
	}
	logger.Info("create table successfully")

	idx := tbl + "_idx"
	logger.Info("begin to create index")

	request := &tablestore.CreateSearchIndexRequest{}
	request.TableName = tbl
	request.IndexName = idx

	schemas := []*tablestore.FieldSchema{
		{
			FieldName:        proto.String("profile_type"),
			FieldType:        tablestore.FieldType_KEYWORD,
			Index:            proto.Bool(true),
			EnableSortAndAgg: proto.Bool(true),
		},
		{
			FieldName:        proto.String("create_time"),
			FieldType:        tablestore.FieldType_LONG,
			Index:            proto.Bool(true),
			EnableSortAndAgg: proto.Bool(true),
		},
		{
			FieldName:        proto.String("service"),
			FieldType:        tablestore.FieldType_KEYWORD,
			Index:            proto.Bool(true),
			EnableSortAndAgg: proto.Bool(true),
		},
		{
			FieldName:        proto.String("ip"),
			FieldType:        tablestore.FieldType_KEYWORD,
			Index:            proto.Bool(true),
			EnableSortAndAgg: proto.Bool(true),
		},
		{
			FieldName:        proto.String("host"),
			FieldType:        tablestore.FieldType_KEYWORD,
			Index:            proto.Bool(true),
			EnableSortAndAgg: proto.Bool(true),
		},
	}
	request.IndexSchema = &tablestore.IndexSchema{
		FieldSchemas: schemas,
	}
	_, err = client.CreateSearchIndex(request)
	if err != nil {
		logger.Info("fail to create index")
		return
	}
	logger.Info("create index successfully")
}
