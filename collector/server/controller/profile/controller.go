package profile

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/aliyun/aliyun-tablestore-go-sdk/v5/tablestore"
	"github.com/gin-gonic/gin"
	"github.com/xiaojiaoyu100/profiler/collector/server/middleware"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type ReceiveProfileReq struct {
	Service        string `json:"service"`
	ServiceVersion string `json:"service_version"`
	Host           string `json:"host"`
	IP             string `json:"ip"`
	GoVersion      string `json:"go_version"`
	ProfileType    string `json:"profile_type"`
	Profile        string `json:"profile"`
	SendTime       int64  `json:"send_time"`
	CreateTime     int64  `json:"create_time"`
}

var cn = time.FixedZone("GMT", 8*3600)

func ReceiveProfile(c *gin.Context) {
	logger := middleware.Env(c).Logger

	var req ReceiveProfileReq
	if err := c.BindJSON(&req); err != nil {
		return
	}
	req.IP = c.ClientIP()

	profileID := primitive.NewObjectID().Hex()

	oss := middleware.Env(c).OSSClient()
	bucket, err := oss.Client.Bucket(oss.Bucket)
	if err != nil {
		logger().WithRequestId(c).Info("new bucket err",
			zap.String("service", req.Service),
			zap.String("service_version", req.ServiceVersion),
			zap.String("ip", req.IP),
			zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	pf, err := base64.StdEncoding.DecodeString(req.Profile)
	if err != nil {
		logger().WithRequestId(c).Info("fail to decode profile",
			zap.String("service", req.Service),
			zap.String("service_version", req.ServiceVersion),
			zap.String("ip", req.IP),
			zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	buf := bytes.NewBuffer(pf)
	if buf.Len() == 0 {
		logger().WithRequestId(c).Info("no profile provided",
			zap.String("service", req.Service),
			zap.String("service_version", req.ServiceVersion),
			zap.String("ip", req.IP),
		)
		c.Status(http.StatusOK)
		return
	}

	objectName := UploadPath(oss.PathPrefix, req.Service, req.ProfileType, profileID)

	err = bucket.PutObject(objectName, buf)
	if err != nil {
		logger().WithRequestId(c).Info("fail to upload",
			zap.String("service", req.Service),
			zap.String("service_version", req.ServiceVersion),
			zap.String("ip", req.IP),
			zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	_, err = bucket.GetObjectDetailedMeta(objectName)
	if err != nil {
		logger().WithRequestId(c).Info("fail to query object",
			zap.String("service", req.Service),
			zap.String("service_version", req.ServiceVersion),
			zap.String("ip", req.IP),
			zap.Error(err))
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
	putRowChange.AddColumn("object_name", objectName)
	putRowChange.SetCondition(tablestore.RowExistenceExpectation_EXPECT_NOT_EXIST)
	putRowRequest.PutRowChange = putRowChange
	_, err = tb.Client.PutRow(putRowRequest)
	if err != nil {
		logger().WithRequestId(c).Info("fail to insert a row",
			zap.String("service", req.Service),
			zap.String("service_version", req.ServiceVersion),
			zap.String("ip", req.IP),
			zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
}

func UploadPath(pathPrefix, service, profileType, fileName string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		pathPrefix,
		time.Now().In(cn).Format("2006-01-02"),
		service,
		profileType,
		fileName)
}
