package profile

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"time"

	"github.com/aliyun/aliyun-tablestore-go-sdk/v5/tablestore"
	"github.com/aliyun/aliyun-tablestore-go-sdk/v5/tablestore/search"
	"github.com/cavaliercoder/grab"
	"github.com/gin-gonic/gin"
	gprofile "github.com/google/pprof/profile"
	"github.com/xiaojiaoyu100/profiler/collector/env"
	"github.com/xiaojiaoyu100/profiler/collector/server/middleware"
	"github.com/xiaojiaoyu100/profiler/collector/server/model/profilemodel"
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

	meta, err := bucket.GetObjectDetailedMeta(objectName)
	if err != nil {
		logger().WithRequestId(c).Info("fail to query object",
			zap.String("service", req.Service),
			zap.String("service_version", req.ServiceVersion),
			zap.String("ip", req.IP),
			zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	size, err := strconv.ParseInt(meta.Get("Content-Length"), 10, 64)
	if err != nil {
		logger().WithRequestId(c).Info("fail to parse int64",
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
	putPk.AddPrimaryKeyColumn(profilemodel.ProfileId, profileID)
	putRowChange.PrimaryKey = putPk
	putRowChange.AddColumn(profilemodel.Service, req.Service)
	putRowChange.AddColumn(profilemodel.ServiceVersion, req.ServiceVersion)
	putRowChange.AddColumn(profilemodel.Host, req.Host)
	putRowChange.AddColumn(profilemodel.IP, req.IP)
	putRowChange.AddColumn(profilemodel.GoVersion, req.GoVersion)
	putRowChange.AddColumn(profilemodel.ProfileType, req.ProfileType)
	putRowChange.AddColumn(profilemodel.SendTime, req.SendTime)
	putRowChange.AddColumn(profilemodel.CreateTime, req.CreateTime)
	putRowChange.AddColumn(profilemodel.ObjectName, objectName)
	putRowChange.AddColumn(profilemodel.Size, size)
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

type MergeProfileReq struct {
	Host        string `json:"host"`
	Ip          string `json:"ip"`
	Service     string `json:"service"`
	ProfileType string `json:"profile_type"`
	StartTime   int64  `json:"start_time"`
	EndTime     int64  `json:"end_time"`
}

type mergeProfileDetail struct {
	Url          string `json:"url"`
	ProfileCount int    `json:"profile_count"`
}

func MergeProfile(c *gin.Context) {
	logger := middleware.Env(c).Logger
	var req MergeProfileReq
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	tb := middleware.Env(c).TablestoreClient()
	ossClient := middleware.Env(c).OSSClient()

	bucket, err := ossClient.Client.Bucket(ossClient.Bucket)
	if err != nil {
		logger().WithRequestId(c).Info("new bucket err",
			zap.Reflect("req", req),
			zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	profileModelList, err := getProfileModelList(tb, req)
	if err != nil {
		logger().WithRequestId(c).Info("list profile err",
			zap.Reflect("req", req),
			zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var reqs []*grab.Request
	for _, profileModel := range profileModelList {
		url := downloadPath(ossClient.Bucket, ossClient.EndPoint, profileModel.ObjectName)
		req, err := grab.NewRequest("", url)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		req.NoStore = true
		reqs = append(reqs, req)
	}

	var profileList []*gprofile.Profile

	responses := grab.NewClient().DoBatch(runtime.NumCPU()*2, reqs...)
Loop:
	for i := 0; i < len(reqs); i++ {
		select {
		case ossResp := <-responses:
			if ossResp == nil {
				break Loop
			}
			if ossResp.HTTPResponse == nil {
				break Loop
			}

			func() {
				b, err := ossResp.Bytes()
				if err != nil {
					logger().WithRequestId(c).Info("oss resp bytes",
						zap.Reflect("req", req),
						zap.Error(err))
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}

				r, err := ossResp.Open()
				if err != nil {
					logger().WithRequestId(c).Info("oss resp open",
						zap.Reflect("req", req),
						zap.Error(err))
					return
				}
				defer r.Close()

				pprofProfile, err := gprofile.ParseData(b)
				if err != nil {
					logger().WithRequestId(c).Info("profile parse err",
						zap.Reflect("req", req),
						zap.Error(err))
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}
				profileList = append(profileList, pprofProfile)
			}()
		}
	}

	mergeProfile, err := gprofile.Merge(profileList)
	if err != nil {
		logger().WithRequestId(c).Info("profile merge err",
			zap.Reflect("req", req),
			zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)
	err = mergeProfile.Write(buf)
	if err != nil {
		logger().WithRequestId(c).Info("profile write err",
			zap.Reflect("req", req),
			zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	newProfileID := primitive.NewObjectID().Hex()
	objectName := UploadPath(ossClient.PathPrefix, req.Service, req.ProfileType, newProfileID)

	err = bucket.PutObject(objectName, buf)
	if err != nil {
		logger().WithRequestId(c).Info("fail to upload",
			zap.Reflect("req", req),
			zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var resp = mergeProfileDetail{
		Url:          downloadPath(ossClient.Bucket, ossClient.EndPoint, objectName),
		ProfileCount: len(profileList),
	}
	c.AbortWithStatusJSON(http.StatusOK, resp)
}

func downloadPath(bucket, endPoint, objectName string) string {
	return fmt.Sprintf("https://%s.%s/%s",
		bucket,
		endPoint,
		objectName)
}

const limit = int32(100)

// getProfileModelList batch get profile model from tableStore
func getProfileModelList(tb *env.TablestoreClient, req MergeProfileReq) ([]*profilemodel.Model, error) {
	result, total, err := searchProfile(tb, req, 0)
	if err != nil {
		return nil, err
	}

	// Error - restrict total profile of 2000
	if total > int64(20*limit) {
		return nil, errors.New("out of total, 2000")
	}

	if len(result) < int(limit) {
		return result, nil
	}

	var offset = limit
	var totalSize int64
	for {
		list, _, err := searchProfile(tb, req, offset)
		if err != nil {
			return nil, err
		}

		for _, v := range list {
			totalSize += v.Size
		}

		// Error - restrict total file size of 100Mb
		if totalSize > 100*1024*1024*1024 {
			return nil, errors.New("out of size, 100MB")
		}

		result = append(result, list...)

		if len(list) < int(limit) {
			break
		}
		offset += limit
	}

	return result, nil
}

// searchProfile search profile model from tableStore with request body
func searchProfile(tb *env.TablestoreClient, req MergeProfileReq, offset int32) ([]*profilemodel.Model, int64, error) {
	if len(req.Service) > 0 {

	}
	if len(req.Host) == 0 {
		return nil, 0, errors.New("lack of host")
	}
	if len(req.ProfileType) == 0 {
		return nil, 0, errors.New("lack of profile_type")
	}
	if req.StartTime > req.EndTime {
		return nil, 0, errors.New("time range wrong")
	}

	boolQuery := search.BoolQuery{
		MustQueries: []search.Query{
			&search.TermQuery{
				FieldName: profilemodel.ProfileType,
				Term:      req.ProfileType,
			},
			&search.RangeQuery{
				FieldName:    profilemodel.CreateTime,
				From:         req.StartTime,
				To:           req.EndTime,
				IncludeLower: true,
				IncludeUpper: true,
			},
		},
	}

	if len(req.Service) > 0 {
		boolQuery.MustQueries = append(boolQuery.MustQueries,
			&search.TermQuery{
				FieldName: profilemodel.Service,
				Term:      req.Service,
			},
		)
	}

	if len(req.Ip) > 0 {
		boolQuery.MustQueries = append(boolQuery.MustQueries,
			&search.TermQuery{
				FieldName: profilemodel.IP,
				Term:      req.Ip,
			},
		)
	}

	if len(req.Host) > 0 {
		boolQuery.MustQueries = append(boolQuery.MustQueries,
			&search.TermQuery{
				FieldName: profilemodel.Host,
				Term:      req.Host,
			},
		)
	}

	sort := search.Sort{
		Sorters: []search.Sorter{
			&search.FieldSort{
				FieldName: profilemodel.CreateTime,
				Order:     search.SortOrder_ASC.Enum(),
			},
		},
	}

	searchQuery := search.NewSearchQuery()
	searchQuery.SetLimit(limit)
	searchQuery.SetQuery(&boolQuery)
	searchQuery.SetSort(&sort)
	if offset > 0 {
		searchQuery.SetOffset(offset)
	} else {
		searchQuery.SetGetTotalCount(true)
	}

	searchRequest := new(tablestore.SearchRequest)
	searchRequest.SetTableName(tb.TableName)
	searchRequest.SetIndexName(tb.TableName + "_idx")
	searchRequest.SetColumnsToGet(&tablestore.ColumnsToGet{ReturnAll: true})
	searchRequest.SetSearchQuery(searchQuery)

	getRangeResp, err := tb.Client.Search(searchRequest)
	if err != nil {
		return nil, 0, err
	}

	var result []*profilemodel.Model
	for _, row := range getRangeResp.Rows {
		profileModel := unMarshalProfileRow(row)
		result = append(result, profileModel)
	}
	return result, getRangeResp.TotalCount, nil
}

// unMarshalProfileRow unmarshal information from tableStore row
func unMarshalProfileRow(row *tablestore.Row) *profilemodel.Model {
	result := new(profilemodel.Model)
	ve := reflect.ValueOf(result).Elem()
	te := reflect.TypeOf(result).Elem()
	ret := make(map[string]reflect.Value)
	for i := 0; i < te.NumField(); i++ {
		tag := te.Field(i).Tag.Get("ots")
		ret[tag] = ve.Field(i)
	}
	for _, column := range row.Columns {
		v, ok := ret[column.ColumnName]
		if !ok {
			continue
		}
		if !v.IsValid() || !v.CanSet() {
			continue
		}
		switch v.Kind() {
		case reflect.String:
			v.SetString(column.Value.(string))
		case reflect.Int64:
			v.SetInt(column.Value.(int64))
		}
	}
	return result
}
