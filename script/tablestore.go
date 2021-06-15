package script

import (
	"github.com/aliyun/aliyun-tablestore-go-sdk/v5/tablestore"
	"github.com/xiaojiaoyu100/profiler/collector/env"
	"go.uber.org/zap"
	"golang.org/x/tools/go/ssa/interp/testdata/src/fmt"
)

func InitOTS(env *env.Env) {
	createTableReq := tablestore.CreateTableRequest{
		TableMeta:          nil,
		TableOption:        nil,
		ReservedThroughput: nil,
		StreamSpec:         nil,
		IndexMetas:         nil,
	}
	_, err := env.TablestoreClient().Client.CreateTable(&createTableReq)
	if err != nil {
		env.Logger().Info(fmt.Sprintf("fail to create table %s", env.TablestoreClient().TableName), zap.Error(err))
		return
	}
}
