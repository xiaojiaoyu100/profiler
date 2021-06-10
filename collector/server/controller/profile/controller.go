package profile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aliyun/aliyun-tablestore-go-sdk/v5/tablestore"
	"github.com/gin-gonic/gin"
	"github.com/xiaojiaoyu100/profiler/collector/server/middleware"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type ReceiveProfileReq struct {
	Service        string          `json:"service"`
	ServiceVersion string          `json:"serviceVersion"`
	Host           string          `json:"host"`
	IP             string          `json:"ip"`
	GoVersion      string          `json:"goVersion"`
	ProfileType    string          `json:"profileType"`
	Profile        json.RawMessage `json:"profile"`
	SendTime       int64           `json:"sendTime"`
	CreateTime     int64           `json:"create_time"`
}

func ReceiveProfile(c *gin.Context) {
	logger := middleware.Env(c).Logger

	var req ReceiveProfileReq
	if err := c.BindJSON(&req); err != nil {
		return
	}

	profileID := primitive.NewObjectID().Hex()

	oss := middleware.Env(c).OSSClient()
	bucket, err := oss.Client.Bucket(oss.Bucket)
	if err != nil {
		logger().Debug("new bucket err", zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return

	}

	buf := bytes.NewBuffer(req.Profile)
	if buf.Len() == 0 {
		logger().Debug("no profile provided")
		c.Status(http.StatusOK)
		return
	}

	err = bucket.PutObject(
		UploadPath(oss.PathPrefix, req.Service, req.ProfileType, profileID),
		buf,
	)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	tb := middleware.Env(c).TablestoreClient()

	putRowRequest := new(tablestore.PutRowRequest)
	putRowChange := new(tablestore.PutRowChange)
	putRowChange.TableName = tb.TableName
	putPk := new(tablestore.PrimaryKey)
	putPk.AddPrimaryKeyColumn("profile_id", profileID)
	putRowChange.PrimaryKey = putPk
	putRowChange.AddColumn("service", req.Service)
	putRowChange.AddColumn("service_version", req.ServiceVersion)
	putRowChange.AddColumn("host", req.Host)
	putRowChange.AddColumn("ip", req.IP)
	putRowChange.AddColumn("go_version", req.GoVersion)
	putRowChange.AddColumn("profile_type", req.ProfileType)
	putRowChange.AddColumn("send_time", req.SendTime)
	putRowChange.AddColumn("create_time", req.CreateTime)
	putRowChange.SetCondition(tablestore.RowExistenceExpectation_EXPECT_NOT_EXIST)
	putRowRequest.PutRowChange = putRowChange
	_, err = tb.Client.PutRow(putRowRequest)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	return
}

func UploadPath(pathPrefix, service, profileType, fileName string) string {
	return fmt.Sprintf("%s/%s/%s/%s", pathPrefix, service, profileType, fileName)
}
